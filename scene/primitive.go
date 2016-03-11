package scene

import "github.com/achilleasa/go-pathtrace/types"

const (
	PlanePrimitive float32 = iota
	SpherePrimitive
	BoxPrimitive
	TrianglePrimitive
)

// Internal representation of scene primitive. Each primitive is represented
// by 128 bytes. The Primitive struct works like a C union; field contents
// depend on the primitive type.
type Primitive struct {
	// Primitive properties
	// x: primitive type
	// y: material index
	Properties types.Vec4

	// If solid, this is its origin. If triangle, this is its center
	// (and w coordinate stores max distance to vertices)
	Center types.Vec4

	// If solid, this is its extends/dimensions. If triangle, this is its normal plane
	Normal types.Vec4

	// Precalculated triangle edge planes
	Edge1 types.Vec4
	Edge2 types.Vec4
	Edge3 types.Vec4

	// Uv coords for each triangle vertex.
	// UV1( u1, v1, u2, v2 )
	// UV2( u3, v3, unused, unused )
	UV1 types.Vec4
	UV2 types.Vec4
}

// Create new plane primitive
func NewPlane(origin types.Vec3, planeDist float32) *Primitive {
	return &Primitive{
		Properties: types.Vec4{PlanePrimitive},
		Center:     origin.Vec4(0),
		Normal:     types.Vec4{planeDist},
	}
}

// Create new sphere primitive.
func NewSphere(origin types.Vec3, radius float32) *Primitive {
	return &Primitive{
		Properties: types.Vec4{SpherePrimitive},
		Center:     origin.Vec4(0),
		Normal:     types.Vec4{radius},
	}
}

// Create new box primitive.
func NewBox(origin types.Vec3, dims types.Vec3) *Primitive {
	return &Primitive{
		Properties: types.Vec4{BoxPrimitive},
		Center:     origin.Vec4(0),
		Normal:     dims.Vec4(0),
	}
}

// Create new triangle primitive. Vertices should be specified in clockwise order.
func NewTriangle(vertices [3]types.Vec3, uv [3]types.Vec2) *Primitive {
	prim := &Primitive{
		Properties: types.Vec4{TrianglePrimitive},
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
	prim.Center = center.Vec4(maxDist)

	// Create triangle edges
	e1 := vertices[1].Sub(vertices[0])
	e2 := vertices[2].Sub(vertices[1])
	e3 := vertices[0].Sub(vertices[2])

	// Calc normal plane by taking the cross product of two edges
	// and then the dot product with the third vertice to get the plane dist
	normal := e1.Cross(e2).Normalize()
	prim.Normal = normal.Vec4(normal.Dot(vertices[0]))

	// Calc 3 edge planes
	e1p := normal.Cross(e1).Normalize()
	e2p := normal.Cross(e2).Normalize()
	e3p := normal.Cross(e3).Normalize()

	// Calculate edge planes
	prim.Edge1 = e1p.Vec4(e1p.Dot(vertices[0]))
	prim.Edge2 = e2p.Vec4(e2p.Dot(vertices[1]))
	prim.Edge3 = e3p.Vec4(e3p.Dot(vertices[2]))

	// Package UV coords for vertices
	prim.UV1 = types.Vec4{uv[0][0], uv[0][1], uv[1][0], uv[1][1]}
	prim.UV2 = types.Vec4{uv[2][0], uv[2][1]}

	return prim
}
