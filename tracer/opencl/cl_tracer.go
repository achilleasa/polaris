package opencl

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/gopencl/v1.2/cl"
)

const (
	tracerSourceFile = "tracer/opencl/cl_tracer.cl"
)

type clTracer struct {
	sync.Mutex
	wg sync.WaitGroup

	logger *log.Logger

	// The tracer's id.
	id string

	// Opencl device used by this tracer.
	device Device

	// Opencl context.
	ctx *cl.Context

	// Opencl command queue
	cmdQueue cl.CommandQueue

	// The tracer opencl program.
	program cl.Program

	// Host buffers.
	hostAccumBuffer []float32
	hostFrameBuffer []uint8

	// Commit buffer for syncing scene changes with the local scene copy.
	updateBuffer map[tracer.ChangeType]interface{}

	// The output frame dimensions.
	frameW uint32
	frameH uint32

	// A channel for receiving block requests from the renderer.
	blockReqChan chan tracer.BlockRequest

	// A channel for signaling the worker to exit.
	closeChan chan struct{}

	// Statistics for last rendered frame
	stats *tracer.Stats
}

// Create a new opencl tracer.
func newTracer(id string, device Device) (*clTracer, error) {
	loggerPrefix := fmt.Sprintf("opencl tracer (%s): ", device.Name)
	tr := &clTracer{
		logger:       log.New(os.Stderr, loggerPrefix, log.LstdFlags),
		id:           id,
		device:       device,
		blockReqChan: make(chan tracer.BlockRequest, 0),
		closeChan:    make(chan struct{}, 0),
		updateBuffer: make(map[tracer.ChangeType]interface{}, 0),
		stats:        &tracer.Stats{},
	}

	return tr, nil
}

// Get tracer id.
func (tr *clTracer) Id() string {
	return tr.id
}

// Shutdown and cleanup tracer.
func (tr *clTracer) Close() {
	tr.Lock()
	defer tr.Unlock()

	tr.cleanup()
}

// Get speed estimate.
func (tr *clTracer) SpeedEstimate() float32 {
	return tr.device.SpeedEstimate()
}

// Setup the tracer.
func (tr *clTracer) Setup(frameW, frameH uint32, accumBuffer []float32, frameBuffer []uint8) error {
	tr.Lock()
	defer tr.Unlock()

	if tr.ctx != nil {
		return ErrAlreadySetup
	}

	tr.frameW = frameW
	tr.frameH = frameH
	tr.hostAccumBuffer = accumBuffer
	tr.hostFrameBuffer = frameBuffer

	// Start worker
	tr.startWorker()
	return nil
}

// Enqueue block request.
func (tr *clTracer) Enqueue(blockReq tracer.BlockRequest) {
	select {
	case tr.blockReqChan <- blockReq:
	default:
		// drop the request if worker is not listening
		tr.logger.Printf("request processor did not receive block request")
	}
}

// Append a change to the tracer's update buffer.
func (tr *clTracer) AppendChange(changeType tracer.ChangeType, data interface{}) {
	tr.updateBuffer[changeType] = data
}

// Apply all pending changes from the update buffer.
func (tr *clTracer) ApplyPendingChanges() error {
	return fmt.Errorf("cl_tracer: ApplyPendingChanges() not implemented")
}

// Retrieve last frame statistics.
func (tr *clTracer) Stats() *tracer.Stats {
	return tr.stats
}

// Free tracer resources.
func (tr *clTracer) cleanup() {
	if tr.ctx == nil {
		return
	}

	// Signal worker to exit and wait till it exits
	close(tr.closeChan)
	tr.wg.Wait()

	if tr.program != nil {
		cl.ReleaseProgram(tr.program)
		tr.program = nil
	}
	if tr.cmdQueue != nil {
		cl.ReleaseCommandQueue(tr.cmdQueue)
		tr.cmdQueue = nil
	}
	if tr.ctx != nil {
		cl.ReleaseContext(tr.ctx)
		tr.ctx = nil
	}
}

// Process block request.
func (tr *clTracer) process(blockReq tracer.BlockRequest) error {
	return fmt.Errorf("cl_tracer: process() method not implemented")
}

// Spawn a go-routine to process block render requests.
func (tr *clTracer) startWorker() {
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

				startTime = time.Now()

				// commit pending changes
				err = tr.ApplyPendingChanges()
				if err != nil {
					blockReq.ErrChan <- err
					continue
				}

				// Render block and reply with our completion status
				err = tr.process(blockReq)
				if err != nil {
					blockReq.ErrChan <- err
					continue
				}

				// Update stats
				tr.stats.BlockH = blockReq.BlockH
				tr.stats.BlockTime = time.Since(startTime).Nanoseconds()

				blockReq.DoneChan <- blockReq.BlockH
			case <-tr.closeChan:
				return
			}
		}
	}()

	// Wait for go-routine to start
	<-readyChan
}
