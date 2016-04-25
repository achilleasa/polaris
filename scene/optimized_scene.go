package scene

import "github.com/achilleasa/go-pathtrace/types"

// Bvh nodes are comprised of two Vec3 and two multipurpose int32 parameters
// whose value depends on the node type:
//
// - For non-leaf nodes (top/bottom) BVH they are both >0 and point to the L/R child nodes
// - For top BVH leafs:
//   - left W is <= 0 and points to the mesh instance index
// - For bottom BVH leafs:
//   - left W is <= 0 and point to the first triangle primitive index
//   - right W is >0 and contains the count of leaf primitives
//
//
type BvhNode struct {
	Min   types.Vec3
	lData int32

	Max   types.Vec3
	rData int32
}

// Set bounding box.
func (n *BvhNode) SetBBox(bbox [2]types.Vec3) {
	n.Min = bbox[0]
	n.Max = bbox[1]
}

// Set left and right child node indices.
func (n *BvhNode) SetChildNodes(left, right uint32) {
	n.lData = int32(left)
	n.rData = int32(right)
}

// Set mesh instance index.
func (n *BvhNode) SetMeshInstance(index uint32) {
	n.lData = -int32(index)
}

// Set primitive index and count.
func (n *BvhNode) SetPrimitives(firstPrimIndex, count uint32) {
	n.lData = -int32(firstPrimIndex)
	n.rData = int32(count)
}

type MeshInstance struct {
	Flags uint32

	Mesh uint32

	BvhNodes [2]uint32

	Transform types.Mat4
}

type Material struct {
	Kval      types.Vec4
	ior       float32
	brdfParam float32

	UnionData [4]int
}

type EmissivePrimitive struct {
	MeshInstance uint32
	Primitive    uint32
	Material     uint32
}

type Scene struct {
	BvhNodeList      []BvhNode
	MeshInstanceList []MeshInstance
	MaterialList     []Material
	EmissiveList     []EmissivePrimitive

	// Primitives are stored as an array of structs.
	VerticeList   []types.Vec4
	NormalList    []types.Vec4
	UvList        []types.Vec2
	MaterialIndex []uint32

	// The scene camera.
	Camera *Camera
}
