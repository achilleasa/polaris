package scene

import "github.com/achilleasa/go-pathtrace/types"

type PrimitiveType uint32

const (
	PlanePrimitive PrimitiveType = iota
	SpherePrimitive
	BoxPrimitive
)

// Defines a scene primitive.
type Primitive struct {
	// The primitive type.
	Type PrimitiveType

	// The primitive origin.
	Origin types.Vec3

	// Primitive dimensions. Stored as a Vec3 but the dimension component
	// count varies depending on primitive type.
	Dimensions types.Vec3

	// The primitive material. Must be added to the scene before the primitive
	Material *Material
}

// Create new plane primitive
func NewPlane(origin types.Vec3, planeDist float32, material *Material) *Primitive {
	return &Primitive{
		Type:       PlanePrimitive,
		Origin:     origin.Normalize(),
		Dimensions: types.Vec3{planeDist},
		Material:   material,
	}
}

// Create new sphere primitive.
func NewSphere(origin types.Vec3, radius float32, material *Material) *Primitive {
	return &Primitive{
		Type:       SpherePrimitive,
		Origin:     origin,
		Dimensions: types.Vec3{radius},
		Material:   material,
	}
}

// Create new box primitive.
func NewBox(origin types.Vec3, dims types.Vec3, material *Material) *Primitive {
	return &Primitive{
		Type:       BoxPrimitive,
		Origin:     origin,
		Dimensions: dims,
		Material:   material,
	}
}
