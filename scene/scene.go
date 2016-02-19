package scene

import "github.com/achilleasa/go-pathtrace/types"

type Scene struct {
	Camera *Camera

	BgColor types.Vec3
}

func NewScene() *Scene {
	return &Scene{}
}

// Attach a camera to the scene.
func (s *Scene) SetCamera(camera *Camera) {
	s.Camera = camera
}
