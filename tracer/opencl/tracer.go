package opencl

import (
	"fmt"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/achilleasa/go-pathtrace/log"
	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/go-pathtrace/types"
)

type Tracer struct {
	logger log.Logger

	sync.Mutex
	wg sync.WaitGroup

	// The device associated with this tracer instance.
	device *device.Device

	// The allocated device resources.
	resources *deviceResources

	// The tracer id.
	id string

	// A buffer for asynchronous updates. Updates are grouped by type and
	// latest updates always overwrite the previous ones.
	changeBuffer map[tracer.ChangeType]interface{}

	// Statistics for last rendered frame.
	stats *tracer.Stats

	// The tracer rendering pipeline.
	pipeline *Pipeline

	// The uploaded optimized scene data.
	sceneData *scene.Scene

	// Camera attributes
	cameraPosition types.Vec3
	cameraFrustrum scene.Frustrum
}

// Create a new opencl tracer.
func NewTracer(id string, device *device.Device, pipeline *Pipeline) (tracer.Tracer, error) {
	loggerName := fmt.Sprintf("opencl tracer (%s)", device.Name)

	tr := &Tracer{
		logger:       log.New(loggerName),
		device:       device,
		id:           id,
		changeBuffer: make(map[tracer.ChangeType]interface{}, 0),
		stats:        &tracer.Stats{},
		pipeline:     pipeline,
	}

	return tr, nil
}

// Get tracer id.
func (tr *Tracer) Id() string {
	return tr.id
}

// Get tracer flags.
func (tr *Tracer) Flags() tracer.Flag {
	return tracer.Local | tracer.GLInterop
}

// Get the computation speed estimate (in GFlops).
func (tr *Tracer) Speed() uint32 {
	return tr.device.Speed
}

// Initialize tracer
func (tr *Tracer) Init() error {
	var err error
	tr.Lock()
	defer tr.Unlock()

	// Init device
	_, thisFile, _, _ := runtime.Caller(0)
	pathToMainKernel := path.Join(path.Dir(thisFile), relativePathToMainKernel)
	err = tr.device.Init(pathToMainKernel)
	if err != nil {
		tr.cleanup()
		return err
	}

	// Load kernels and allocate buffers
	tr.resources, err = newDeviceResources(tr.device)
	if err != nil {
		tr.cleanup()
		return err
	}

	return nil
}

// Shutdown and cleanup tracer.
func (tr *Tracer) Close() {
	tr.Lock()
	defer tr.Unlock()

	tr.cleanup()
}

// Cleanup tracer. This method is meant to be called while holding tr.Lock()
func (tr *Tracer) cleanup() {
	// Cleanup allocated resources
	if tr.resources != nil {
		tr.resources.Close()
		tr.resources = nil
	}

	// Shutdown device
	if tr.device != nil {
		tr.device.Close()
		tr.device = nil
	}

	tr.sceneData = nil
}

// Retrieve last frame statistics.
func (tr *Tracer) Stats() *tracer.Stats {
	return tr.stats
}

// Update tracer state
func (tr *Tracer) UpdateState(mode tracer.UpdateMode, changeType tracer.ChangeType, data interface{}) (time.Duration, error) {
	tr.changeBuffer[changeType] = data

	if mode == tracer.Synchronous {
		return tr.commitChanges()
	}

	return time.Duration(0), nil
}

// Commit queued state changes.
func (tr *Tracer) commitChanges() (time.Duration, error) {
	if len(tr.changeBuffer) == 0 {
		return 0, nil
	}

	var err error
	start := time.Now()
	for changeType, data := range tr.changeBuffer {
		switch changeType {
		case tracer.FrameDimensions:
			dims := data.([2]uint32)
			err = tr.resources.ResizeBuffers(dims[0], dims[1])
		case tracer.SceneData:
			tr.sceneData = data.(*scene.Scene)
			err = tr.resources.buffers.UploadSceneData(tr.sceneData)
		case tracer.CameraData:
			camera := data.(*scene.Camera)
			tr.cameraPosition = camera.Position
			tr.cameraFrustrum = camera.Frustrum
		default:
			err = fmt.Errorf("unsupported change type %d", changeType)
		}

		if err != nil {
			return time.Since(start), err
		}
	}

	tr.changeBuffer = make(map[tracer.ChangeType]interface{}, 0)
	return time.Since(start), nil
}

// Process block request.
func (tr *Tracer) Trace(blockReq *tracer.BlockRequest) (time.Duration, error) {
	var err error
	start := time.Now()

	_, err = tr.commitChanges()
	if err != nil {
		return time.Since(start), err
	}

	if tr.sceneData == nil {
		return time.Since(start), ErrNoSceneData
	}

	// If we have reset our sample counter, reset the accumulator
	if blockReq.AccumulatedSamples == 0 && tr.pipeline.Reset != nil {
		_, err = tr.pipeline.Reset(tr, blockReq)
		if err != nil {
			return time.Since(start), err
		}
	}

	var sample uint32
	for sample = 0; sample < blockReq.SamplesPerPixel; sample++ {
		// Generate primary rays
		if tr.pipeline.PrimaryRayGenerator != nil {
			_, err = tr.pipeline.PrimaryRayGenerator(tr, blockReq)
			if err != nil {
				return time.Since(start), err
			}
		}

		// Run integrator
		if tr.pipeline.Integrator != nil {
			_, err = tr.pipeline.Integrator(tr, blockReq)
			if err != nil {
				return time.Since(start), err
			}
		}
	}

	return time.Since(start), nil
}

// Run post-process filters and update the framebuffer with the processed output.
func (tr *Tracer) SyncFramebuffer(blockReq *tracer.BlockRequest) (time.Duration, error) {
	var err error
	start := time.Now()

	if tr.sceneData == nil {
		return time.Since(start), ErrNoSceneData
	}

	if tr.pipeline.PostProcess == nil {
		return time.Since(start), nil
	}

	for _, stage := range tr.pipeline.PostProcess {
		_, err = stage(tr, blockReq)
		if err != nil {
			return time.Since(start), err
		}
	}

	return time.Since(start), nil
}
