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

	// Device buffers where the kernel writes its output and frustrum corners.
	traceOutput     cl.Mem
	frustrumCorners cl.Mem

	// The scene to be rendered.
	scene *scene.Scene

	// The output frame dimensions.
	frameW uint
	frameH uint

	// A channel for receiving block requests from the renderer.
	blockReqChan chan tracer.BlockRequest

	// A channel for signaling the worker to exit.
	closeChan chan struct{}
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
	// Lock tracer and perform cleanup
	tr.cleanup(true)
}

// Attach tracer to render target and start processing incoming block requests.
func (tr *clTracer) Setup(sc *scene.Scene, frameW, frameH uint) error {
	tr.Lock()
	defer tr.Unlock()

	if tr.traceKernel != nil {
		return ErrAlreadyAttached
	}

	err := tr.setupKernel(sc, frameW, frameH)
	if err != nil {
		return err
	}

	// Save scene and frame dims
	tr.scene = sc
	tr.frameW = frameW
	tr.frameH = frameH

	readyChan := make(chan struct{}, 0)
	tr.wg.Add(1)
	go func() {
		defer tr.wg.Done()
		var blockReq tracer.BlockRequest
		var err error
		close(readyChan)
		for {
			select {
			case blockReq = <-tr.blockReqChan:
				// Render block and reply with our completion status
				err = tr.process(blockReq)
				if err != nil {
					blockReq.ErrChan <- err
					continue
				}
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

// Cleanup tracer resources optionally using a lock.
func (tr *clTracer) cleanup(useLock bool) {
	if useLock {
		tr.Lock()
		defer tr.Unlock()
	}

	if tr.ctx == nil {
		return
	}

	// Signal worker to exit and wait till it exits
	close(tr.closeChan)
	tr.wg.Wait()

	if tr.traceOutput != nil {
		cl.ReleaseMemObject(tr.traceOutput)
		tr.traceOutput = nil
	}
	if tr.frustrumCorners != nil {
		cl.ReleaseMemObject(tr.frustrumCorners)
		tr.frustrumCorners = nil
	}
	if tr.traceKernel != nil {
		cl.ReleaseKernel(tr.traceKernel)
		tr.traceKernel = nil
	}
	if tr.traceProgram != nil {
		cl.ReleaseProgram(tr.traceProgram)
		tr.traceProgram = nil
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
	// Copy camera frustrum corners to allocated buffer.
	errCode := cl.EnqueueWriteBuffer(
		tr.cmdQueue,
		tr.frustrumCorners,
		cl.TRUE,
		0,
		4*16,
		unsafe.Pointer(&tr.scene.Camera.Frustrum[0]),
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		tr.logger.Printf("Failed to write frustrum corner data")
		return ErrCopyingDataToDevice
	}

	// Set kernel params
	errCode = cl.SetKernelArg(tr.traceKernel, 0, 8, unsafe.Pointer(&tr.traceOutput))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("Failed to write kernel arg 0")
		return ErrSettingKernelArguments
	}
	errCode = cl.SetKernelArg(tr.traceKernel, 1, 8, unsafe.Pointer(&tr.frustrumCorners))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("Failed to write kernel arg 1")
		return ErrSettingKernelArguments
	}
	errCode = cl.SetKernelArg(tr.traceKernel, 2, 4, unsafe.Pointer(&blockReq.BlockY))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("Failed to write kernel arg 2")
		return ErrSettingKernelArguments
	}
	errCode = cl.SetKernelArg(tr.traceKernel, 3, 4, unsafe.Pointer(&blockReq.SamplesPerPixel))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("Failed to write kernel arg 3")
		return ErrSettingKernelArguments
	}
	errCode = cl.SetKernelArg(tr.traceKernel, 4, 4, unsafe.Pointer(&blockReq.Exposure))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("Failed to write kernel arg 4")
		return ErrSettingKernelArguments
	}

	// Execute kernel
	workOffset := []uint64{0, uint64(blockReq.BlockY)}
	workSize := []uint64{uint64(tr.frameW), uint64(blockReq.BlockH)}
	errCode = cl.EnqueueNDRangeKernel(
		tr.cmdQueue,
		tr.traceKernel,
		2,
		(*uint64)(unsafe.Pointer(&workOffset[0])),
		(*uint64)(unsafe.Pointer(&workSize[0])),
		nil,
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		return ErrKernelExecutionFailed
	}

	// Wait for the kernel to finish executing
	cl.Finish(tr.cmdQueue)

	// Copy the rendered block from device buffer to the render target
	readOffset := uint64(tr.frameW * 4 * 4 * blockReq.BlockY)
	blockSizeBytes := uint64(tr.frameW * 4 * 4 * blockReq.BlockH)
	errCode = cl.EnqueueReadBuffer(
		tr.cmdQueue,
		tr.traceOutput,
		cl.TRUE,
		readOffset,     // start at beginning of block
		blockSizeBytes, // read just the block data
		unsafe.Pointer(&blockReq.RenderTarget[readOffset]),
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		return ErrCopyingDataToHost
	}
	return nil
}

// Generate and compile the opencl kernel to be used by this tracer.
func (tr *clTracer) setupKernel(sc *scene.Scene, frameW, frameH uint) error {
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
		tr.cleanup(false)
		return ErrProgramBuildFailed
	}

	// Fetch kernel entrypoint and query global and local workgroup info
	tr.traceKernel = cl.CreateKernel(tr.traceProgram, cl.Str("tracePixel"+"\x00"), errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		tr.cleanup(false)
		return ErrKernelCreationFailed
	}

	// Allocate an output buffer large enough to fit a full frame even
	// though it will never be fully used if more than one tracers are used.
	tr.traceOutput = cl.CreateBuffer(*tr.ctx, cl.MEM_WRITE_ONLY, cl.MemFlags(4*4*frameW*frameH), nil, errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		tr.cleanup(false)
		return ErrAllocatingBuffers
	}

	// Allocate buffer for passing frustrum corners (4 x Vec4 = 64 bytes)
	tr.frustrumCorners = cl.CreateBuffer(*tr.ctx, cl.MEM_READ_ONLY, 4*4*4, nil, errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		tr.cleanup(false)
		return ErrAllocatingBuffers
	}

	return nil
}
