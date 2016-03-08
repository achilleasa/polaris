package scene

import (
	"fmt"

	"github.com/achilleasa/go-pathtrace/types"
)

type Scene struct {
	Camera *Camera

	Materials                []Material
	Primitives               []Primitive
	EmissivePrimitiveIndices []int
	matNameToIndex           map[string]int

	BgColor types.Vec3
}

func NewScene() *Scene {
	return &Scene{
		Materials:                make([]Material, 0),
		Primitives:               make([]Primitive, 0),
		EmissivePrimitiveIndices: make([]int, 0),
		matNameToIndex:           make(map[string]int, 0),
	}
}

// Attach a camera to the scene.
func (s *Scene) SetCamera(camera *Camera) {
	s.Camera = camera
}

// Add a material to the scene.
func (s *Scene) AddMaterial(name string, material *Material) error {
	if _, exists := s.matNameToIndex[name]; exists {
		return fmt.Errorf("scene: material already added")
	}
	s.Materials = append(s.Materials, *material)
	s.matNameToIndex[name] = len(s.Materials) - 1
	return nil
}

// Add a primitive to the scene.
func (s *Scene) AddPrimitive(primitive *Primitive, matName string) error {
	if matName == "" {
		return fmt.Errorf("scene: no material assigned to primitive")
	}
	matIndex, exists := s.matNameToIndex[matName]
	if !exists {
		return fmt.Errorf("scene: primitive references unknown material '%s'; ensure that the material is added to the scene before adding the primitive", matName)
	}
	// Patch material index into primitive properties
	primitive.properties[1] = float32(matIndex)
	s.Primitives = append(s.Primitives, *primitive)

	// If primitive uses an emissive material remember its index
	emissive := s.Materials[matIndex].emissive
	if emissive[0] > 0.0 || emissive[1] > 0.0 || emissive[2] > 0.0 {
		s.EmissivePrimitiveIndices = append(s.EmissivePrimitiveIndices, len(s.Primitives)-1)
	}

	return nil
}
