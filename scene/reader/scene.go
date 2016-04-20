package reader

import "github.com/achilleasa/go-pathtrace/types"

// The primitive struct represents a parsed triangle primitive.
type Primitive struct {
	Vertices      [3]types.Vec3
	Normals       [3]types.Vec3
	UVs           [3]types.Vec2
	BBox          [2]types.Vec3
	MaterialIndex uint32
}

// A mesh is comprised of a list of primitives
type Mesh struct {
	Name       string
	Primitives []*Primitive
	BBox       [2]types.Vec3
}

// A mesh instance reuses the geometry of a mesh and combines it with a
// transformation matrix.
type MeshInstance struct {
	MeshIndex uint32
	Transform types.Mat4
	BBox      [2]types.Vec3
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
		Name:       name,
		Primitives: make([]*Primitive, 0),
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
