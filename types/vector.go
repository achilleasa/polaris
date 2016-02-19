package types

import (
	"math"

	"golang.org/x/image/math/f32"
)

type Vec2 f32.Vec2
type Vec3 f32.Vec3
type Vec4 f32.Vec4

// Define a 2 component vector.
func XY(x, y float32) Vec2 {
	return Vec2{x, y}
}

// Define a 3 component vector.
func XYZ(x, y, z float32) Vec3 {
	return Vec3{x, y, z}
}

// Define a 4 component vector.
func XYZW(x, y, z, w float32) Vec4 {
	return Vec4{x, y, z, w}
}

// Expand a 3 component vector to a Vec4.
func (v Vec3) Vec4(w float32) Vec4 {
	return Vec4{v[0], v[1], v[2], w}
}

// Subtract a vector.
func (v Vec3) Sub(v2 Vec3) Vec3 {
	return Vec3{v[0] - v2[0], v[1] - v2[1], v[2] - v2[2]}
}

// Multiply a 3 component vector with a scalar.
func (v Vec3) Mul(s float32) Vec3 {
	return Vec3{v[0] * s, v[1] * s, v[2] * s}
}

// Get 3 component vector length.
func (v Vec3) Len() float32 {
	return float32(math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
}

// Normalize 3 component vector.
func (v Vec3) Normalize() Vec3 {
	l := 1.0 / v.Len()
	if l < floatCmpEpsilon {
		return Vec3{}
	}
	return Vec3{v[0] * l, v[1] * l, v[2] * l}
}

// Calculate dot product of 2 vectors
func (v Vec2) Dot(v2 Vec2) float32 {
	return v[0]*v2[0] + v[1]*v2[1]
}

// Calculate cross product of 2 vectors.
func (v Vec3) Cross(v2 Vec3) Vec3 {
	return Vec3{v[1]*v2[2] - v[2]*v2[1], v[2]*v2[0] - v[0]*v2[2], v[0]*v2[1] - v[1]*v2[0]}
}

// Reduce a 4 component vector to a Vec3.
func (v Vec4) Vec3() Vec3 {
	return Vec3{v[0], v[1], v[2]}
}

// Multiply 4 component vector with scalar.
func (v Vec4) Mul(s float32) Vec4 {
	return Vec4{v[0] * s, v[1] * s, v[2] * s, v[3] * s}
}

// Extract the top-left 3x3 matrix from a 4x4 matrix.
func (m Mat4) Mat3() Mat3 {
	return Mat3{
		m[0], m[1], m[2],
		m[4], m[5], m[6],
		m[8], m[9], m[10],
	}
}
