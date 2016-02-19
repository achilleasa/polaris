package opencl

import (
	"fmt"
	"sync"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/hydroflame/gopencl/v1.2/cl"
)

type clTracer struct {
	sync.Mutex
	wg sync.WaitGroup

	// The tracer's id.
	id string

	// Opencl device used by this tracer.
	device Device

	// Opencl context.
	ctx *cl.Context

	// Opencl command queue
	cmdQueue cl.CommandQueue

	// A channel for receiving block requests from the renderer.
	blockReqChan chan tracer.BlockRequest

	// A channel for signaling the worker to exit.
	closeChan chan struct{}

	// A 4-component float32 buffer assigned to the tracer by the renderer.
	renderTarget []float32
}

// Create a new opencl tracer.
func newTracer(id string, device Device) (*clTracer, error) {
	var errptr *int32

	// Create context
	ctx := cl.CreateContext(nil, 1, &device.Id, nil, nil, errptr)
	if errptr != nil && cl.ErrorCode(*errptr) != cl.SUCCESS {
		return nil, ErrContextCreationFailed
	}

	//Create Command Queue
	cq := cl.CreateCommandQueue(*ctx, device.Id, 0, errptr)
	if errptr != nil && cl.ErrorCode(*errptr) != cl.SUCCESS {
		cl.ReleaseContext(ctx)
		return nil, ErrCmdQueueCreationFailed
	}

	return &clTracer{
		id:           id,
		device:       device,
		ctx:          ctx,
		cmdQueue:     cq,
		blockReqChan: make(chan tracer.BlockRequest, 0),
		closeChan:    make(chan struct{}, 0),
	}, nil

}

// Get tracer id.
func (tr *clTracer) Id() string {
	return tr.id
}

// Get speed estimate
func (tr *clTracer) SpeedEstimate() float32 {
	return tr.device.SpeedEstimate()
}

// Shutdown and cleanup tracer.
func (tr *clTracer) Close() {
	tr.Lock()
	defer tr.Unlock()

	if tr.ctx == nil {
		return
	}

	// Signal worker to exit and wait till it exits
	close(tr.closeChan)
	tr.wg.Wait()

	cl.ReleaseCommandQueue(tr.cmdQueue)
	cl.ReleaseContext(tr.ctx)
	tr.ctx = nil
	tr.renderTarget = nil
}

// Attach tracer to render target and start processing incoming block requests.
func (tr *clTracer) Attach(sc *scene.Scene, renderTarget []float32) error {
	tr.Lock()
	defer tr.Unlock()

	if tr.renderTarget != nil {
		return ErrAlreadyAttached
	}

	tr.renderTarget = renderTarget
	tr.wg.Add(1)
	go func() {
		defer tr.wg.Done()
		var blockReq tracer.BlockRequest
		for {
			select {
			case blockReq = <-tr.blockReqChan:
				// Render block and reply with our completion status
				tr.process(blockReq)
				blockReq.DoneChan <- blockReq.BlockH
			case <-tr.closeChan:
				return
			}
		}
	}()

	return nil
}

// Enqueue blcok request.
func (tr *clTracer) Enqueue(blockReq tracer.BlockRequest) {
	select {
	case tr.blockReqChan <- blockReq:
		return
	default:
		// drop the request if worker is not listening
	}
}

// Process block request.
func (tr *clTracer) process(blockReq tracer.BlockRequest) {
	// @todo
	fmt.Printf("ToDo: process block request (BlockY %d, %d rows)\n", blockReq.BlockY, blockReq.BlockH)
}
