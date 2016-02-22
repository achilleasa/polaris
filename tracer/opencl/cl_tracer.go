package opencl

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"unsafe"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/hydroflame/gopencl/v1.2/cl"
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
	traceProgram cl.Program

	// A kernel that is run for each screen pixel.
	traceKernel cl.Kernel

	// Local opencl workgroup size.
	localWorkgroupSize uint64

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

	// Create command queue
	cq := cl.CreateCommandQueue(*ctx, device.Id, 0, errptr)
	if errptr != nil && cl.ErrorCode(*errptr) != cl.SUCCESS {
		cl.ReleaseContext(ctx)
		return nil, ErrCmdQueueCreationFailed
	}

	loggerPrefix := fmt.Sprintf("opencl tracer (%s): ", device.Name)
	return &clTracer{
		logger:       log.New(os.Stderr, loggerPrefix, log.LstdFlags),
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

	cl.ReleaseKernel(tr.traceKernel)
	cl.ReleaseProgram(tr.traceProgram)
	cl.ReleaseCommandQueue(tr.cmdQueue)
	cl.ReleaseContext(tr.ctx)
	tr.ctx = nil
	tr.renderTarget = nil
}

// Attach tracer to render target and start processing incoming block requests.
func (tr *clTracer) Attach(sc *scene.Scene, renderTarget []float32, frameW, frameH uint) error {
	tr.Lock()
	defer tr.Unlock()

	if tr.renderTarget != nil {
		return ErrAlreadyAttached
	}

	err := tr.setupKernel(sc)
	if err != nil {
		return err
	}

	readyChan := make(chan struct{}, 0)

	tr.renderTarget = renderTarget
	tr.wg.Add(1)
	go func() {
		defer tr.wg.Done()
		var blockReq tracer.BlockRequest
		close(readyChan)
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

	// Wait for worker goroutine to start
	<-readyChan
	return nil
}

// Enqueue block request.
func (tr *clTracer) Enqueue(blockReq tracer.BlockRequest) {
	select {
	case tr.blockReqChan <- blockReq:
	default:
		// drop the request if worker is not listening
	}
}

// Process block request.
func (tr *clTracer) process(blockReq tracer.BlockRequest) {
	// @todo
	fmt.Printf("ToDo: process block request (BlockY %d, %d rows)\n", blockReq.BlockY, blockReq.BlockH)
}

// Generate and compile the opencl kernel to be used by this tracer.
func (tr *clTracer) setupKernel(sc *scene.Scene) error {
	// Load kernel template
	templateFile, err := os.Open(tracerSourceFile)
	if err != nil {
		return err
	}
	defer templateFile.Close()

	template, err := ioutil.ReadAll(templateFile)
	if err != nil {
		return err
	}

	// @todo: process the scene and embed object and material properties
	// into the kernel using text/template

	// Create and build program
	var errPtr *int32
	programSrc := cl.Str(string(template) + "\x00")
	tr.traceProgram = cl.CreateProgramWithSource(*tr.ctx, 1, &programSrc, nil, errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		return ErrProgramCreationFailed
	}

	errCode := cl.BuildProgram(tr.traceProgram, 1, &tr.device.Id, nil, nil, nil)
	if errCode != cl.SUCCESS {
		var dataLen uint64
		data := make([]byte, 1024)

		cl.GetProgramBuildInfo(tr.traceProgram, tr.device.Id, cl.PROGRAM_BUILD_LOG, uint64(len(data)), unsafe.Pointer(&data[0]), &dataLen)
		tr.logger.Printf("Error building kernel (log follows):\n%s\n", string(data[0:dataLen-1]))
		cl.ReleaseProgram(tr.traceProgram)
		return ErrProgramBuildFailed
	}

	// Fetch kernel entrypoint and query global and local workgroup info
	tr.traceKernel = cl.CreateKernel(tr.traceProgram, cl.Str("tracePixel"+"\x00"), errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		cl.ReleaseProgram(tr.traceProgram)
		return ErrKernelCreationFailed
	}

	errCode = cl.GetKernelWorkGroupInfo(tr.traceKernel, tr.device.Id, cl.KERNEL_WORK_GROUP_SIZE, 8, unsafe.Pointer(&tr.localWorkgroupSize), nil)
	if errCode != cl.SUCCESS {
		cl.ReleaseKernel(tr.traceKernel)
		cl.ReleaseProgram(tr.traceProgram)
		return ErrGettingWorkgroupInfo
	}
	return nil
}
