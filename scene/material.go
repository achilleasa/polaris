package scene

import "github.com/achilleasa/go-pathtrace/types"

const (
	diffuseMaterial float32 = iota
	specularMaterial
	refractiveMaterial
	emissiveMaterial
)

// Internal material representation. Materials are packed to 48 bytes
type Material struct {
	// Material properties:
	// x: surface type
	// y: IOR (refractive materials)
	// z: specular roughness (when set to 0 this amounts to ideal specular reflection)
	properties types.Vec4

	// Diffuse color.
	diffuse types.Vec4

	// Emissive color (if surface emits light).
	emissive types.Vec4
}

// Define a new diffuse material
func DiffuseMaterial(color types.Vec3) *Material {
	return &Material{
		properties: types.Vec4{diffuseMaterial},
		diffuse:    color.Vec4(0),
	}
}

// Define a new specular material
func SpecularMaterial(color types.Vec3, roughness float32) *Material {
	return &Material{
		properties: types.Vec4{specularMaterial, 0, roughness},
		diffuse:    color.Vec4(0),
	}
}

// Define a new refractive material
func RefractiveMaterial(color types.Vec3, IOR float32, roughness float32) *Material {
	return &Material{
		properties: types.Vec4{refractiveMaterial, IOR, roughness},
		diffuse:    color.Vec4(0),
	}
}

// Emissive material
func EmissiveMaterial(emissive types.Vec3) *Material {
	return &Material{
		properties: types.Vec4{emissiveMaterial},
		emissive:   emissive.Vec4(0),
	}
}
