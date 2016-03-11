package scene

import "github.com/achilleasa/go-pathtrace/types"

const (
	DiffuseMaterial float32 = iota
	SpecularMaterial
	RefractiveMaterial
	EmissiveMaterial
)

// Internal material representation. Materials are packed to 48 bytes
type Material struct {
	// Material properties:
	// x: surface type
	// y: IOR (refractive materials)
	// z: specular roughness (when set to 0 this amounts to ideal specular reflection)
	Properties types.Vec4

	// Diffuse color.
	Diffuse types.Vec4

	// Emissive color (if surface emits light).
	Emissive types.Vec4
}

// Define a new diffuse material.
func NewDiffuseMaterial(color types.Vec3) *Material {
	return &Material{
		Properties: types.Vec4{DiffuseMaterial},
		Diffuse:    color.Vec4(0),
	}
}

// Define a new specular material.
func NewSpecularMaterial(color types.Vec3, roughness float32) *Material {
	return &Material{
		Properties: types.Vec4{SpecularMaterial, 0, roughness},
		Diffuse:    color.Vec4(0),
	}
}

// Define a new refractive material.
func NewRefractiveMaterial(color types.Vec3, IOR float32, roughness float32) *Material {
	return &Material{
		Properties: types.Vec4{RefractiveMaterial, IOR, roughness},
		Diffuse:    color.Vec4(0),
	}
}

// Emissive material.
func NewEmissiveMaterial(emissive types.Vec3) *Material {
	return &Material{
		Properties: types.Vec4{EmissiveMaterial},
		Emissive:   emissive.Vec4(0),
	}
}
