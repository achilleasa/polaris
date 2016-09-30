package bvh

import (
	"testing"

	"github.com/achilleasa/polaris/scene"
	"github.com/achilleasa/polaris/types"
)

func TestLeafCallback(t *testing.T) {
	type primSpec struct {
		min types.Vec3
		max types.Vec3
	}

	primSpecs := []primSpec{
		{types.Vec3{-2, 0, -2}, types.Vec3{-1, 1, -1}},
		{types.Vec3{1, 0, -2}, types.Vec3{2, 1, -1}},
		{types.Vec3{-2, 0, 1}, types.Vec3{-1, 1, 2}},
		{types.Vec3{1, 0, 1}, types.Vec3{2, 1, 2}},
	}

	itemList := make([]BoundedVolume, len(primSpecs))
	for idx, ps := range primSpecs {
		inst := &scene.ParsedMeshInstance{}
		inst.SetBBox([2]types.Vec3{ps.min, ps.max})
		inst.SetCenter(ps.min.Add(ps.max).Mul(0.5))
		itemList[idx] = inst
	}

	var cbCount = 0
	var expItemListCount = 0
	cb := func(leaf *scene.BvhNode, itemList []BoundedVolume) {
		cbCount++
		if len(itemList) != expItemListCount {
			t.Fatalf("expected leaf callback to be called with %d items; got %d", expItemListCount, len(itemList))
		}
	}

	var expCount = 0

	// Partition each item in a single leaf
	cbCount = 0
	expItemListCount = 1
	treeNodes := Build(itemList, 1, cb, SurfaceAreaHeuristic)

	expCount = 4
	if cbCount != expCount {
		t.Fatalf("expected leaf callback to be called %d times; called %d", expCount, cbCount)
	}
	expCount = 7
	if len(treeNodes) != expCount {
		t.Fatalf("expected bvh tree to have %d nodes; got %d", expCount, len(treeNodes))
	}

	// Partition two items in a single leaf
	cbCount = 0
	expItemListCount = 2
	treeNodes = Build(itemList, 2, cb, SurfaceAreaHeuristic)

	expCount = 2
	if cbCount != expCount {
		t.Fatalf("expected leaf callback to be called %d times; called %d", expCount, cbCount)
	}
	expCount = 3
	if len(treeNodes) != expCount {
		t.Fatalf("expected bvh tree to have %d nodes; got %d", expCount, len(treeNodes))
	}
}
