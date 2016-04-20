package reader

import "github.com/achilleasa/go-pathtrace/types"

// The primitive struct represents a parsed triangle primitive.
type primitive struct {
	vertices [3]types.Vec3
	normals  [3]types.Vec3
	uvs      [3]types.Vec2
	bbox     [2]types.Vec3
	material uint32
}

// A mesh is comprised of a list of primitives
type mesh struct {
	name       string
	primitives []*primitive
	bbox       [2]types.Vec3
}

// A mesh instance reuses the geometry of a mesh and combines it with a
// transformation matrix.
type meshInstance struct {
	mesh      uint32
	transform types.Mat4
	bbox      [2]types.Vec3
}

// A material consists of a set of vector and scalar parameters that define the
// surface characteristics. In addition, it may define set of textures to modulate
// these parameters.
type material struct {
	name string

	// Diffuse/Albedo color.
	kd types.Vec3

	// Specular color.
	ks types.Vec3

	// Emissive color.
	ke types.Vec3

	// Index of refraction.
	ni float32

	// Roughness.
	nr float32

	// Textures for modulating above parameters.
	kdTex     int32
	ksTex     int32
	keTex     int32
	normalTex int32
	niTex     int32
	nrTex     int32
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
type texture struct {
	format TextureFormat

	width  uint32
	height uint32

	data []byte
}

// Camera settings
type camera struct {
	fov  float32
	eye  types.Vec3
	look types.Vec3
	up   types.Vec3
}

// The parsed scene contains all the scene elements that were loaded by a reader.
type scene struct {
	meshes        []*mesh
	meshInstances []*meshInstance
	textures      []*texture
	materials     []*material
	camera        *camera
}

func newScene() *scene {
	return &scene{
		meshes:        make([]*mesh, 0),
		meshInstances: make([]*meshInstance, 0),
		textures:      make([]*texture, 0),
		materials:     make([]*material, 0),
		camera: &camera{
			fov:  45.0,
			eye:  types.Vec3{0, 0, 0},
			look: types.Vec3{0, 0, -1},
			up:   types.Vec3{0, 1, 0},
		},
	}
}

func newMesh(name string) *mesh {
	return &mesh{
		name:       name,
		primitives: make([]*primitive, 0),
	}
}

func newMaterial(name string) *material {
	return &material{
		name: name,
		// Disable textures,
		kdTex:     -1,
		ksTex:     -1,
		keTex:     -1,
		normalTex: -1,
		niTex:     -1,
		nrTex:     -1,
	}
}
