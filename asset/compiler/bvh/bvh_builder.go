package bvh

import (
	"math"
	"time"

	"github.com/achilleasa/go-pathtrace/asset/scene"
	"github.com/achilleasa/go-pathtrace/log"
	"github.com/achilleasa/go-pathtrace/types"
)

type Axis uint8

const (
	XAxis Axis = iota
	YAxis
	ZAxis

	// The BVH builder will not attempt to calculate split candidates
	// if the node bbox along an axis is less than this threshold.
	minSideLength float32 = 1e-3

	// If the split step (calculated as side length / (1024 * depth+1))
	// is less than this threshold the BVH builder will not evaluate
	// split candidates.
	minSplitStep float32 = 1e-5
)

var (
	// A split scoring strategy that uses the surface area heuristic (SAH).
	SurfaceAreaHeuristic = surfaceAreaHeuristic{}
)

// The BoundedVolume interface is implemented by all meshes/primitives that can
// be partitioned by the bvh builder.
type BoundedVolume interface {
	BBox() [2]types.Vec3
	Center() types.Vec3
}

// A callback that is called whenever the BVH builder creates a new leaf.
type LeafCallback func(leaf *scene.BvhNode, itemList []BoundedVolume)

// A split scoring strategy.
type ScoreStrategy interface {
	// Calculate a score for splitting workList at splitPoint along a particular Axis.
	ScoreSplit(workList []BoundedVolume, splitAxis Axis, splitPoint float32) (leftCount, rightCount int, score float32)

	// Calculate a score for all items in workList.
	ScorePartition(workList []BoundedVolume) (score float32)
}

type splitScore struct {
	axis       Axis
	splitPoint float32

	leftCount, rightCount int
	score                 float32
}

type stats struct {
	partitionedItems int
	totalItems       int
	nodes            int
	leafs            int
	maxDepth         int
}

type builder struct {
	logger log.Logger

	// Bvh nodes stored as a contiguous list
	nodes []scene.BvhNode

	// A callback invoked to set up BVH leafs depending on the type of
	// partitioned bounding volume
	leafCb LeafCallback

	// The minimum number of items that are required for creating a leaf.
	minLeafItems int

	// A channel for receiving score results.
	scoreChan chan splitScore

	// The split scoring strategy to use.
	scoreStrategy ScoreStrategy

	// Stats
	stats stats
}

// Construct a BVH from a set of bounded volumes.
//
// The builder uses SAH for scoring splits:
// score = num_polygons * node bbox face area.
//
// The minLeafItems param should be used to specified the minimum number of
// items that can form a leaf. The BVH builder will automatically generate leafs
// if the incoming work length is <= minLeafItems.
func Build(workList []BoundedVolume, minLeafItems int, leafCb LeafCallback, scoreStrategy ScoreStrategy) []scene.BvhNode {
	b := &builder{
		logger:        log.New("builder"),
		nodes:         make([]scene.BvhNode, 0),
		leafCb:        leafCb,
		minLeafItems:  minLeafItems,
		scoreChan:     make(chan splitScore, 0),
		scoreStrategy: scoreStrategy,
		stats: stats{
			totalItems: len(workList),
		},
	}

	start := time.Now()
	b.partition(workList, 0)
	b.logger.Debugf(
		"BVH tree build time: %d ms, maxDepth: %d, nodes: %d, leafs: %d\n",
		time.Since(start).Nanoseconds()/1e6,
		b.stats.maxDepth, b.stats.nodes, b.stats.leafs,
	)
	return b.nodes
}

// Partition worklist and return node index.
func (b *builder) partition(workList []BoundedVolume, depth int) uint32 {
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
	var bestScore float32 = b.scoreStrategy.ScorePartition(workList)
	var bestSplit *splitScore = nil

	// Try partioning along each axis and select the split with best score
	pendingScores := 0

	// Run axis split tests in parallel
	side := node.Max.Sub(node.Min)
	for axis := XAxis; axis <= ZAxis; axis++ {
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
			pendingScores++
			go func(axis Axis, splitPoint float32) {
				lCount, rCount, score := b.scoreStrategy.ScoreSplit(workList, axis, splitPoint)
				b.scoreChan <- splitScore{
					axis:       axis,
					splitPoint: splitPoint,

					leftCount:  lCount,
					rightCount: rCount,
					score:      score,
				}
			}(axis, splitPoint)
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

// Setup the given node item as a leaf node containing all items in the work list.
// Returns the index to the node in the bvh node array.
func (b *builder) createLeaf(node *scene.BvhNode, workList []BoundedVolume) uint32 {
	b.leafCb(node, workList)

	// append node to list
	nodeIndex := len(b.nodes)
	b.nodes = append(b.nodes, *node)

	// update stats
	b.stats.leafs++
	b.stats.partitionedItems += len(workList)

	return uint32(nodeIndex)
}

// A score implementation that uses surface area heuristic for calculating split scores.
type surfaceAreaHeuristic struct{}

// Score a BVH split based on the surface area heuristic. The SAH calculates
// the split score using the formula (lower score is better):
//
// left count * left BBOX area + rightCount * right BBOX area.
//
// SAH avoids splits that generate empty partitions by assigning the worst
// possible score (MaxFloat32) when it enounters such cases.
func (h surfaceAreaHeuristic) ScoreSplit(workList []BoundedVolume, axis Axis, splitPoint float32) (leftCount, rightCount int, score float32) {
	lmin := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	rmin := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	lmax := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	rmax := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}

	leftCount = 0
	rightCount = 0
	for _, item := range workList {
		center := item.Center()
		itemBBox := item.BBox()
		if center[axis] < splitPoint {
			leftCount++
			lmin = types.MinVec3(lmin, itemBBox[0])
			lmax = types.MaxVec3(lmax, itemBBox[1])
		} else {
			rightCount++
			rmin = types.MinVec3(rmin, itemBBox[0])
			rmax = types.MaxVec3(rmax, itemBBox[1])
		}
	}

	// Make sure that we don't generate empty partitions
	if leftCount == 0 || rightCount == 0 {
		return leftCount, rightCount, math.MaxFloat32
	}

	lside := lmax.Sub(lmin)
	rside := rmax.Sub(rmin)
	score = (float32(leftCount) * (lside[0]*lside[1] + lside[1]*lside[2] + lside[0]*lside[2])) +
		(float32(rightCount) * (rside[0]*rside[1] + rside[1]*rside[2] + rside[0]*rside[2]))

	return leftCount, rightCount, score
}

// Calculate score for a partitioned workList using formula:
// count * BBOX area
//
// If the workList is empty, then this method returns the worst possible
// score (MaxFloat32).
func (h surfaceAreaHeuristic) ScorePartition(workList []BoundedVolume) (score float32) {
	if len(workList) == 0 {
		return math.MaxFloat32
	}

	min := types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
	max := types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}

	for _, item := range workList {
		itemBBox := item.BBox()
		min = types.MinVec3(min, itemBBox[0])
		max = types.MaxVec3(max, itemBBox[1])
	}

	side := max.Sub(min)
	return float32(len(workList)) * (side[0]*side[1] + side[1]*side[2] + side[0]*side[2])
}
