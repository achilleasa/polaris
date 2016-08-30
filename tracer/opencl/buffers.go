package opencl

import (
	"reflect"

	"github.com/achilleasa/go-pathtrace/asset/scene"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/gopencl/v1.2/cl"
)

// Size of buffer elements in bytes.
const (
	sizeofRay               = 32
	sizeofPath              = 32
	sizeofHitFlag           = 4 // uint32
	sizeofIntersection      = 32
	sizeofEmissiveSample    = 16 // float3 but takes same space as float4
	sizeofAccumulatorSample = 16 // float3
)

type bufferSet struct {
	// Output frame buffer
	FrameBuffer *device.Buffer

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

	// Emissive primitives
	EmissivePrimitives *device.Buffer

	// Primary/occlusion/indirect rays and paths
	Rays  [3]*device.Buffer
	Paths *device.Buffer

	// Intesection tests
	HitFlags      *device.Buffer
	Intersections *device.Buffer

	Accumulator     *device.Buffer
	EmissiveSamples *device.Buffer
	DebugOutput     *device.Buffer

	// Counters
	RayCounters [3]*device.Buffer
}

// Allocate new buffer set.
func newBufferSet(dev *device.Device) *bufferSet {
	return &bufferSet{
		// Output
		FrameBuffer: dev.Buffer("frameBuffer"),
		// Scene data
		BvhNodes:           dev.Buffer("bvhNodes"),
		MeshInstances:      dev.Buffer("meshInstances"),
		MaterialNodes:      dev.Buffer("materialNodes"),
		Textures:           dev.Buffer("textures"),
		TextureMetadata:    dev.Buffer("textureMetadata"),
		Vertices:           dev.Buffer("vertices"),
		Normals:            dev.Buffer("normals"),
		UV:                 dev.Buffer("uv"),
		MaterialIndices:    dev.Buffer("materialIndices"),
		EmissivePrimitives: dev.Buffer("emissivePrimitives"),
		// Tracer data
		Rays: [3]*device.Buffer{
			dev.Buffer("rays0"),
			dev.Buffer("rays1"),
			dev.Buffer("rays2"),
		},
		Paths:           dev.Buffer("paths"),
		HitFlags:        dev.Buffer("hitFlags"),
		Intersections:   dev.Buffer("intersections"),
		EmissiveSamples: dev.Buffer("emissiveSamples"),
		Accumulator:     dev.Buffer("accumulator"),
		DebugOutput:     dev.Buffer("debugOutput"),
		RayCounters: [3]*device.Buffer{
			dev.Buffer("numRays0"),
			dev.Buffer("numRays1"),
			dev.Buffer("numRays2"),
		},
	}
}

// Release all buffers.
func (bs *bufferSet) Release() {
	reflVal := reflect.ValueOf(*bs)
	var iface interface{}
	for fieldIndex := 0; fieldIndex < reflVal.NumField(); fieldIndex++ {
		iface = reflVal.Field(fieldIndex).Interface()
		switch val := iface.(type) {
		case *device.Buffer:
			val.Release()
		case []*device.Buffer:
			for _, d := range val {
				d.Release()
			}
		}
	}
}

// Resize frame-related buffers to the given frame dimensions.
func (bs *bufferSet) Resize(frameW, frameH uint32) error {
	var err error
	pixels := frameW * frameH

	err = bs.FrameBuffer.Allocate(int(pixels*4), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	for index := 0; index < len(bs.Rays); index++ {
		err = bs.Rays[index].Allocate(int(pixels*sizeofRay), cl.MEM_READ_WRITE)
		if err != nil {
			return err
		}
		err = bs.RayCounters[index].Allocate(4, cl.MEM_READ_WRITE)
		if err != nil {
			return err
		}
	}
	err = bs.Paths.Allocate(int(pixels*sizeofPath), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	err = bs.HitFlags.Allocate(int(pixels*sizeofHitFlag), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	err = bs.Intersections.Allocate(int(pixels*sizeofIntersection), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	err = bs.Accumulator.Allocate(int(pixels*sizeofAccumulatorSample), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	err = bs.EmissiveSamples.Allocate(int(pixels*sizeofEmissiveSample), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	err = bs.DebugOutput.Allocate(int(pixels*4), cl.MEM_READ_WRITE)
	if err != nil {
		return err
	}
	return nil
}

// Upload scene data to the device buffers.
func (bs *bufferSet) UploadSceneData(scene *scene.Scene) error {
	var err error

	targets := map[*device.Buffer]interface{}{
		bs.BvhNodes:           scene.BvhNodeList,
		bs.MeshInstances:      scene.MeshInstanceList,
		bs.MaterialNodes:      scene.MaterialNodeList,
		bs.Textures:           scene.TextureData,
		bs.TextureMetadata:    scene.TextureMetadata,
		bs.Vertices:           scene.VertexList,
		bs.Normals:            scene.NormalList,
		bs.UV:                 scene.UvList,
		bs.MaterialIndices:    scene.MaterialIndex,
		bs.EmissivePrimitives: scene.EmissivePrimitives,
	}

	for buf, data := range targets {
		err = buf.AllocateAndWriteData(data, cl.MEM_READ_ONLY)
		if err != nil {
			return err
		}
	}

	return nil
}
