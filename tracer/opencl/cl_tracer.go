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

	// A kernel that is run for each screen pixel.
	kernel cl.Kernel

	// Device buffers.
	frameBuffer     cl.Mem
	frustrumCorners cl.Mem
	materials       cl.Mem
	bvhNodes        cl.Mem
	primitives      cl.Mem
	emissiveIndices cl.Mem

	// Commit buffer for syncing scene changes with the local scene copy.
	updateBuffer map[tracer.ChangeType]interface{}

	// The output frame dimensions.
	frameW uint32
	frameH uint32

	// A channel for receiving block requests from the renderer.
	blockReqChan chan tracer.BlockRequest

	// A channel for signaling the worker to exit.
	closeChan chan struct{}
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
func (tr *clTracer) Setup(frameW, frameH uint32) error {
	tr.Lock()
	defer tr.Unlock()

	if tr.ctx != nil {
		return ErrAlreadySetup
	}

	tr.frameW = frameW
	tr.frameH = frameH

	// Init kernel
	err := tr.setupKernel()
	if err != nil {
		return err
	}

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
	var err error
	var hostData unsafe.Pointer
	var deviceBuffer *cl.Mem
	var deviceBufferIsImage bool
	var sizeInBytes uint64
	var kernelArgName string
	var kernelArgIndex uint32
	for changeType, changeData := range tr.updateBuffer {
		hostData = nil

		switch changeType {
		case tracer.SetMaterials:
			kernelArgName = "materials"
			kernelArgIndex = 2
			if data, valid := changeData.([]scene.Material); valid {
				hostData = unsafe.Pointer(&data[0])
				sizeInBytes = uint64(len(data)) * uint64(unsafe.Sizeof(&data[0]))
				deviceBuffer = &tr.materials
				deviceBufferIsImage = true
			} else {
				err = ErrInvalidChangeData
			}
		case tracer.SetBvhNodes:
			kernelArgName = "bvhNodes"
			kernelArgIndex = 3
			if data, valid := changeData.([]scene.BvhNode); valid {
				hostData = unsafe.Pointer(&data[0])
				sizeInBytes = uint64(len(data)) * uint64(unsafe.Sizeof(&data[0]))
				deviceBuffer = &tr.bvhNodes
				deviceBufferIsImage = true
			} else {
				err = ErrInvalidChangeData
			}
		case tracer.SetPrimitivies:
			kernelArgName = "primitives"
			kernelArgIndex = 4
			if data, valid := changeData.([]scene.Primitive); valid {
				hostData = unsafe.Pointer(&data[0])
				sizeInBytes = uint64(len(data)) * uint64(unsafe.Sizeof(&data[0]))
				deviceBuffer = &tr.primitives
				deviceBufferIsImage = true
			} else {
				err = ErrInvalidChangeData
			}
		case tracer.SetEmissiveLightIndices:
			kernelArgName = "emissiveLightIndices"
			kernelArgIndex = 5
			if data, valid := changeData.([]uint32); valid {
				// Update count
				numIndices := len(data)
				errCode := cl.SetKernelArg(tr.kernel, 6, 4, unsafe.Pointer(&numIndices))
				if errCode != cl.SUCCESS {
					tr.logger.Printf("error %d setting kernel arg 6 (numEmissiveIndices)", errCode)
					return ErrSettingKernelArgument
				}

				// Update indices
				hostData = unsafe.Pointer(&data[0])
				sizeInBytes = uint64(len(data)) * uint64(unsafe.Sizeof(&data[0]))
				deviceBuffer = &tr.emissiveIndices
				deviceBufferIsImage = false
			} else {
				err = ErrInvalidChangeData
			}
		case tracer.UpdateCamera:
			kernelArgName = "frustrumCorners"
			kernelArgIndex = 1
			if data, valid := changeData.(*scene.Camera); valid {
				// Update eye position
				eyePos := data.Position()
				errCode := cl.SetKernelArg(tr.kernel, 7, 16, unsafe.Pointer(&eyePos[0]))
				if errCode != cl.SUCCESS {
					tr.logger.Printf("error %d setting kernel arg 7 (eyePos)", errCode)
					return ErrSettingKernelArgument
				}

				// Update frustrum corners
				frustrum := data.Frustrum
				hostData = unsafe.Pointer(&frustrum[0])
				sizeInBytes = 64
				deviceBuffer = &tr.frustrumCorners
				deviceBufferIsImage = false
			} else {
				err = ErrInvalidChangeData
			}

		default:
			tr.logger.Printf("unsupported change type %d", changeType)
			return ErrUnsupportedChangeType
		}

		if err == ErrInvalidChangeData {
			tr.logger.Printf("invalid change data for kernel argument %d (%s)", kernelArgIndex, kernelArgName)
		}

		if deviceBufferIsImage {
			err = tr.uploadDataAsImage(hostData, sizeInBytes, deviceBuffer, kernelArgIndex, kernelArgName)
		} else {
			err = tr.uploadDataAsBuffer(hostData, sizeInBytes, deviceBuffer, kernelArgIndex, kernelArgName)
		}

		if err != nil {
			return err
		}
	}

	tr.updateBuffer = make(map[tracer.ChangeType]interface{}, 0)
	return nil
}

