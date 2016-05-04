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
func (n *BvhNode) SetMeshIndex(index uint32) {
	n.lData = -int32(index)
}

// Get Mesh index.
func (n *BvhNode) GetMeshIndex() (index uint32) {
	return uint32(-n.lData)
}

// Set primitive index and count.
func (n *BvhNode) SetPrimitives(firstPrimIndex, count uint32) {
	n.lData = -int32(firstPrimIndex)
	n.rData = int32(count)
}

// Get primitive index and count.
func (n *BvhNode) GetPrimitives() (firstPrimIndex, count uint32) {
	return uint32(-n.lData), uint32(n.rData)
}

// Add offset to indices of child nodes.
func (n *BvhNode) OffsetChildNodes(offset int32) {
	// Ignore leafs
	if n.lData <= 0 {
		return
	}

	n.lData += offset
	n.rData += offset
}

// The MeshInstance structure allows us to apply a transformation matrix to
// a scene mesh so that it can be positioned inside the scene.
type MeshInstance struct {
	MeshIndex uint32

	// The BVH tree root for the mesh geometry. This is shared by all
	// instances of the same mesh.
	BvhRoot uint32

	padding [2]uint32

	// A transformation matrix for positioning the mesh.
	Transform types.Mat4
}

// The texture metadata. All texture data is stored as a contiguous memory block.
type TextureMetadata struct {
	// Texture format.
	Format TextureFormat

	// Texture dimensions.
	Width  uint32
	Height uint32

	// Offset to the beginning of texture data
	DataOffset uint32
}

type Scene struct {
	BvhNodeList      []BvhNode
	MeshInstanceList []MeshInstance

	// Texture definitions and the associated data.
	TextureData     []byte
	TextureMetadata []TextureMetadata

	// Primitives are stored as an array of structs.
	VertexList    []types.Vec4
	NormalList    []types.Vec4
	UvList        []types.Vec2
	MaterialIndex []uint32

	// The scene camera.
	Camera *Camera
}
