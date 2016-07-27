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
)

type pipelineStage func(tr *clTracer, blockReq *tracer.BlockRequest) (time.Duration, error)

type clTracer struct {
	logger log.Logger

	sync.Mutex
	wg sync.WaitGroup

	// The device associated with this tracer instance.
	device *device.Device

	// The allocated device resources.
	resources *deviceResources

	// The tracer id.
	id string

	// A buffer for queuing updates. Updates are grouped by type and
	// latest updates always overwrite the previous ones.
	updateBuffer map[tracer.UpdateType]interface{}

	// A channel for receiving block requests from the renderer.
	blockReqChan chan tracer.BlockRequest

	// A channel for signaling the worker to exit.
	closeChan chan struct{}

	// Statistics for last rendered frame.
	stats *tracer.Stats

	// The tracer rendering pipeline.
	pipeline []pipelineStage

	// Device speed in Gflops.
	speed uint32

	// The uploaded optimized scene data.
	sceneData *scene.Scene
}

// Create a new opencl tracer.
func NewTracer(id string, device *device.Device) (tracer.Tracer, error) {
	loggerName := fmt.Sprintf("opencl tracer (%s)", device.Name)

	tr := &clTracer{
		logger:       log.New(loggerName),
		device:       device,
		id:           id,
		blockReqChan: make(chan tracer.BlockRequest, 0),
		updateBuffer: make(map[tracer.UpdateType]interface{}, 0),
		stats:        &tracer.Stats{},
		speed:        device.Speed,
		pipeline:     make([]pipelineStage, 0),
	}

	return tr, nil
}

// Get tracer id.
func (tr *clTracer) Id() string {
	return tr.id
}

// Get tracer flags.
func (tr *clTracer) Flags() tracer.Flag {
	return tracer.Local | tracer.GLInterop
}

// Get the computation speed estimate (in GFlops).
func (tr *clTracer) Speed() uint32 {
	return tr.speed
}

// Initialize tracer
func (tr *clTracer) Init(frameW, frameH uint32, stages ...tracer.Stage) error {
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
	tr.resources, err = newDeviceResources(frameW, frameH, tr.device)
	if err != nil {
		tr.cleanup()
		return err
	}

	// Attach pipeline stages
	for _, stageFn := range stages {
		err = stageFn(tr)
		if err != nil {
			tr.cleanup()
			return err
		}
	}

	// Start worker
	if tr.closeChan == nil {
		tr.startWorker()
	}

	return nil
}

// Shutdown and cleanup tracer.
func (tr *clTracer) Close() {
	tr.Lock()
	defer tr.Unlock()

	tr.cleanup()
}

// Cleanup tracer. This method is meant to be called while holding tr.Lock()
func (tr *clTracer) cleanup() {
	// If the worker is running shut it down
	if tr.closeChan != nil {
		tr.closeChan <- struct{}{}

		// wait for worker to ack close and shutdown channel
		<-tr.closeChan
		close(tr.closeChan)
	}

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

// Enqueue block request.
func (tr *clTracer) Enqueue(blockReq tracer.BlockRequest) {
	select {
	case tr.blockReqChan <- blockReq:
	default:
		// drop the request if worker is not listening
		tr.logger.Error("request processor did not receive block request")
	}
}

// Append a change to the tracer's update buffer.
func (tr *clTracer) Update(updateType tracer.UpdateType, data interface{}) {
	tr.updateBuffer[updateType] = data
}

// Retrieve last frame statistics.
func (tr *clTracer) Stats() *tracer.Stats {
	return tr.stats
}

// Upload scene data.
func (tr *clTracer) UploadSceneData(sc *scene.Scene) error {
	tr.sceneData = sc
	return tr.resources.buffers.UploadSceneData(sc)
}

// Upload camera data.
func (tr *clTracer) UploadCameraData(camera *scene.Camera) error {
	tr.sceneData.Camera = camera
	return nil
}

// Commit queued changes.
func (tr *clTracer) commitUpdates() error {
	var err error
	for updateType, data := range tr.updateBuffer {
		switch updateType {
		case tracer.UpdateScene:
			err = tr.UploadSceneData(data.(*scene.Scene))
		case tracer.UpdateCamera:
			err = tr.UploadCameraData(data.(*scene.Camera))
		default:
			return fmt.Errorf("unsupported update type %d", updateType)
		}

		if err != nil {
			return err
		}
	}

	tr.updateBuffer = make(map[tracer.UpdateType]interface{}, 0)
	return nil
}

// Spawn a go-routine to process block render requests.
func (tr *clTracer) startWorker() {
	// Worker already running
	if tr.closeChan != nil {
		return
	}

	readyChan := make(chan struct{}, 0)
	tr.wg.Add(1)
	go func() {
		defer tr.wg.Done()
		var blockReq tracer.BlockRequest
		var startTime time.Time
		var err error
		close(readyChan)
		for {
			select {
			case blockReq = <-tr.blockReqChan:

				// Apply any pending changes
				if len(tr.updateBuffer) != 0 {
					startTime = time.Now()
					err = tr.commitUpdates()
					if err != nil {
						blockReq.ErrChan <- err
						continue
					}
					tr.stats.UpdateTime = time.Since(startTime)
				}

				// Render block and reply with our completion status
				err = tr.renderBlock(&blockReq)
				if err != nil {
					blockReq.ErrChan <- err
					continue
				}

				// Update stats
				tr.stats.BlockH = blockReq.BlockH
				tr.stats.RenderTime = time.Since(startTime)

				blockReq.DoneChan <- blockReq.BlockH
			case <-tr.closeChan:
				// Ack close
				tr.closeChan <- struct{}{}
				return
			}
		}
	}()

	// Wait for go-routine to start
	<-readyChan
}

// Render block.
func (tr *clTracer) renderBlock(blockReq *tracer.BlockRequest) error {
	var err error

	if tr.sceneData == nil {
		return ErrNoSceneData
	}

	// Execute pipeline
	for _, stage := range tr.pipeline {
		_, err = stage(tr, blockReq)
		if err != nil {
			return err
		}
	}

	return nil
}
