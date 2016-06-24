package compiler

import (
	"math"
	"time"

	"github.com/achilleasa/go-pathtrace/log"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/types"
)

const (
	// The BVH builder will not attempt to calculate split candidates
	// if the node bbox along an axis is less than this threshold.
	minSideLength float32 = 1e-3

	// If the split step (calculated as side length / (1024 * depth+1))
	// is less than this threshold the BVH builder will not evaluate
	// split candidates.
	minSplitStep float32 = 1e-5
)

// The BoundedVolume interface is implemented by all meshes/primitives that can
// be partitioned by the bvh builder.
type BoundedVolume interface {
	BBox() [2]types.Vec3
	Center() types.Vec3
}

// A callback that is called whenever the BVH builder creates a new leaf.
type BvhLeafCallback func(leaf *scene.BvhNode, itemList []BoundedVolume)

type bvhSplitCandidate struct {
	axis                  int
	splitPoint            float32
	leftCount, rightCount int
	score                 float32
}

type bvhStats struct {
	partitionedItems int
	totalItems       int
	nodes            int
	leafs            int
	maxDepth         int
}

type bvhBuilder struct {
	logger log.Logger

	// Bvh nodes stored as a contiguous list
	nodes []scene.BvhNode

	// A callback invoked to set up BVH leafs depending on the type of
	// partitioned bounding volume
	leafCb BvhLeafCallback

	// The minimum number of items that are required for creating a leaf.
	minLeafItems int

	// Score result chan
	scoreChan chan bvhSplitCandidate

	// Stats
	stats bvhStats
}

// Construct a BVH from a set of bounded volumes.
//
// The builder uses SAH for scoring splits:
// score = num_polygons * node bbox face area.
//
// The minLeafItems param should be used to specified the minimum number of
// items that can form a leaf. The BVH builder will automatically generate leafs
// if the incoming work length is <= minLeafItems.
func BuildBVH(workList []BoundedVolume, minLeafItems int, leafCb BvhLeafCallback) []scene.BvhNode {
	builder := &bvhBuilder{
		logger:       log.New("bvhBuilder"),
		nodes:        make([]scene.BvhNode, 0),
		leafCb:       leafCb,
		minLeafItems: minLeafItems,
		scoreChan:    make(chan bvhSplitCandidate, 0),
		stats: bvhStats{
			totalItems: len(workList),
		},
	}

	start := time.Now()
	builder.partition(workList, 0)
	builder.logger.Debugf(
		"BVH tree build time: %d ms, maxDepth: %d, nodes: %d, leafs: %d\n",
		time.Since(start).Nanoseconds()/1e6,
		builder.stats.maxDepth, builder.stats.nodes, builder.stats.leafs,
	)
	return builder.nodes
}

// Partition worklist and return node index.
func (b *bvhBuilder) partition(workList []BoundedVolume, depth int) uint32 {
	if depth > b.stats.maxDepth {
		b.stats.maxDepth = depth
	}

	node := scene.BvhNode{
		Min: types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32},
		Max: types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32},
	}

	// Calculate bounding box for node
	for _, item := range workList {
		itemBBox := item.BBox()
		node.Min = types.MinVec3(node.Min, itemBBox[0])
		node.Max = types.MaxVec3(node.Max, itemBBox[1])
	}

	// Do we have enough items for partitioning? If not create a leaf
	if len(workList) <= b.minLeafItems {
		return b.createLeaf(&node, workList)
	}

	// Calc current node score
	side := node.Max.Sub(node.Min)
	var bestScore float32 = float32(len(workList)) * (side[0]*side[1] + side[1]*side[2] + side[0]*side[2])
	var bestSplit *bvhSplitCandidate = nil

	// Try partioning along each axis and select the split with best score
	pendingScores := 0

	// Run axis split tests in parallel
	for axis := 0; axis < 3; axis++ {
		// Skip axis if bbox dimension is too small
		if side[axis] < minSideLength {
			continue
		}

		// We want the split steps to become more granular the deeper we go
		splitStep := side[axis] / (1024.0 / float32(depth+1))
		if splitStep < minSplitStep {
			continue
		}

		for splitPoint := node.Min[axis]; splitPoint < node.Max[axis]; splitPoint += splitStep {
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
		return b.createLeaf(&node, workList)
	}

	// split work list into two sets
	leftWorkList := make([]BoundedVolume, bestSplit.leftCount)
	rightWorkList := make([]BoundedVolume, bestSplit.rightCount)
	leftIndex := 0
	rightIndex := 0
	for _, item := range workList {
		center := item.Center()
		if center[bestSplit.axis] < bestSplit.splitPoint {
			leftWorkList[leftIndex] = item
			leftIndex++
		} else {
			rightWorkList[rightIndex] = item
			rightIndex++
		}
	}

	// Add node to list
	nodeIndex := len(b.nodes)
	b.nodes = append(b.nodes, node)
	b.stats.nodes++

	// Partition children and update node indices
	leftNodeIndex := b.partition(leftWorkList, depth+1)
	rightNodeIndex := b.partition(rightWorkList, depth+1)
	b.nodes[nodeIndex].SetChildNodes(leftNodeIndex, rightNodeIndex)

	return uint32(nodeIndex)
}

// Calculate the score for splitting the workList with this split candidate
// and report the result to the supplied channel.
func (c bvhSplitCandidate) Score(workList []BoundedVolume, resChan chan<- bvhSplitCandidate) {
	lmin := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	rmin := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	lmax := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	rmax := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}

	for _, item := range workList {
		center := item.Center()
		itemBBox := item.BBox()
		if center[c.axis] < c.splitPoint {
			c.leftCount++
			lmin = types.MinVec3(lmin, itemBBox[0])
			lmax = types.MaxVec3(lmax, itemBBox[1])
		} else {
			c.rightCount++
			rmin = types.MinVec3(rmin, itemBBox[0])
			rmax = types.MaxVec3(rmax, itemBBox[1])
		}
	}

	// Make sure that we got enough items of each side of the split
	minItemsOnEachSide := 2
	if len(workList) == 2 {
		minItemsOnEachSide = 1
	}
	if c.leftCount < minItemsOnEachSide || c.rightCount < minItemsOnEachSide {
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

// Setup the given node item as a leaf node containing all items in the work list.
// Returns the index to the node in the bvh node array.
func (b *bvhBuilder) createLeaf(node *scene.BvhNode, workList []BoundedVolume) uint32 {
	b.leafCb(node, workList)

	// append node to list
	nodeIndex := len(b.nodes)
	b.nodes = append(b.nodes, *node)

	// update stats
	b.stats.leafs++
	b.stats.partitionedItems += len(workList)

	return uint32(nodeIndex)
}
