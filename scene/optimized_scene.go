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

// Specification of material node blend functions.
type MatNodeBlendFunc int

const (
	Mix MatNodeBlendFunc = iota
	Fresnel
)

// Specification of brdf types for material leaf nodes.
type MatBrdfType int

const (
	Diffuse MatBrdfType = iota
	Specular
	Refractive
)

// Materials are represented as a tree where nodes define a blending operation
// and leaves define a BRDF for the surface. This allows us to define complex
// materials (e.g. 20% diffuse and 80% specular). In order to use the same structure
// for both nodes and leaves we define a "union-like" field whose values depend on
// the node type.
type MaterialNode struct {
	// Specifies a color for this node. Depending on the BRDF type
	// this can be diffuse, specular e.t.c.
	Kval types.Vec4

	// Material refractive Index
	IOR float32

	// This field has different contents depending on the node type.
	//
	// For intermediate material nodes it contains a value for controlling blending
	// if blend mode (UnionData[2] == Mix). For leaf nodes it contains a parameter
	// that depends on the selected BRDF type. This can be reflectance, specularity, emission scaler e.t.c
	Nval float32

	// A coefficient for scaling the material contribution. Its value is
	// calculated by multiplying the probabilities of selecting each intermediate
	// node.
	BlendCoeff float32

	// Small padding to keep fields aligned
	padding float32

	// This union like structure has different contents depending on the node
	// type.
	//
	// For intermediate material nodes:
	// - UnionData[0] is the index of the left child
	// - UnionData[1] is the index of the right child
	// - Uniondata[2] specifies the blending function (mix, fresnel blend)
	// - Uniondata[3] is unused
	//
	// For leaf nodes:
	// - UnionData[0] points to the tex index that overrides Kval (-1 if unused)
	// - UnionData[1] points to the tex index that serves as a normal map (-1 if unused)
	// - UnionData[2] points to the tex index that overrides NVal (-1 if unused)
	// - UnionData[3] specifies the BRDF type (diffuse, specular e.t.c)
	UnionData [4]int
}

// Set left child node index.
func (m *MaterialNode) SetLeftIndex(index int) {
	m.UnionData[0] = index
}

// Set right child node index.
func (m *MaterialNode) SetRightIndex(index int) {
	m.UnionData[1] = index
}

// Set node blend function.
func (m *MaterialNode) SetBlendFunc(blendfunc MatNodeBlendFunc) {
	m.UnionData[2] = int(blendfunc)
}

// Set Kval tex index.
func (m *MaterialNode) SetKvalTex(texIndex int) {
	m.UnionData[0] = texIndex
}

// Set normal tex index.
func (m *MaterialNode) SetNormalTex(texIndex int) {
	m.UnionData[1] = texIndex
}

// Set Nval tex index.
func (m *MaterialNode) SetNvalTex(texIndex int) {
	m.UnionData[2] = texIndex
}

// Set leaf BRDF type.
func (m *MaterialNode) SetBrdfType(brdfType MatBrdfType) {
	m.UnionData[3] = int(brdfType)
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
	MaterialNodeList []MaterialNode

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
