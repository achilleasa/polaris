package integrator

import (
	"fmt"
	"path"
	"runtime"
	"time"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/go-pathtrace/types"
)

const (
	relativePathToMainKernel = "CL/main.cl"
)

// Monte-carlo integrator implementation.
type MonteCarlo struct {
	// The associated device.
	device *device.Device

	// The allocated device buffers.
	buffers *bufferSet

	// The integrator kernels.
	kernels []*device.Kernel

	// Camera position and frustrum corners.
	cameraEyePos   types.Vec3
	cameraFrustrum scene.Frustrum
}

// Create a new monte-carlo integrator.
func NewMonteCarloIntegrator(dev *device.Device) *MonteCarlo {
	return &MonteCarlo{
		device: dev,
	}
}

// Initialize integrator.
func (in *MonteCarlo) Init() error {
	var err error

	if in.device == nil {
		return fmt.Errorf("integrator: invalid device handle")
	}

	// Initialize device
	_, thisFile, _, _ := runtime.Caller(0)
	pathToMainKernel := path.Join(path.Dir(thisFile), relativePathToMainKernel)
	err = in.device.Init(pathToMainKernel)
	if err != nil {
		in.Close()
		return err
	}

	// Allocate buffers
	in.buffers, err = newBufferSet(in.device)
	if err != nil {
		in.Close()
		return err
	}

	// Load all tracer-related kernels
	in.kernels = make([]*device.Kernel, numKernels)

	var kType kernelType
	for kType = 0; kType < numKernels; kType++ {
		in.kernels[kType], err = in.device.Kernel(kType.String())
		if err != nil {
			in.Close()
			return err
		}
	}

	return nil
}

// Release all integrator resources.
func (in *MonteCarlo) Close() {
	if in.buffers != nil {
		in.buffers.Release()
		in.buffers = nil
	}

	if in.kernels != nil {
		for _, kernel := range in.kernels {
			if kernel != nil {
				kernel.Release()
			}
		}
		in.kernels = nil
	}

	// Shutdown device
	if in.device != nil {
		in.device.Close()
	}
}

// Upload scene data to device buffers.
func (in *MonteCarlo) UploadSceneData(sc *scene.Scene) error {
	return in.buffers.UploadSceneData(sc)
}

// Upload camera data to device buffers.
func (in *MonteCarlo) UploadCameraData(camera *scene.Camera) error {
	in.cameraEyePos = camera.Position
	in.cameraFrustrum = camera.Frustrum
	return nil
}

// Resize device buffer whenever frame dimensions change.
func (in *MonteCarlo) ResizeOutputFrame(frameW, frameH uint32) error {
	return in.buffers.Resize(frameW, frameH)
}

// Generate primary rays.
func (in *MonteCarlo) GeneratePrimaryRays(blockReq *tracer.BlockRequest) (time.Duration, error) {
	kernel := in.kernels[generatePrimaryRays]

	texelDims := types.Vec2{
		1.0 / float32(blockReq.FrameW),
		1.0 / float32(blockReq.FrameH),
	}

	err := kernel.SetArgs(
		in.buffers.Rays,
		in.buffers.Paths,
		in.cameraFrustrum[0],
		in.cameraFrustrum[1],
		in.cameraFrustrum[2],
		in.cameraFrustrum[3],
		in.cameraEyePos,
		texelDims,
		blockReq.BlockY,
		blockReq.FrameW,
		blockReq.FrameH,
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
func (in *MonteCarlo) RayIntersectionTest(numRays uint32) (time.Duration, error) {
	kernel := in.kernels[rayIntersectionTest]

	err := kernel.SetArgs(
		in.buffers.Rays,
		in.buffers.BvhNodes,
		in.buffers.MeshInstances,
		in.buffers.Vertices,
		in.buffers.HitFlags,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, int(numRays), 0)
}

// Calculate ray intersections and fill out the hit buffer and the intersection
// buffer with intersection data for the closest ray/triangle intersection.
func (in *MonteCarlo) RayIntersectionQuery(numRays uint32) (time.Duration, error) {
	kernel := in.kernels[rayIntersectionQuery]

	err := kernel.SetArgs(
		in.buffers.Rays,
		in.buffers.BvhNodes,
		in.buffers.MeshInstances,
		in.buffers.Vertices,
		in.buffers.HitFlags,
		in.buffers.Intersections,
	)
	if err != nil {
		return 0, err
	}

	return kernel.Exec1D(0, int(numRays), 0)
}
