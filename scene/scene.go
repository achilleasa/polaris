package scene

import (
	"fmt"

	"github.com/achilleasa/go-pathtrace/types"
)

type Scene struct {
	Camera *Camera

	Materials  []*Material
	Primitives []*Primitive

	BgColor types.Vec3
}

func NewScene() *Scene {
	return &Scene{
		Materials:  make([]*Material, 0),
		Primitives: make([]*Primitive, 0),
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

// Add a primitive to the scene.
func (s *Scene) AddPrimitive(primitive *Primitive) error {
	for _, prim := range s.Primitives {
		if prim == primitive {
			return fmt.Errorf("scene: primitive already added")
		}
	}
	if primitive.Material == nil {
		return fmt.Errorf("scene: no material assigned to primitive")
	}
	for _, mat := range s.Materials {
		if mat == primitive.Material {
			s.Primitives = append(s.Primitives, primitive)
			return nil
		}
	}

	return fmt.Errorf("scene: primitive references unknown material; ensure that the material is added to the scene before adding the primitive")
}
