package scene

import (
	"fmt"

	"github.com/achilleasa/go-pathtrace/types"
)

type Scene struct {
	Camera *Camera

	Materials []*Material

	BgColor types.Vec3
}

func NewScene() *Scene {
	return &Scene{
		Materials: make([]*Material, 0),
	}
}

// Attach a camera to the scene.
func (s *Scene) SetCamera(camera *Camera) {
	s.Camera = camera
}

// Add a material to the scene.
func (s *Scene) AddMaterial(material *Material) error {
	for _, mat := range s.Materials {
		if mat == material {
			return fmt.Errorf("scene: material already added")
		}
	}
	s.Materials = append(s.Materials, material)
	return nil
}
