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

// Packed solid or triangle primitive (128 bytes). This is a dual-use
// structure where some fields serve as c-like unions whose contents depend
// on the primitive type.
type packedPrimitive struct {
	// Primitive properties
	// x: primitive type
	// y: material index
	// z: 1.0 if primitive uses an emissive material
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

	// Uv coords for each vertex.
	// UV1( u1, v1, u2, v2 )
	// UV2( u3, v3, unused, unused )
	UV1 types.Vec4
	UV2 types.Vec4
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
		// Find material index
		found := false
		matIndex := 0
		isEmissive := 0
		for midx, mat := range sc.Materials {
			if mat == prim.Material {
				matIndex = midx
				found = true
				if mat.Type == scene.EmissiveMaterial {
					isEmissive = 1
				}
				break
			}
		}

		if !found {
			return nil, nil, fmt.Errorf("opencl: primitive %d references unknown material (%#+v)", idx, prim.Material)
		}

		switch prim.Type {
		case scene.TrianglePrimitive:
			primList[idx] = packedPrimitive{
				Properties: types.Vec4{float32(prim.Type), float32(matIndex), float32(isEmissive)},
				Center:     prim.TriCenter,
				Normal:     prim.TriNormal,
				Edge1:      prim.TriEdge[0],
				Edge2:      prim.TriEdge[1],
				Edge3:      prim.TriEdge[2],
				UV1:        types.Vec4{prim.UV[0][0], prim.UV[0][1], prim.UV[1][0], prim.UV[1][1]},
				UV2:        types.Vec4{prim.UV[2][0], prim.UV[2][1]},
			}
		default: // All other solids
			primList[idx] = packedPrimitive{
				Properties: types.Vec4{float32(prim.Type), float32(matIndex), float32(isEmissive)},
				// Center encodes origin
				Center: prim.Origin.Vec4(0),
				// Normal encodes extents
				Normal: prim.Dimensions.Vec4(0),
			}
		}

	}

	return matList, primList, nil
}
