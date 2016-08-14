package opencl

import (
	"fmt"
	"math"
	"time"

	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/go-pathtrace/types"
)

const (
	relativePathToMainKernel = "CL/main.cl"
)

// A container that stores handles to open CL kernels and any allocated device buffers.
type deviceResources struct {
	// The allocated device buffers.
	buffers *bufferSet

	// The set of kernels.
	kernels []*device.Kernel
}

// Using the supplied device as a target, load and compile all defined kernels.
func newDeviceResources(dev *device.Device) (*deviceResources, error) {
	var err error

	if dev == nil {
		return nil, fmt.Errorf("device_resources: invalid device handle")
	}

	// Allocate buffers
	dr := &deviceResources{
		buffers: newBufferSet(dev),
	}

	// Load all kernels
	dr.kernels = make([]*device.Kernel, numKernels)

	var kType kernelType
	for kType = 0; kType < numKernels; kType++ {
		dr.kernels[kType], err = dev.Kernel(kType.String())
		if err != nil {
			dr.Close()
			return nil, err
		}
	}

	return dr, nil
}

// Resize buffers to fit frame size.
func (dr *deviceResources) ResizeBuffers(frameW, frameH uint32) error {
	return dr.buffers.Resize(frameW, frameH)
}

// Release all allocated resources.
func (dr *deviceResources) Close() {
	if dr.buffers != nil {
		dr.buffers.Release()
		dr.buffers = nil
	}

	if dr.kernels != nil {
		for _, kernel := range dr.kernels {
			if kernel != nil {
				kernel.Release()
			}
		}
		dr.kernels = nil
	}
}