// Sync host data buffer with opencl device buffer. If the device buffer is not
// initialized or the host data exceeds the device buffer's capacity, a new
// buffer will be initialized and the data will get copied to it. Data is stored
// on the device as a buffer.
//
// This method will also attach the device buffer as a kernel argument.
func (tr *clTracer) uploadDataAsBuffer(hostData unsafe.Pointer, hostDataLen uint64, deviceBuffer *cl.Mem, kernelArgIndex uint32, kernelArgName string) error {
	if hostData == nil {
		return nil
	}

	var errCode cl.ErrorCode

	if *deviceBuffer != nil {
		var curDataLen uint64
		errCode = cl.GetMemObjectInfo(*deviceBuffer, cl.MEM_SIZE, uint64(8), unsafe.Pointer(&curDataLen), nil)
		if errCode != cl.SUCCESS {
			tr.logger.Printf("error %d while querying buffer size for %s arg", errCode, kernelArgName)
			return ErrCopyingDataToDevice
		}

		// If existing buffer is not large enough we need to reallocate it
		if curDataLen < hostDataLen {
			cl.ReleaseMemObject(*deviceBuffer)
			*deviceBuffer = nil
		}
	}

	// If buffer is not yet allocated, allocate it now, copy data and attach to kernel
	if *deviceBuffer == nil {
		var errPtr *int32

		*deviceBuffer = cl.CreateBuffer(
			*tr.ctx,
			cl.MEM_READ_ONLY|cl.MEM_HOST_WRITE_ONLY|cl.MEM_COPY_HOST_PTR,
			cl.MemFlags(hostDataLen),
			hostData,
			errPtr,
		)
		if *deviceBuffer == nil || (errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS) {
			tr.logger.Printf("error %d allocating device buffer for kernel arg %d (%s)", cl.ErrorCode(*errPtr), kernelArgIndex, kernelArgName)
			return ErrAllocatingBuffer
		}

		errCode = cl.SetKernelArg(tr.kernel, kernelArgIndex, 8, unsafe.Pointer(deviceBuffer))
		if errCode != cl.SUCCESS {
			tr.logger.Printf("error %d setting kernel arg %d (%s)", errCode, kernelArgIndex, kernelArgName)
			return ErrSettingKernelArgument
		}

		return nil
	}

	// New data fits in current buffer; copy data over
	errCode = cl.EnqueueWriteBuffer(
		tr.cmdQueue,
		*deviceBuffer,
		cl.TRUE,
		0,
		hostDataLen,
		hostData,
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		tr.logger.Printf("error %d copying data for kernel arg %d (%s)", errCode, kernelArgIndex, kernelArgName)
		return ErrCopyingDataToDevice
	}

	return nil
}

