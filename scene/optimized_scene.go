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
	LData int32

	Max   types.Vec3
	RData int32
}

// Set bounding box.
func (n *BvhNode) SetBBox(bbox [2]types.Vec3) {
	n.Min = bbox[0]
	n.Max = bbox[1]
}

// Set left and right child node indices.
func (n *BvhNode) SetChildNodes(left, right uint32) {
	n.LData = int32(left)
	n.RData = int32(right)
}

// Set mesh instance index.
func (n *BvhNode) SetMeshIndex(index uint32) {
	n.LData = -int32(index)
}

// Get Mesh index.
func (n *BvhNode) GetMeshIndex() (index uint32) {
	return uint32(-n.LData)
}

// Set primitive index and count.
func (n *BvhNode) SetPrimitives(firstPrimIndex, count uint32) {
	n.LData = -int32(firstPrimIndex)
	n.RData = int32(count)
}

// Get primitive index and count.
func (n *BvhNode) GetPrimitives() (firstPrimIndex, count uint32) {
	return uint32(-n.LData), uint32(n.RData)
}

// Add offset to indices of child nodes.
func (n *BvhNode) OffsetChildNodes(offset int32) {
	// Ignore leafs
	if n.LData <= 0 {
		return
	}

	n.LData += offset
	n.RData += offset
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
type MatNodeBlendFunc int32

const (
	Mix MatNodeBlendFunc = iota
	Fresnel
)

// Specification of bxdf types for material leaf nodes.
type MatBxdfType int32

const (
	Emissive MatBxdfType = 1 << iota
	Diffuse
	SpecularReflection
	SpecularTransmission
	SpecularMicrofacetReflection
	SpecularMicrofacetTransmission
)

// Materials are represented as a tree where nodes define a blending operation
// and leaves define a BRDF for the surface. This allows us to define complex
// materials (e.g. 20% diffuse and 80% specular). In order to use the same structure
// for both nodes and leaves we define a "union-like" field whose values depend on
// the node type.
type MaterialNode struct {
	// Specifies a color for this node. Depending on the BRDF type
	// this can be diffuse, emissive e.t.c.
	Kval types.Vec4

	// This field has different contents depending on the node type.
	//
	// For intermediate material nodes it contains a value for controlling blending:
	// - Mix: probability for selecting the left child
	// - Fresnel: surface IOR
	//
	// For leafs it contains a BRDF-specific parameter like roughness e.t.c
	Nval float32

	IOR    float32
	IORTex int32

	// Set to 1 if this is a node; 0 if this is a leaf
	IsNode uint32

	// This union like structure has different contents depending on the node
	// type.
	//
	// For intermediate material nodes:
	// - UnionData[0] is the index of the left child
	// - UnionData[1] is the index of the right child
	// - UnionData[2] points to the tex index that overrides NVal (-1 if unused)
	// - Uniondata[3] specifies the blending function (mix, fresnel blend)
	//
	// For leaf nodes:
	// - UnionData[0] points to the tex index that overrides Kval (-1 if unused)
	// - UnionData[1] points to the tex index that serves as a normal map (-1 if unused)
	// - UnionData[2] points to the tex index that overrides NVal (-1 if unused)
	// - UnionData[3] specifies the BRDF type (diffuse, specular e.t.c)
	UnionData [4]int32
}

// Initialize material node.
func (m *MaterialNode) Init() {
	m.UnionData[0] = -1
	m.UnionData[1] = -1
	m.UnionData[2] = -1
	m.UnionData[3] = -1
}

// Set left child node index.
func (m *MaterialNode) SetLeftIndex(index uint32) {
	m.UnionData[0] = int32(index)
}

// Set right child node index.
func (m *MaterialNode) SetRightIndex(index uint32) {
	m.UnionData[1] = int32(index)
}

// Get left child node index.
func (m *MaterialNode) GetLeftIndex() uint32 {
	return uint32(m.UnionData[0])
}

// Get right child node index.
func (m *MaterialNode) GetRightIndex() uint32 {
	return uint32(m.UnionData[1])
}

// Set node blend function.
func (m *MaterialNode) SetBlendFunc(blendfunc MatNodeBlendFunc) {
	m.UnionData[3] = int32(blendfunc)
}

// Set IOR tex index.
func (m *MaterialNode) SetIORTex(tex *ParsedTexture) {
	if tex == nil {
		m.IORTex = -1
		return
	}
	m.IORTex = int32(tex.TexIndex)
}

// Set Kval tex index.
func (m *MaterialNode) SetKvalTex(tex *ParsedTexture) {
	if tex == nil {
		m.UnionData[0] = -1
		return
	}
	m.UnionData[0] = int32(tex.TexIndex)
}

// Get Kval tex index.
func (m *MaterialNode) GetKvalTex() int32 {
	return m.UnionData[0]
}

// Set normal tex index.
func (m *MaterialNode) SetNormalTex(tex *ParsedTexture) {
	if tex == nil {
		m.UnionData[1] = -1
		return
	}
	m.UnionData[1] = int32(tex.TexIndex)
}

// Set Nval tex index.
func (m *MaterialNode) SetNvalTex(tex *ParsedTexture) {
	if tex == nil {
		m.UnionData[2] = -1
		return
	}
	m.UnionData[2] = int32(tex.TexIndex)
}

// Set leaf BxDF type.
func (m *MaterialNode) SetBxdfType(bxdfType MatBxdfType) {
	m.UnionData[3] = int32(bxdfType)
}

// Get leaf BxDF type.
func (m *MaterialNode) GetBxdfType() MatBxdfType {
	return MatBxdfType(m.UnionData[3])
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

// The type of an emissive primitive.
type EmissivePrimitiveType uint32

const (
	AreaLight EmissivePrimitiveType = iota
	EnvironmentLight
)

// An emissive primitive.
type EmissivePrimitive struct {
	// A transformation matrix for converting the primitive vertices from
	// local space to world space.
	Transform types.Mat4

	// The area of the emissive primitive.
	Area float32

	// The triangle index for this emissive.
	PrimitiveIndex uint32

	// The material node index for this emissive.
	MaterialNodeIndex uint32

	// The type of the emissive primitive.
	Type EmissivePrimitiveType
}

type Scene struct {
	BvhNodeList        []BvhNode
	MeshInstanceList   []MeshInstance
	MaterialNodeList   []MaterialNode
	MaterialNodeRoots  []uint32
	EmissivePrimitives []EmissivePrimitive

	// Texture definitions and the associated data.
	TextureData     []byte
	TextureMetadata []TextureMetadata

	// Primitives are stored as an array of structs.
	VertexList    []types.Vec4
	NormalList    []types.Vec4
	UvList        []types.Vec2
	MaterialIndex []uint32

	// Indices to material nodes used for storing the scene global
	// properties such as diffuse and emissive colors.
	SceneDiffuseMatIndex  int32
	SceneEmissiveMatIndex int32

	// The scene camera.
	Camera *Camera
}
