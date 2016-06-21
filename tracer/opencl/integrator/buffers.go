package integrator

import (
	"reflect"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/gopencl/v1.2/cl"
)

// Size of buffer elements in bytes.
const (
	sizeofRay     = 32
	sizeofPath    = 32
	sizeofHitFlag = 4 // uint32
)

type bufferSet struct {
	// Bvh node storage.
	BvhNodes *device.Buffer

	// Mesh instances.
	MeshInstances *device.Buffer

	// Surface materials.
	MaterialNodes *device.Buffer

	// Texture data
	Textures        *device.Buffer
	TextureMetadata *device.Buffer

	// Geometry
	Vertices        *device.Buffer
	Normals         *device.Buffer
	UV              *device.Buffer
	MaterialIndices *device.Buffer

	// Primary/secondary rays and paths
	Rays  *device.Buffer
	Paths *device.Buffer

	// Intesection tests
	HitFlags *device.Buffer
}

// Allocate new buffer set.
func newBufferSet(dev *device.Device) (*bufferSet, error) {
	return &bufferSet{
		// Scene data
		BvhNodes:        dev.Buffer("bvhNodes"),
		MeshInstances:   dev.Buffer("meshInstances"),
		MaterialNodes:   dev.Buffer("materialNodes"),
		Textures:        dev.Buffer("textures"),
		TextureMetadata: dev.Buffer("textureMetadata"),
		Vertices:        dev.Buffer("vertices"),
		Normals:         dev.Buffer("normals"),
		UV:              dev.Buffer("uv"),
		MaterialIndices: dev.Buffer("materialIndices"),
		// Tracer data
		Rays:     dev.Buffer("rays"),
		Paths:    dev.Buffer("paths"),
		HitFlags: dev.Buffer("hitFlags"),
	}, nil
}

// Release all buffers.
func (bs *bufferSet) Release() {
	reflVal := reflect.ValueOf(*bs)

	for fieldIndex := 0; fieldIndex < reflVal.NumField(); fieldIndex++ {
		reflVal.Field(fieldIndex).Interface().(*device.Buffer).Release()
	}
}

// Resize frame-related buffers to the given frame dimensions.
func (bs *bufferSet) Resize(frameW, frameH uint32) error {
	var err error
	pixels := frameW * frameH

	err = bs.Rays.Allocate(int(pixels*sizeofRay), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	err = bs.Paths.Allocate(int(pixels*sizeofPath), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	err = bs.HitFlags.Allocate(int(pixels*sizeofHitFlag), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}

	return nil
}

// Upload scene data to the device buffers.
func (bs *bufferSet) UploadSceneData(scene *scene.Scene) error {
	var err error

	targets := map[*device.Buffer]interface{}{
		bs.BvhNodes:        scene.BvhNodeList,
		bs.MeshInstances:   scene.MeshInstanceList,
		bs.MaterialNodes:   scene.MaterialNodeList,
		bs.Textures:        scene.TextureData,
		bs.TextureMetadata: scene.TextureMetadata,
		bs.Vertices:        scene.VertexList,
		bs.Normals:         scene.NormalList,
		bs.UV:              scene.UvList,
		bs.MaterialIndices: scene.MaterialIndex,
	}

	for buf, data := range targets {
		err = buf.AllocateAndWriteData(data, cl.MEM_READ_ONLY)
		if err != nil {
			return err
		}
	}

	return nil
}
