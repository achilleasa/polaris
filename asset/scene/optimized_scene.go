package scene

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/achilleasa/polaris/asset/texure"
	"github.com/achilleasa/polaris/types"
	"github.com/olekukonko/tablewriter"
)

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

// Materials are represented as a tree where nodes define a blending operation
// and leaves define a BxDF for the surface. This allows us to define complex
// materials (e.g. 20% diffuse and 80% specular). In order to use the same structure
// for both nodes and leaves we define a "union-like" field whose values depend on
// the node type.
type MaterialNode struct {
	// Layout:
	// [0] type
	// [1] left child
	// [2] right child or transmittance texture
	// [3] bump map, reflectance, specularity or radiance texture
	Union1 [4]int32

	// Layout:
	// [0-3] reflectance or specularity or radiance
	// [0-3] RGB intIORs for dispersion
	// [0-2] blend weights
	Union2 types.Vec4

	// Layout:
	// [0-3] transmittance
	// [0-3] RGB extIORs for dispersion
	Union3 types.Vec4

	// Layout:
	// [0] internal IOR
	// [1] external IOR
	// [2] roughness or radiance scaler
	Union4 types.Vec3

	// Layout:
	// [0] roughness texture
	Union5 [1]int32
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
	Format texture.Format

	// Texture dimensions.
	Width  uint32
	Height uint32

	// Offset to the beginning of texture data
	DataOffset uint32
}

type Scene struct {
	BvhNodeList        []BvhNode
	MeshInstanceList   []MeshInstance
	MaterialNodeList   []MaterialNode
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

// Build a tabular representation of scene statistics.
func (sc *Scene) Stats() string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoFormatHeaders(false)
	table.SetHeader([]string{"Asset Type", "Asset", "Size"})
	table.Append([]string{"Geometry", "---", fmtSize(sc.VertexList, sc.NormalList, sc.UvList, sc.BvhNodeList)})
	table.Append([]string{"", "Vertices", fmtSize(sc.VertexList)})
	table.Append([]string{"", "Normals", fmtSize(sc.NormalList)})
	table.Append([]string{"", "UVs", fmtSize(sc.UvList)})
	table.Append([]string{"", "BVH", fmtSize(sc.BvhNodeList)})
	table.Append([]string{" ", " ", " "})
	table.Append([]string{"Mesh/emissives", "---", fmtSize(sc.MeshInstanceList, sc.EmissivePrimitives)})
	table.Append([]string{"", "Mesh instances", fmtSize(sc.MeshInstanceList)})
	table.Append([]string{"", "Emissives", fmtSize(sc.EmissivePrimitives)})
	table.Append([]string{" ", " ", " "})
	table.Append([]string{"Materials", "---", fmtSize(sc.MaterialIndex, sc.MaterialNodeList)})
	table.Append([]string{"", "Mat. indices", fmtSize(sc.MaterialIndex)})
	table.Append([]string{"", "Mat. nodes", fmtSize(sc.MaterialNodeList)})
	table.Append([]string{" ", " ", " "})
	table.Append([]string{"Textures", "---", fmtSize(sc.TextureMetadata, sc.TextureData)})
	table.Append([]string{"", "Metadata", fmtSize(sc.TextureMetadata)})
	table.Append([]string{"", "Data", fmtSize(sc.TextureData)})
	table.SetFooter([]string{"Total", " ", strings.TrimLeft(fmtSize(sc.VertexList, sc.NormalList, sc.UvList, sc.BvhNodeList, sc.MeshInstanceList, sc.EmissivePrimitives, sc.MaterialNodeList, sc.MaterialIndex, sc.TextureMetadata, sc.TextureData), " ")})

	table.Render()
	return buf.String()
}

// Sum the total space used by a set of slices and return back a formatted
// value with the appropriate byte/kb/mb unit.
func fmtSize(items ...interface{}) string {
	var totalBytes float32 = 0.0
	for _, item := range items {
		t := reflect.TypeOf(item)
		v := reflect.ValueOf(item)
		if v.Len() == 0 {
			continue
		}

		totalBytes += float32(int(t.Elem().Size()) * v.Len())
	}

	if totalBytes < 1e3 {
		return fmt.Sprintf("%3d bytes", int(totalBytes))
	} else if totalBytes < 1e6 {
		return fmt.Sprintf("%3.1f kb", totalBytes/1e3)
	}
	return fmt.Sprintf("%5.1f mb", totalBytes/1e6)
}
