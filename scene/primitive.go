package scene

import "github.com/achilleasa/go-pathtrace/types"

const (
	planePrimitive float32 = iota
	spherePrimitive
	boxPrimitive
	trianglePrimitive
)

// Internal representation of scene primitive. Each primitive is represented
// by 128 bytes. The Primitive struct works like a C union; field contents
// depend on the primitive type.
type Primitive struct {
	// Primitive properties
	// x: primitive type
	// y: material index
	properties types.Vec4

	// If solid, this is its origin. If triangle, this is its center
	// (and w coordinate stores max distance to vertices)
	center types.Vec4

	// If solid, this is its extends/dimensions. If triangle, this is its normal plane
	normal types.Vec4

	// Precalculated triangle edge planes
	edge1 types.Vec4
	edge2 types.Vec4
	edge3 types.Vec4

	// Uv coords for each triangle vertex.
	// UV1( u1, v1, u2, v2 )
	// UV2( u3, v3, unused, unused )
	uv1 types.Vec4
	uv2 types.Vec4
}

// Create new plane primitive
func NewPlane(origin types.Vec3, planeDist float32) *Primitive {
	return &Primitive{
		properties: types.Vec4{planePrimitive},
		center:     origin.Vec4(0),
		normal:     types.Vec4{planeDist},
	}
}

// Create new sphere primitive.
func NewSphere(origin types.Vec3, radius float32) *Primitive {
	return &Primitive{
		properties: types.Vec4{spherePrimitive},
		center:     origin.Vec4(0),
		normal:     types.Vec4{radius},
	}
}

// Create new box primitive.
func NewBox(origin types.Vec3, dims types.Vec3) *Primitive {
	return &Primitive{
		properties: types.Vec4{boxPrimitive},
		center:     origin.Vec4(0),
		normal:     dims.Vec4(0),
	}
}

// Create new triangle primitive. Vertices should be specified in clockwise order.
func NewTriangle(vertices [3]types.Vec3, uv [3]types.Vec2) *Primitive {
	prim := &Primitive{
		properties: types.Vec4{trianglePrimitive},
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
	prim.center = center.Vec4(maxDist)

	// Create triangle edges
	e1 := vertices[1].Sub(vertices[0])
	e2 := vertices[2].Sub(vertices[1])
	e3 := vertices[0].Sub(vertices[2])

	// Calc normal plane by taking the cross product of two edges
	// and then the dot product with the third vertice to get the plane dist
	normal := e1.Cross(e2).Normalize()
	prim.normal = normal.Vec4(normal.Dot(vertices[0]))

	// Calc 3 edge planes
	e1p := normal.Cross(e1).Normalize()
	e2p := normal.Cross(e2).Normalize()
	e3p := normal.Cross(e3).Normalize()

	// Calculate edge planes
	prim.edge1 = e1p.Vec4(e1p.Dot(vertices[0]))
	prim.edge2 = e2p.Vec4(e2p.Dot(vertices[1]))
	prim.edge3 = e3p.Vec4(e3p.Dot(vertices[2]))

	// Package UV coords for vertices
	prim.uv1 = types.Vec4{uv[0][0], uv[0][1], uv[1][0], uv[1][1]}
	prim.uv2 = types.Vec4{uv[2][0], uv[2][1]}

	return prim
}