func (tr *clTracer) uploadDataAsImage(hostData unsafe.Pointer, hostDataLen uint64, deviceBuffer *cl.Mem, kernelArgIndex uint32, kernelArgName string) error {
	if hostData == nil {
		return nil
	}

	// Calculate image height. Image width is capped to 8192 pixels which
	// is the mimimum width supported by all opencl 1.2 implementations.
	imageH := (hostDataLen >> 13) + 1

	var errCode cl.ErrorCode

	if *deviceBuffer != nil {
		var curHeight uint64
		errCode = cl.GetImageInfo(*deviceBuffer, cl.IMAGE_HEIGHT, uint64(8), unsafe.Pointer(&curHeight), nil)
		if errCode != cl.SUCCESS {
			tr.logger.Printf("error %d while querying image height for %s arg", errCode, kernelArgName)
			return ErrCopyingDataToDevice
		}

		// If existing buffer is not large enough we need to reallocate it
		if curHeight < imageH {
			cl.ReleaseMemObject(*deviceBuffer)
			*deviceBuffer = nil
		}
	}

	// If buffer is not yet allocated, allocate it now, copy data and attach to kernel
	if *deviceBuffer == nil {
		var errPtr *int32

		*deviceBuffer = cl.CreateImage(
			*tr.ctx,
			cl.MEM_READ_ONLY|cl.MEM_HOST_WRITE_ONLY|cl.MEM_COPY_HOST_PTR,
			cl.ImageFormat{cl.RGBA, cl.FLOAT}, // 16 bytes per pixel
			cl.ImageDesc{
				ImageType:   cl.MEM_OBJECT_IMAGE2D,
				ImageWidth:  8192,
				ImageHeight: imageH,
			},
			// The buffer will probably contain less than W*H bytes
			// so opencl will also copy some junk from memory addresses
			// below the buffer. Ideally, we should pad our data so it fills
			// the last image row but as we don't access it anyway we can
			// skip that.
			hostData,
			errPtr,
		)
		if *deviceBuffer == nil || (errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS) {
			tr.logger.Printf("error %d allocating device buffer for kernel arg %d (%s)", cl.ErrorCode(*errPtr), kernelArgIndex, kernelArgName)
			return ErrAllocatingBuffer
		}

		errCode = cl.SetKernelArg(tr.kernel, kernelArgIndex, 8, unsafe.Pointer(deviceBuffer))
		if errCode != cl.SUCCESS {
			tr.logger.Printf("error %d setting kernel arg %d (%s)", errCode, kernelArgIndex, kernelArgName)
			return ErrSettingKernelArgument
		}

		return nil
	}

	// New data fits in current buffer; copy data over
	origin := [3]uint64{0, 0, 0}
	region := [3]uint64{8192, imageH, 1}
	errCode = cl.EnqueueWriteImage(
		tr.cmdQueue,
		*deviceBuffer,
		cl.TRUE,
		&origin[0],
		&region[0],
		0,
		0,
		hostData,
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		tr.logger.Printf("error %d copying data for kernel arg %d (%s)", errCode, kernelArgIndex, kernelArgName)
		return ErrCopyingDataToDevice
	}

	return nil
}

// Free tracer resources.
func (tr *clTracer) cleanup() {
	if tr.ctx == nil {
		return
	}

	// Signal worker to exit and wait till it exits
	close(tr.closeChan)
	tr.wg.Wait()

	if tr.emissiveIndices != nil {
		cl.ReleaseMemObject(tr.emissiveIndices)
		tr.emissiveIndices = nil
	}
	if tr.primitives != nil {
		cl.ReleaseMemObject(tr.primitives)
		tr.primitives = nil
	}
	if tr.bvhNodes != nil {
		cl.ReleaseMemObject(tr.bvhNodes)
		tr.bvhNodes = nil
	}
	if tr.materials != nil {
		cl.ReleaseMemObject(tr.materials)
		tr.materials = nil
	}
	if tr.frustrumCorners != nil {
		cl.ReleaseMemObject(tr.frustrumCorners)
		tr.frustrumCorners = nil
	}
	if tr.frameBuffer != nil {
		cl.ReleaseMemObject(tr.frameBuffer)
		tr.frameBuffer = nil
	}
	if tr.kernel != nil {
		cl.ReleaseKernel(tr.kernel)
		tr.kernel = nil
	}
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
	var errCode cl.ErrorCode

	// Set variable kernel params
	errCode = cl.SetKernelArg(tr.kernel, 8, 4, unsafe.Pointer(&blockReq.BlockY))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("error %d setting kernel arg 8 (blockY)", errCode)
		return ErrSettingKernelArgument
	}
	errCode = cl.SetKernelArg(tr.kernel, 9, 4, unsafe.Pointer(&blockReq.SamplesPerPixel))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("error %d setting kernel arg 9 (samplesPerPixel)", errCode)
		return ErrSettingKernelArgument
	}
	errCode = cl.SetKernelArg(tr.kernel, 10, 4, unsafe.Pointer(&blockReq.Exposure))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("error %d setting kernel arg 10 (exposure)", errCode)
		return ErrSettingKernelArgument
	}
	errCode = cl.SetKernelArg(tr.kernel, 11, 4, unsafe.Pointer(&blockReq.Seed))
	if errCode != cl.SUCCESS {
		tr.logger.Printf("error %d setting kernel arg 11 (seed)", errCode)
		return ErrSettingKernelArgument
	}

	// Execute kernel
	workOffset := []uint64{0, uint64(blockReq.BlockY)}
	workSize := []uint64{uint64(tr.frameW), uint64(blockReq.BlockH)}
	errCode = cl.EnqueueNDRangeKernel(
		tr.cmdQueue,
		tr.kernel,
		2,
		(*uint64)(unsafe.Pointer(&workOffset[0])),
		(*uint64)(unsafe.Pointer(&workSize[0])),
		nil,
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		tr.logger.Printf("error %d while requesting kernel execution", errCode)
		return ErrKernelExecutionFailed
	}

	// Wait for the kernel to finish executing
	errCode = cl.Finish(tr.cmdQueue)
	if errCode != cl.SUCCESS {
		tr.logger.Printf("error %d while waiting for kernel to finish executing", errCode)
		return ErrKernelExecutionFailed
	}

	// Copy the rendered block from device buffer to the render target
	readOffset := uint64(tr.frameW * 4 * 4 * blockReq.BlockY)
	blockSizeBytes := uint64(tr.frameW * 4 * 4 * blockReq.BlockH)
	errCode = cl.EnqueueReadBuffer(
		tr.cmdQueue,
		tr.frameBuffer,
		cl.TRUE,
		readOffset,     // start at beginning of block
		blockSizeBytes, // read just the block data
		// target is []float32 so we need to divide offset by 4
		unsafe.Pointer(&blockReq.RenderTarget[readOffset>>2]),
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		tr.logger.Printf("Error copying data to host: (blockY: %d, blockH: %d, readOffset: %d, bytes: %d, code %d)", blockReq.BlockY, blockReq.BlockH, readOffset, blockSizeBytes, errCode)
		return ErrCopyingDataToHost
	}

	return nil
}

