package reader

import (
	"math"

	"github.com/achilleasa/go-pathtrace/types"
)

// The primitive struct represents a parsed triangle primitive.
type Primitive struct {
	Vertices      [3]types.Vec3
	Normals       [3]types.Vec3
	UVs           [3]types.Vec2
	MaterialIndex uint32

	bbox   [2]types.Vec3
	center types.Vec3
}

// Implements optimizer.BoundedVolume
func (prim *Primitive) BBox() [2]types.Vec3 {
	return prim.bbox
}

// Implements optimized.BoundedVolume
func (prim *Primitive) Center() types.Vec3 {
	return prim.center
}

// A mesh is comprised of a list of primitives
type Mesh struct {
	Name       string
	Primitives []*Primitive

	bbox            [2]types.Vec3
	bboxNeedsUpdate bool
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

// A mesh instance reuses the geometry of a mesh and combines it with a
// transformation matrix.
type MeshInstance struct {
	MeshIndex uint32
	Transform types.Mat4
	bbox      [2]types.Vec3
	center    types.Vec3
}

// Implements optimizer.BoundedVolume
func (mi *MeshInstance) BBox() [2]types.Vec3 {
	return mi.bbox
}

// Implements optimized.BoundedVolume
func (mi *MeshInstance) Center() types.Vec3 {
	return mi.center
}

// A material consists of a set of vector and scalar parameters that define the
// surface characteristics. In addition, it may define set of textures to modulate
// these parameters.
type Material struct {
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

type TextureFormat uint32

const (
	Luminance8 TextureFormat = iota
	Luminance32
	Rgb8
	Rgba8
	Rgb32
	Rgba32
)

// A texture image and its metadata.
type Texture struct {
	Format TextureFormat

	Width  uint32
	Height uint32

	Data []byte
}

// Camera settings
type Camera struct {
	FOV  float32
	Eye  types.Vec3
	Look types.Vec3
	Up   types.Vec3
}

// The parsed scene contains all the scene elements that were loaded by a reader.
type Scene struct {
	Meshes        []*Mesh
	MeshInstances []*MeshInstance
	Textures      []*Texture
	Materials     []*Material
	Camera        *Camera
}

func newScene() *Scene {
	return &Scene{
		Meshes:        make([]*Mesh, 0),
		MeshInstances: make([]*MeshInstance, 0),
		Textures:      make([]*Texture, 0),
		Materials:     make([]*Material, 0),
		Camera: &Camera{
			FOV:  45.0,
			Eye:  types.Vec3{0, 0, 0},
			Look: types.Vec3{0, 0, -1},
			Up:   types.Vec3{0, 1, 0},
		},
	}
}

func newMesh(name string) *Mesh {
	return &Mesh{
		Name:            name,
		Primitives:      make([]*Primitive, 0),
		bboxNeedsUpdate: true,
	}
}

func newMaterial(name string) *Material {
	return &Material{
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
