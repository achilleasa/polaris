package tools

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/types"
)

type bvhSplitCandidate struct {
	axis                  int
	splitPoint            float32
	leftCount, rightCount int
	score                 float32
}

type bvhBuilder struct {
	logger *log.Logger

	// Bvh nodes stored as a contiguous list suitable for GPUs.
	nodes []scene.BvhNode

	// Partioned primitives contiguous list. Primitives belonging to a BVH
	// leaf are packed sequentially.
	primitives []scene.Primitive

	// The primitive list is pre-allocated as the number of primitives is
	// already known. We need to maintain an offset so we know where to
	// insert the next leaf primitive.
	nextPrimitiveIndex int

	// Score result chan
	scoreChan chan bvhSplitCandidate

	// Stats
	numNodes      int
	numLeafs      int
	numPrimitives int
	maxDepth      int
}

// Construct a BVH from a set of primitives. The builder uses surface area as
// the split heuristic: score = num_polygons * node bbox face area. The
// minNodePrimitives parameter defines the minimum number of primitives for
// partitioning a node. If an incoming work list contains less primitives it
// will be automatically converted into a leaf.
func BuildBVH(workList []*scene.BvhPrimitive, minNodePrimitives int) ([]scene.BvhNode, []scene.Primitive) {
	builder := &bvhBuilder{
		logger: log.New(os.Stderr, "bvhBuilder: ", log.LstdFlags),
		nodes:  make([]scene.BvhNode, 0),
		// All primitives will be eventually partitioned so we can
		// pre-allocate the primitive list now since its size is known
		primitives:    make([]scene.Primitive, len(workList)),
		scoreChan:     make(chan bvhSplitCandidate, 0),
		numPrimitives: len(workList),
	}

	start := time.Now()
	builder.partition(workList, minNodePrimitives, 0)
	builder.logger.Printf(
		"BVH tree build time: %d ms, maxDepth: %d, nodes: %d, leafs: %d\n",
		time.Since(start).Nanoseconds()/1000000,
		builder.maxDepth, builder.numNodes, builder.numLeafs,
	)
	return builder.nodes, builder.primitives
}

// Partition worklist and return node index.
func (b *bvhBuilder) partition(workList []*scene.BvhPrimitive, minNodePrimitives int, depth int) int {
	if depth > b.maxDepth {
		b.maxDepth = depth
	}
	fmt.Printf("partitioned %02d%%\r", int(100.0*float32(b.nextPrimitiveIndex)/float32(b.numPrimitives)))

	node := scene.BvhNode{}
	nmin := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	nmax := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}

	// Calculate bounding box for node
	for _, prim := range workList {
		nmin = types.MinVec3(nmin, prim.Min)
		nmax = types.MaxVec3(nmax, prim.Max)
	}

	// Do we have enough primitives for partitioning? If not create a leaf
	if len(workList) < minNodePrimitives {
		node.Min = nmin.Vec4(float32(-b.nextPrimitiveIndex))
		node.Max = nmax.Vec4(float32(-len(workList)))
		for _, prim := range workList {
			b.primitives[b.nextPrimitiveIndex] = *prim.Primitive
			b.nextPrimitiveIndex++
		}

		// Add to list
		nodeIndex := len(b.nodes)
		b.nodes = append(b.nodes, node)
		b.numLeafs++
		return nodeIndex
	}

	// Calc current node score
	side := nmax.Sub(nmin)
	var bestScore float32 = float32(len(workList)) * (side[0]*side[1] + side[1]*side[2] + side[0]*side[2])
	var bestSplit *bvhSplitCandidate = nil

	// Try partioning along each axis and select the split with best score
	pendingScores := 0

	// Run axis split tests in parallel
	for axis := 0; axis < 3; axis++ {
		// Skip axis if bbox dimension is too small
		if side[axis] < 1e-4 {
			continue
		}

		// We want the split steps to become more granular the deeper we go
		splitStep := side[axis] / (1024.0 / float32(depth+1))

		for splitPoint := nmin[axis]; splitPoint < nmax[axis]; splitPoint += splitStep {
			candidate := bvhSplitCandidate{
				axis:       axis,
				splitPoint: splitPoint,
			}
			pendingScores++
			go candidate.Score(workList, b.scoreChan)
		}
	}

	// Process all scores and pick the best split
	for ; pendingScores > 0; pendingScores-- {
		candidate := <-b.scoreChan
		if candidate.score < bestScore {
			bestScore = candidate.score
			bestSplit = &candidate
		}
	}

	// If we can't find a split that improves the current node score create a leaf
	if bestSplit == nil {
		node.Min = nmin.Vec4(float32(-b.nextPrimitiveIndex))
		node.Max = nmax.Vec4(float32(-len(workList)))
		for _, prim := range workList {
			b.primitives[b.nextPrimitiveIndex] = *prim.Primitive
			b.nextPrimitiveIndex++
		}

		// Add to list
		nodeIndex := len(b.nodes)
		b.nodes = append(b.nodes, node)
		b.numLeafs++
		return nodeIndex
	}

	// split work list into two sets
	leftWorkList := make([]*scene.BvhPrimitive, bestSplit.leftCount)
	rightWorkList := make([]*scene.BvhPrimitive, bestSplit.rightCount)
	leftIndex := 0
	rightIndex := 0
	for _, prim := range workList {
		if prim.Center[bestSplit.axis] < bestSplit.splitPoint {
			leftWorkList[leftIndex] = prim
			leftIndex++
		} else {
			rightWorkList[rightIndex] = prim
			rightIndex++
		}
	}

	// Add node to list
	nodeIndex := len(b.nodes)
	b.nodes = append(b.nodes, node)
	b.numNodes++

	// Partition children and update node indices
	b.nodes[nodeIndex].Min = nmin.Vec4(float32(b.partition(leftWorkList, minNodePrimitives, depth+1)))
	b.nodes[nodeIndex].Max = nmax.Vec4(float32(b.partition(rightWorkList, minNodePrimitives, depth+1)))

	return nodeIndex
}

// Calculate the score for splitting the workList with this split candidate
// and report the result to the supplied channel.
func (c bvhSplitCandidate) Score(workList []*scene.BvhPrimitive, resChan chan<- bvhSplitCandidate) {
	lmin := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	rmin := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	lmax := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	rmax := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}

	for _, prim := range workList {
		if prim.Center[c.axis] < c.splitPoint {
			c.leftCount++
			lmin = types.MinVec3(lmin, prim.Min)
			lmax = types.MaxVec3(lmax, prim.Max)
		} else {
			c.rightCount++
			rmin = types.MinVec3(rmin, prim.Min)
			rmax = types.MaxVec3(rmax, prim.Max)
		}
	}

	// Make sure that we got more than one primitive on each sides
	if c.leftCount <= 1 || c.rightCount <= 1 {
		c.score = math.MaxFloat32
		resChan <- c
		return
	}

	lside := lmax.Sub(lmin)
	rside := rmax.Sub(rmin)
	c.score = (float32(c.leftCount) * (lside[0]*lside[1] + lside[1]*lside[2] + lside[0]*lside[2])) +
		(float32(c.rightCount) * (rside[0]*rside[1] + rside[1]*rside[2] + rside[0]*rside[2]))
	resChan <- c
}