// Spawn a go-routine to process block render requests.
func (tr *clTracer) startWorker() {
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
				blockReq.DoneChan <- blockReq.BlockH
			case <-tr.closeChan:
				return
			}
		}
	}()

	// Wait for go-routine to start
	<-readyChan
}

// Generate and compile the opencl kernel to be used by this tracer.
func (tr *clTracer) setupKernel() error {
	var errPtr *int32

	// Create context
	tr.ctx = cl.CreateContext(nil, 1, &tr.device.Id, nil, nil, errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		return ErrContextCreationFailed
	}

	// Create command queue
	tr.cmdQueue = cl.CreateCommandQueue(*tr.ctx, tr.device.Id, 0, errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		return ErrCmdQueueCreationFailed
	}

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

	// Create and build program
	programSrc := cl.Str(string(template) + "\x00")
	tr.program = cl.CreateProgramWithSource(*tr.ctx, 1, &programSrc, nil, errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		return ErrProgramCreationFailed
	}

	programDefs := cl.Str(fmt.Sprintf("-D FRAME_W=%d -D FRAME_H=%d\x00", tr.frameW, tr.frameH))
	errCode := cl.BuildProgram(tr.program, 1, &tr.device.Id, programDefs, nil, nil)
	if errCode != cl.SUCCESS {
		var dataLen uint64
		data := make([]byte, 120000)

		cl.GetProgramBuildInfo(tr.program, tr.device.Id, cl.PROGRAM_BUILD_LOG, uint64(len(data)), unsafe.Pointer(&data[0]), &dataLen)
		tr.logger.Printf("Error building kernel (log follows):\n%s\n", string(data[0:dataLen-1]))
		tr.cleanup()
		return ErrProgramBuildFailed
	}

	// Fetch kernel entrypoint and query global and local workgroup info
	tr.kernel = cl.CreateKernel(tr.program, cl.Str("tracePixel"+"\x00"), errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		tr.cleanup()
		return ErrKernelCreationFailed
	}

	// Allocate an output buffer large enough to fit a full frame even
	// though it will never be fully used if more than one tracers are used.
	tr.frameBuffer = cl.CreateBuffer(*tr.ctx, cl.MEM_WRITE_ONLY, cl.MemFlags(tr.frameW*tr.frameH*16), nil, errPtr)
	if tr.frameBuffer == nil || (errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS) {
		tr.cleanup()
		return ErrAllocatingBuffer
	}
	errCode = cl.SetKernelArg(tr.kernel, 0, 8, unsafe.Pointer(&tr.frameBuffer))
	if errCode != cl.SUCCESS {
		tr.cleanup()
		tr.logger.Printf("error %d setting kernel arg 0 (frameBuffer)", errCode)
		return ErrSettingKernelArgument
	}

	return nil
}