// Clear accumulator.
func (dr *deviceResources) ClearAccumulator(blockReq *tracer.BlockRequest) (time.Duration, error) {
	kernel := dr.kernels[clearAccumulator]
	numPixels := int(blockReq.FrameW * blockReq.BlockH)

	err := kernel.SetArgs(
		dr.buffers.Accumulator,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Generate primary rays.
func (dr *deviceResources) GeneratePrimaryRays(blockReq *tracer.BlockRequest, cameraEyePos types.Vec3, cameraFrustrum [4]types.Vec4) (time.Duration, error) {
	kernel := dr.kernels[generatePrimaryRays]

	texelDims := types.Vec2{
		1.0 / float32(blockReq.FrameW),
		1.0 / float32(blockReq.FrameH),
	}

	err := kernel.SetArgs(
		dr.buffers.Rays[0],
		dr.buffers.RayCounters[0],
		dr.buffers.Paths,
		cameraFrustrum[0],
		cameraFrustrum[1],
		cameraFrustrum[2],
		cameraFrustrum[3],
		cameraEyePos,
		texelDims,
		blockReq.BlockY,
		blockReq.FrameW,
		blockReq.FrameH,
		blockReq.Seed,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec2D(0, 0, int(blockReq.FrameW), int(blockReq.BlockH), 0, 0)
}

// Test for ray intersection. This method will update the hit buffer to indicate
// whether each ray intersects with the scene geometry or not. This method is
// much faster than an intersection query as it terminates on the first found
// intersection and does not evaulate intersection data.
func (dr *deviceResources) RayIntersectionTest(rayBufferIndex uint32, numPixels int) (time.Duration, error) {
	kernel := dr.kernels[rayIntersectionTest]

	err := kernel.SetArgs(
		dr.buffers.Rays[rayBufferIndex],
		dr.buffers.RayCounters[rayBufferIndex],
		dr.buffers.BvhNodes,
		dr.buffers.MeshInstances,
		dr.buffers.Vertices,
		dr.buffers.HitFlags,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Calculate ray intersections and fill out the hit buffer and the intersection
// buffer with intersection data for the closest ray/triangle intersection.
func (dr *deviceResources) RayIntersectionQuery(rayBufferIndex uint32, numPixels int) (time.Duration, error) {
	kernel := dr.kernels[rayIntersectionQuery]

	err := kernel.SetArgs(
		dr.buffers.Rays[rayBufferIndex],
		dr.buffers.RayCounters[rayBufferIndex],
		dr.buffers.BvhNodes,
		dr.buffers.MeshInstances,
		dr.buffers.Vertices,
		dr.buffers.HitFlags,
		dr.buffers.Intersections,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 64)
}

// Calculate ray intersections and fill out the hit buffer and the intersection
// buffer with intersection data for the closest ray/triangle intersection.
// This kernel works with ray packets and should only be used for primary rays.
func (dr *deviceResources) RayPacketIntersectionQuery(rayBufferIndex uint32, numPixels int) (time.Duration, error) {
	kernel := dr.kernels[rayPacketIntersectionQuery]

	err := kernel.SetArgs(
		dr.buffers.Rays[rayBufferIndex],
		dr.buffers.RayCounters[rayBufferIndex],
		dr.buffers.BvhNodes,
		dr.buffers.MeshInstances,
		dr.buffers.Vertices,
		dr.buffers.HitFlags,
		dr.buffers.Intersections,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 32)
}

// Evaluate shading for intersections. For each intersection, this kernel may
// generate an occlusion ray and a emissive sample as well as an indirect
// ray to be used for future bounces.
func (dr *deviceResources) ShadeHits(bounce, randSeed, numEmissives, rayBufferIndex uint32, numPixels int) (time.Duration, error) {
	kernel := dr.kernels[shadeHits]

	err := kernel.SetArgs(
		dr.buffers.Rays[rayBufferIndex],
		dr.buffers.RayCounters[rayBufferIndex],
		dr.buffers.Paths,
		dr.buffers.HitFlags,
		dr.buffers.Intersections,
		dr.buffers.Vertices,
		dr.buffers.Normals,
		dr.buffers.UV,
		dr.buffers.MaterialIndices,
		dr.buffers.MaterialNodes,
		dr.buffers.EmissivePrimitives,
		numEmissives,
		dr.buffers.TextureMetadata,
		dr.buffers.Textures,
		bounce,
		randSeed,
		// Occlusion rays and emissive samples
		dr.buffers.Rays[2], // occlusion rays always go to last ray buf
		dr.buffers.RayCounters[2],
		dr.buffers.EmissiveSamples,
		// Indirect rays
		dr.buffers.Rays[1-rayBufferIndex],
		dr.buffers.RayCounters[1-rayBufferIndex],
		//
		dr.buffers.Accumulator,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Shade primary ray misses by sampling the scene background. This kernel samples
// the background color or envmap using the ray direction and sets the
// accumulator to the sampled value.
func (dr *deviceResources) ShadePrimaryRayMisses(diffuseMatNodeIndex, rayBufferIndex uint32, numPixels int) (time.Duration, error) {
	kernel := dr.kernels[shadePrimaryRayMisses]

	err := kernel.SetArgs(
		dr.buffers.Rays[rayBufferIndex],
		dr.buffers.RayCounters[rayBufferIndex],
		dr.buffers.Paths,
		dr.buffers.HitFlags,
		dr.buffers.MaterialNodes,
		diffuseMatNodeIndex,
		dr.buffers.TextureMetadata,
		dr.buffers.Textures,
		dr.buffers.Accumulator,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Shade indirect ray misses by sampling the scene background. The main difference
// with ShadePrimaryRayMisses is that this kernel multiplies the path throughput
// with the bg sample and adds that to the accumulator.
func (dr *deviceResources) ShadeIndirectRayMisses(diffuseMatNodeIndex, rayBufferIndex uint32, numPixels int) (time.Duration, error) {
	kernel := dr.kernels[shadeIndirectRayMisses]

	err := kernel.SetArgs(
		dr.buffers.Rays[rayBufferIndex],
		dr.buffers.RayCounters[rayBufferIndex],
		dr.buffers.Paths,
		dr.buffers.HitFlags,
		dr.buffers.MaterialNodes,
		diffuseMatNodeIndex,
		dr.buffers.TextureMetadata,
		dr.buffers.Textures,
		dr.buffers.Accumulator,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Accumulate emissive samples for which no occlusion has been detected
// between the surface and the emissive primitive.
func (dr *deviceResources) AccumulateEmissiveSamples(rayBufferIndex uint32, numPixels int) (time.Duration, error) {
	kernel := dr.kernels[accumulateEmissiveSamples]

	err := kernel.SetArgs(
		dr.buffers.Rays[rayBufferIndex],
		dr.buffers.RayCounters[rayBufferIndex],
		dr.buffers.HitFlags,
		dr.buffers.EmissiveSamples,
		dr.buffers.Accumulator,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Perform tone-mapping using a simple version of Reinhard.
func (dr *deviceResources) TonemapSimpleReinhard(blockReq *tracer.BlockRequest) (time.Duration, error) {
	kernel := dr.kernels[tonemapSimpleReinhard]
	numPixels := int(blockReq.FrameW * blockReq.BlockH)
	sampleWeight := float32(1.0 / float32(blockReq.AccumulatedSamples+blockReq.SamplesPerPixel))
	err := kernel.SetArgs(
		dr.buffers.Accumulator,
		dr.buffers.Paths,
		dr.buffers.FrameBuffer,
		sampleWeight,
		blockReq.Exposure,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Clear debug buffer
func (dr *deviceResources) DebugClearBuffer(blockReq *tracer.BlockRequest) (time.Duration, error) {
	kernel := dr.kernels[debugClearBuffer]
	numPixels := int(blockReq.FrameW * blockReq.BlockH)

	err := kernel.SetArgs(
		dr.buffers.DebugOutput,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Generate a depth map based on the primary ray intersections.
func (dr *deviceResources) DebugPrimaryRayIntersectionDepth(blockReq *tracer.BlockRequest) (time.Duration, error) {
	_, err := dr.DebugClearBuffer(blockReq)
	if err != nil {
		return 0, err
	}

	type intersection struct {
		wuvt         types.Vec4
		meshInstance uint32
		triIndex     uint32
		_padding1    uint32
		_padding2    uint32
	}

	data, err := dr.buffers.Intersections.ReadDataIntoSlice(make([]intersection, 0))
	if err != nil {
		return 0, err
	}
	var maxDepth float32 = 1.0
	for _, i := range data.([]intersection) {
		if i.wuvt[3] != math.MaxFloat32 && i.wuvt[3] > maxDepth {
			maxDepth = i.wuvt[3]
		}
	}

	kernel := dr.kernels[debugPrimaryRayIntersectionDepth]
	numPixels := int(blockReq.FrameW * blockReq.BlockH)

	err = kernel.SetArgs(
		dr.buffers.RayCounters[0],
		dr.buffers.Paths,
		dr.buffers.HitFlags,
		dr.buffers.Intersections,
		maxDepth,
		dr.buffers.DebugOutput,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Generate a depth map based on the primary ray intersections.
func (dr *deviceResources) DebugPrimaryRayIntersectionNormals(blockReq *tracer.BlockRequest) (time.Duration, error) {
	_, err := dr.DebugClearBuffer(blockReq)
	if err != nil {
		return 0, err
	}

	kernel := dr.kernels[debugPrimaryRayIntersectionNormals]
	numPixels := int(blockReq.FrameW * blockReq.BlockH)

	err = kernel.SetArgs(
		dr.buffers.Rays[0],
		dr.buffers.RayCounters[0],
		dr.buffers.Paths,
		dr.buffers.HitFlags,
		dr.buffers.Intersections,
		dr.buffers.Vertices,
		dr.buffers.Normals,
		dr.buffers.UV,
		dr.buffers.MaterialIndices,
		dr.buffers.MaterialNodes,
		dr.buffers.TextureMetadata,
		dr.buffers.Textures,
		dr.buffers.DebugOutput,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Render emissiveSamples optionally masking occluded/not-occluded rays.
func (dr *deviceResources) DebugEmissiveSamples(blockReq *tracer.BlockRequest, maskOccluded, maskNotOccluded uint32) (time.Duration, error) {
	_, err := dr.DebugClearBuffer(blockReq)
	if err != nil {
		return 0, err
	}

	kernel := dr.kernels[debugEmissiveSamples]
	numPixels := int(blockReq.FrameW * blockReq.BlockH)

	err = kernel.SetArgs(
		dr.buffers.Rays[2],
		dr.buffers.RayCounters[2],
		dr.buffers.Paths,
		dr.buffers.HitFlags,
		dr.buffers.EmissiveSamples,
		maskOccluded,
		maskNotOccluded,
		dr.buffers.DebugOutput,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Render path throughput.
func (dr *deviceResources) DebugThroughput(blockReq *tracer.BlockRequest) (time.Duration, error) {
	_, err := dr.DebugClearBuffer(blockReq)
	if err != nil {
		return 0, err
	}

	kernel := dr.kernels[debugThroughput]
	numPixels := int(blockReq.FrameW * blockReq.BlockH)

	err = kernel.SetArgs(
		dr.buffers.Paths,
		dr.buffers.DebugOutput,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}

// Render accumulator contents.
func (dr *deviceResources) DebugAccumulator(blockReq *tracer.BlockRequest) (time.Duration, error) {
	_, err := dr.DebugClearBuffer(blockReq)
	if err != nil {
		return 0, err
	}

	kernel := dr.kernels[debugAccumulator]
	numPixels := int(blockReq.FrameW * blockReq.BlockH)
	sampleWeight := float32(1.0 / float32(blockReq.AccumulatedSamples+blockReq.SamplesPerPixel))

	err = kernel.SetArgs(
		sampleWeight,
		dr.buffers.Paths,
		dr.buffers.Accumulator,
		dr.buffers.DebugOutput,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, numPixels, 0)
}
