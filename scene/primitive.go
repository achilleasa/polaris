package scene

import "github.com/achilleasa/go-pathtrace/types"

type PrimitiveType uint32

const (
	PlanePrimitive PrimitiveType = iota
	SpherePrimitive
	BoxPrimitive
	TrianglePrimitive
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

	// Triangle primitive center, normal-, edge-planes and uv coords
	TriCenter types.Vec4
	TriNormal types.Vec4
	TriEdge   [3]types.Vec4
	UV        [3]types.Vec2

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

// Create new triangle primitive. Vertices should be specified in clockwise order.
func NewTriangle(vertices [3]types.Vec3, uv [3]types.Vec2, material *Material) *Primitive {
	prim := &Primitive{
		Type:     TrianglePrimitive,
		Material: material,
		UV:       uv,
	}

	// Calc center and max distance
	center := vertices[0].Add(vertices[1]).Add(vertices[2]).Mul(1.0 / 3.0)
	maxDist := vertices[0].Sub(center).Len()
	d := vertices[1].Sub(center).Len()
	if d > maxDist {
		maxDist = d
	}
	d = vertices[2].Sub(center).Len()
	if d > maxDist {
		maxDist = d
	}
	prim.TriCenter = center.Vec4(maxDist)

	// Create triangle edges
	e1 := vertices[1].Sub(vertices[0])
	e2 := vertices[2].Sub(vertices[1])
	e3 := vertices[0].Sub(vertices[2])

	// Calc normal plane by taking the cross product of two edges
	// and then the dot product with the third vertice to get the plane dist
	normal := e1.Cross(e2).Normalize()
	prim.TriNormal = normal.Vec4(normal.Dot(vertices[0]))

	// Calc 3 edge planes
	e1p := normal.Cross(e1).Normalize()
	e2p := normal.Cross(e2).Normalize()
	e3p := normal.Cross(e3).Normalize()

	// Calculate edge planes
	prim.TriEdge[0] = e1p.Vec4(e1p.Dot(vertices[0]))
	prim.TriEdge[1] = e2p.Vec4(e2p.Dot(vertices[1]))
	prim.TriEdge[2] = e3p.Vec4(e3p.Dot(vertices[2]))
	return prim
}
