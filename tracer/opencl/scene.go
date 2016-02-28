package opencl

import (
	"fmt"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/types"
)

// Internal representation of each scene material.
// Each material takes 48 bytes.
type packedMaterial struct {
	// Material properties:
	// x: surface type
	// y: IOR (refractive materials)
	Properties types.Vec4

	// Diffuse color.
	Diffuse types.Vec4

	// Emissive color (if surface emits light).
	Emissive types.Vec4
}

// Internal representation of each scene primitive.
// Each primitive takes 48 bytes.
type packedPrimitive struct {
	// Primitive properties
	// x: primitive type
	// y: material index
	// z: 1.0 if primitive uses an emissive material
	Properties types.Vec4

	// The primitive origin.
	Origin types.Vec4

	// Primitive extents/dimensions
	Extents types.Vec4
}

// Process scene and return packed form of materials and primitives
func packScene(sc *scene.Scene) ([]packedMaterial, []packedPrimitive, error) {
	// package each material
	matList := make([]packedMaterial, len(sc.Materials))
	for idx, mat := range sc.Materials {
		matList[idx] = packedMaterial{
			Properties: types.Vec4{float32(mat.Type), mat.IOR},
			Diffuse:    mat.Diffuse.Vec4(0),
			Emissive:   mat.Emissive.Vec4(0),
		}
	}

	// package primitives
	primList := make([]packedPrimitive, len(sc.Primitives))
	for idx, prim := range sc.Primitives {
		primList[idx] = packedPrimitive{
			Properties: types.Vec4{float32(prim.Type)},
			Origin:     prim.Origin.Vec4(0),
			Extents:    prim.Dimensions.Vec4(0),
		}

		// Find material index
		found := false
		for matIdx, mat := range sc.Materials {
			if mat == prim.Material {
				primList[idx].Properties[1] = float32(matIdx)
				if mat.Type == scene.EmissiveMaterial {
					primList[idx].Properties[2] = 1.0
				}
				found = true
				break
			}
		}

		if !found {
			return nil, nil, fmt.Errorf("opencl: primitive %d references unknown material (%#+v)", idx, prim.Material)
		}

	}

	return matList, primList, nil
}
