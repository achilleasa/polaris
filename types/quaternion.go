package types

import "math"

// Quaternion implementation taken from https://github.com/go-gl/mathgl/blob/master/mgl32/quat.go
type Quat struct {
	V Vec3
	W float32
}

// Create identity quaternion.
func QuatIdent() Quat {
	return Quat{
		V: Vec3{},
		W: 1.0,
	}
}

// Create a quaternion from an axis vector and an angle.
func QuatFromAxisAngle(axis Vec3, angle float32) Quat {
	sin := float32(math.Sin(float64(angle * 0.5)))
	cos := float32(math.Cos(float64(angle * 0.5)))
	return Quat{
		V: axis.Mul(sin),
		W: cos,
	}
}

// Rotates a vector by the rotation this quaternion represents.
// This will result in a 3D vector.
func (q1 Quat) Rotate(v Vec3) Vec3 {
	cross := q1.V.Cross(v)
	// v + 2q_w * (q_v x v) + 2q_v x (q_v x v)
	return v.Add(cross.Mul(2 * q1.W)).Add(q1.V.Mul(2).Cross(cross))
}

// Multiplies two quaternions. This can be seen as a rotation. Note that
// Multiplication is NOT commutative, meaning q1.Mul(q2) does not necessarily
// equal q2.Mul(q1).
func (q1 Quat) Mul(q2 Quat) Quat {
	return Quat{
		q1.V.Cross(q2.V).Add(q2.V.Mul(q1.W)).Add(q1.V.Mul(q2.W)),
		q1.W*q2.W - q1.V.Dot(q2.V),
	}
}

// Returns the Length of the quaternion, also known as its Norm. This is the same thing as
// the Len of a Vec4
func (q1 Quat) Len() float32 {
	return float32(math.Sqrt(float64(q1.W*q1.W + q1.V[0]*q1.V[0] + q1.V[1]*q1.V[1] + q1.V[2]*q1.V[2])))
}

// Normalizes the quaternion, returning its versor (unit quaternion).
//
// This is the same as normalizing it as a Vec4.
func (q1 Quat) Normalize() Quat {
	length := q1.Len()

	absDelta := 1 - length
	if absDelta < 0 {
		absDelta = -absDelta
	}

	if absDelta < floatCmpEpsilon {
		return q1
	}
	if length == 0 {
		return QuatIdent()
	}
	if length == float32(math.Inf(1)) {
		length = math.MaxFloat32
	}

	return Quat{q1.V.Mul(1 / length), q1.W * 1 / length}
}

// The inverse of a quaternion. The inverse is equivalent
// to the conjugate divided by the square of the length.
func (q1 Quat) Inverse() Quat {
	scaler := 1.0 / (q1.V.Dot(q1.V) + q1.W*q1.W)
	return Quat{
		q1.V.Mul(-1.0 * scaler),
		q1.W * scaler,
	}
}

// Returns the homogeneous 3D rotation matrix corresponding to the quaternion.
func (q1 Quat) Mat4() Mat4 {
	w, x, y, z := q1.W, q1.V[0], q1.V[1], q1.V[2]
	return Mat4{
		1 - 2*y*y - 2*z*z, 2*x*y + 2*w*z, 2*x*z - 2*w*y, 0,
		2*x*y - 2*w*z, 1 - 2*x*x - 2*z*z, 2*y*z + 2*w*x, 0,
		2*x*z + 2*w*y, 2*y*z - 2*w*x, 1 - 2*x*x - 2*y*y, 0,
		0, 0, 0, 1,
	}
}
