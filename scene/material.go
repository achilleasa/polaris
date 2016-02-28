package scene

import "github.com/achilleasa/go-pathtrace/types"

type MaterialType uint8

const (
	DiffuseMaterial MaterialType = iota
	SpecularMaterial
	RefractiveMaterial
	EmissiveMaterial
)

// Defines a scene material.
type Material struct {
	// The type of the material.
	Type MaterialType

	// Diffuse color.
	Diffuse types.Vec3

	// Emissive color (if material is light).
	Emissive types.Vec3

	// Index of refraction (refractive materials only)
	IOR float32
}
