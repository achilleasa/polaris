package scene

import (
	"math"

	"github.com/achilleasa/go-pathtrace/types"
)

// The primitive struct represents a parsed triangle primitive.
type ParsedPrimitive struct {
	Vertices      [3]types.Vec3
	Normals       [3]types.Vec3
	UVs           [3]types.Vec2
	MaterialIndex uint32

	bbox   [2]types.Vec3
	center types.Vec3
}

// Set the primitive AABB.
func (prim *ParsedPrimitive) SetBBox(bbox [2]types.Vec3) {
	prim.bbox = bbox
}

// Set the primitive center.
func (prim *ParsedPrimitive) SetCenter(center types.Vec3) {
	prim.center = center
}

// Implements compiler.BoundedVolume
func (prim *ParsedPrimitive) BBox() [2]types.Vec3 {
	return prim.bbox
}

// Implements compiler.BoundedVolume
func (prim *ParsedPrimitive) Center() types.Vec3 {
	return prim.center
}

// A mesh is comprised of a list of primitives
type ParsedMesh struct {
	Name       string
	Primitives []*ParsedPrimitive

	bbox            [2]types.Vec3
	bboxNeedsUpdate bool
}

// Set the mesh AABB.
func (m *ParsedMesh) SetBBox(bbox [2]types.Vec3) {
	m.bbox = bbox
}

// Mark the bbox of this mesh as dirty.
func (m *ParsedMesh) MarkBBoxDirty() {
	m.bboxNeedsUpdate = true
}

// Get mesh bounding box.
func (m *ParsedMesh) BBox() [2]types.Vec3 {
	if m.bboxNeedsUpdate {
		m.bbox = [2]types.Vec3{
			types.Vec3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32},
			types.Vec3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32},
		}

		for _, prim := range m.Primitives {
			primBBox := prim.BBox()
			m.bbox[0] = types.MinVec3(m.bbox[0], primBBox[0])
			m.bbox[1] = types.MaxVec3(m.bbox[1], primBBox[1])
		}

		m.bboxNeedsUpdate = false
	}

	return m.bbox
}

// A mesh instance reuses the geometry of a mesh and combines it with a
// transformation matrix.
type ParsedMeshInstance struct {
	MeshIndex uint32
	Transform types.Mat4
	bbox      [2]types.Vec3
	center    types.Vec3
}

// Set the  mesh instance AABB.
func (mi *ParsedMeshInstance) SetBBox(bbox [2]types.Vec3) {
	mi.bbox = bbox
}

// Set the mesh instance center.
func (mi *ParsedMeshInstance) SetCenter(center types.Vec3) {
	mi.center = center
}

// Implements compiler.BoundedVolume
func (mi *ParsedMeshInstance) BBox() [2]types.Vec3 {
	return mi.bbox
}

// Implements compiler.BoundedVolume
func (mi *ParsedMeshInstance) Center() types.Vec3 {
	return mi.center
}

// A material consists of a set of vector and scalar parameters that define the
// surface characteristics. In addition, it may define set of textures to modulate
// these parameters.
type ParsedMaterial struct {
	Name string

	// Diffuse/Albedo color.
	Kd types.Vec3

	// Specular color.
	Ks types.Vec3

	// Emissive color.
	Ke types.Vec3

	// Index of refraction.
	Ni float32

	// Roughness.
	Nr float32

	// Textures for modulating above parameters.
	KdTex     int32
	KsTex     int32
	KeTex     int32
	NormalTex int32
	NiTex     int32
	NrTex     int32
}

// Return true if material contains a diffuse component.
func (pm *ParsedMaterial) IsDiffuse() bool {
	return pm.Kd.Len() > 0 || pm.KdTex != -1
}

// Return true if material contains a specular component.
func (pm *ParsedMaterial) IsSpecular() bool {
	return pm.Ks.Len() > 0 || pm.KsTex != -1
}

// Return true if material contains an emissive component.
func (pm *ParsedMaterial) IsEmissive() bool {
	return pm.Ke.Len() > 0 || pm.KeTex != -1
}

// Return true if material is refractive.
func (pm *ParsedMaterial) IsRefractive() bool {
	return pm.Ni != 0 || pm.NiTex != -1
}

// A texture image and its metadata.
type ParsedTexture struct {
	Format TextureFormat

	Width  uint32
	Height uint32

	Data []byte
}

// Camera settings
type ParsedCamera struct {
	FOV  float32
	Eye  types.Vec3
	Look types.Vec3
	Up   types.Vec3
}

// The parsed scene contains all the scene elements that were loaded by a reader.
type ParsedScene struct {
	Meshes        []*ParsedMesh
	MeshInstances []*ParsedMeshInstance
	Textures      []*ParsedTexture
	Materials     []*ParsedMaterial
	Camera        *ParsedCamera
}

// Create a new parsed scene.
func NewParsedScene() *ParsedScene {
	return &ParsedScene{
		Meshes:        make([]*ParsedMesh, 0),
		MeshInstances: make([]*ParsedMeshInstance, 0),
		Textures:      make([]*ParsedTexture, 0),
		Materials:     make([]*ParsedMaterial, 0),
		Camera: &ParsedCamera{
			FOV:  45.0,
			Eye:  types.Vec3{0, 0, 0},
			Look: types.Vec3{0, 0, -1},
			Up:   types.Vec3{0, 1, 0},
		},
	}
}

// Create a new parsed mesh.
func NewParsedMesh(name string) *ParsedMesh {
	return &ParsedMesh{
		Name:            name,
		Primitives:      make([]*ParsedPrimitive, 0),
		bboxNeedsUpdate: true,
	}
}

// Create a new parsed material.
func NewParsedMaterial(name string) *ParsedMaterial {
	return &ParsedMaterial{
		Name: name,
		// Disable textures,
		KdTex:     -1,
		KsTex:     -1,
		KeTex:     -1,
		NormalTex: -1,
		NiTex:     -1,
		NrTex:     -1,
	}
}
