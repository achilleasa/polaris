package scene

import "github.com/achilleasa/go-pathtrace/types"

// BvhPrimitive wraps a primitive pointer with an AABB.
type BvhPrimitive struct {
	Min    types.Vec3
	Max    types.Vec3
	Center types.Vec3

	Primitive *Primitive
}

// Bvh node definition. Each node takes 32 bytes.
type BvhNode struct {
	// Bounding box min extent. If this is a node then
	// the W component is > 0 and contains the index to the left node; If this
	// is a leaf then W component is <=0 and contains the index of
	// the first primitive in the leaf.
	Min types.Vec4

	// Bounding box max extent. If this is a node then
	// the W component is > 0 and contains the index to the right node; If this
	// is a leaf then W component is < 0 and contains the index of
	// the count of primitives in the leaf
	Max types.Vec4
}
