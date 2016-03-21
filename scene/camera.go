package scene

import (
	"fmt"

	"github.com/achilleasa/go-pathtrace/types"
)

// Stores the ray directions at the for corners of our camera frustrum. It is
// used as a shortcut for generating per pixel rays via interpolation of the
// corner rays. While we don't care about the W coordinate we use Vec4 since
// opencl provides a vectorized float4 type
type Frustrum [4]types.Vec4

func (fr Frustrum) String() string {
	return fmt.Sprintf(
		"Frustrum Rays:\nTL : (%3.3f, %3.3f, %3.3f)\nTR : (%3.3f, %3.3f, %3.3f)\nBL : (%3.3f, %3.3f, %3.3f)\nBR : (%3.3f, %3.3f, %3.3f)",
		fr[0][0], fr[0][1], fr[0][2],
		fr[1][0], fr[1][1], fr[1][2],
		fr[2][0], fr[2][1], fr[2][2],
		fr[3][0], fr[3][1], fr[3][2],
	)
}

// The camera type controls the scene camera.
type Camera struct {
	Position types.Vec3
	LookAt   types.Vec3
	Up       types.Vec3
	Pitch    float32
	Yaw      float32

	ViewMat  types.Mat4
	ProjMat  types.Mat4
	Frustrum Frustrum

	// Camera FOV
	FOV float32

	// Adjust the frustrum so that Y is inverted
	InvertY bool
}

func NewCamera(fov float32) *Camera {
	return &Camera{
		ViewMat:  types.Ident4(),
		ProjMat:  types.Ident4(),
		Position: types.Vec3{0, 0, 0},
		LookAt:   types.Vec3{0, 0, -1},
		Up:       types.Vec3{0, 1, 0},
		FOV:      fov,
	}
}

// Setup camera projection matrix.
func (c *Camera) SetupProjection(aspect float32) {
	c.ProjMat = types.Perspective4(c.FOV, aspect, 1, 1000)
	c.Update()
}

// Update camera.
func (c *Camera) Update() {
	dir := c.LookAt.Sub(c.Position).Normalize()
	pitchAxis := dir.Cross(c.Up)
	pitchQuat := types.QuatFromAxisAngle(pitchAxis, c.Pitch)
	yawQuat := types.QuatFromAxisAngle(c.Up, c.Yaw)

	orientQuat := pitchQuat.Mul(yawQuat).Normalize()

	// Update direction
	dir = orientQuat.Rotate(dir)
	c.LookAt = c.Position.Add(dir.Mul(1.0))

	c.ViewMat = types.LookAtV(c.Position, c.LookAt, c.Up)
	c.updateFrustrum()
}

func (c *Camera) InvViewProjMat() types.Mat4 {
	return c.ProjMat.Mul4(c.ViewMat).Inv()
}

// Generate a ray vector for each corner of the camera frustrum by
// multiplying clip space vectors for each corner with the inv proj/view
// matrix, applying perspective and subtracting the camera eye position.
func (c *Camera) updateFrustrum() {
	var v types.Vec4
	invProjViewMat := c.InvViewProjMat()

	var yUp float32 = 1.0
	if c.InvertY {
		yUp = -1.0
	}

	v = invProjViewMat.Mul4x1(types.XYZW(-1, yUp, -1, 1))
	c.Frustrum[0] = v.Mul(1.0 / v[3]).Vec3().Sub(c.Position).Vec4(0)

	v = invProjViewMat.Mul4x1(types.XYZW(1, yUp, -1, 1))
	c.Frustrum[1] = v.Mul(1.0 / v[3]).Vec3().Sub(c.Position).Vec4(0)

	v = invProjViewMat.Mul4x1(types.XYZW(-1, -yUp, -1, 1))
	c.Frustrum[2] = v.Mul(1.0 / v[3]).Vec3().Sub(c.Position).Vec4(0)

	v = invProjViewMat.Mul4x1(types.XYZW(1, -yUp, -1, 1))
	c.Frustrum[3] = v.Mul(1.0 / v[3]).Vec3().Sub(c.Position).Vec4(0)
}
