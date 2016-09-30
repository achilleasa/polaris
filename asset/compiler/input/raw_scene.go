package input

import (
	"math"

	"github.com/achilleasa/polaris/asset"
	"github.com/achilleasa/polaris/types"
)

type Material struct {
	Name string

	// Material expression.
	Expression string

	// Relative path for textures.
	AssetRelPath *asset.Resource

	// True if material is referenced by scene geometry.
	Used bool
}

// A triangle primitive
type Primitive struct {
	Vertices      [3]types.Vec3
	Normals       [3]types.Vec3
	UVs           [3]types.Vec2
	MaterialIndex int

	bbox   [2]types.Vec3
	center types.Vec3
}

// Set the primitive AABB.
func (prim *Primitive) SetBBox(bbox [2]types.Vec3) {
	prim.bbox = bbox
}

// Set the primitive center.
func (prim *Primitive) SetCenter(center types.Vec3) {
	prim.center = center
}

// Get the primitive AABB.
func (prim *Primitive) BBox() [2]types.Vec3 {
	return prim.bbox
}

// Get primitive AABB center.
func (prim *Primitive) Center() types.Vec3 {
	return prim.center
}

// A mesh is constructed by a list of primitive.
type Mesh struct {
	Name       string
	Primitives []*Primitive

	bbox            [2]types.Vec3
	bboxNeedsUpdate bool
}

// A mesh instance applies a transformation to a particular Mesh.
type MeshInstance struct {
	MeshIndex uint32
	Transform types.Mat4

	bbox   [2]types.Vec3
	center types.Vec3
}

// Set the mesh instance AABB.
func (mi *MeshInstance) SetBBox(bbox [2]types.Vec3) {
	mi.bbox = bbox
}

// Set the mesh instance center.
func (mi *MeshInstance) SetCenter(center types.Vec3) {
	mi.center = center
}

// Get AABB.
func (mi *MeshInstance) BBox() [2]types.Vec3 {
	return mi.bbox
}

// Get AABB center.
func (mi *MeshInstance) Center() types.Vec3 {
	return mi.center
}

// Set the mesh AABB.
func (m *Mesh) SetBBox(bbox [2]types.Vec3) {
	m.bbox = bbox
}

// Mark the bbox of this mesh as dirty.
func (m *Mesh) MarkBBoxDirty() {
	m.bboxNeedsUpdate = true
}

// Get mesh bounding box.
func (m *Mesh) BBox() [2]types.Vec3 {
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

// Create a new mesh.
func NewMesh(name string) *Mesh {
	return &Mesh{
		Name:            name,
		Primitives:      make([]*Primitive, 0),
		bboxNeedsUpdate: true,
	}
}

// Camera settings.
type Camera struct {
	FOV  float32
	Eye  types.Vec3
	Look types.Vec3
	Up   types.Vec3
}

// The scene contains all elements that are processed and optimized by the scene compiler.
// optimized
type Scene struct {
	Meshes        []*Mesh
	MeshInstances []*MeshInstance
	Materials     []*Material
	Camera        *Camera
}

// Create a new scene.
func NewScene() *Scene {
	return &Scene{
		Meshes:        make([]*Mesh, 0),
		MeshInstances: make([]*MeshInstance, 0),
		Materials:     make([]*Material, 0),
		Camera: &Camera{
			FOV:  45.0,
			Eye:  types.Vec3{0, 0, 0},
			Look: types.Vec3{0, 0, -1},
			Up:   types.Vec3{0, 1, 0},
		},
	}
}
