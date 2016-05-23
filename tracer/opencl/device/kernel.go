package device

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/achilleasa/gopencl/v1.2/cl"
)

// A wrapper around opencl kernelHandles.
type Kernel struct {
	device       *Device
	kernelHandle cl.Kernel
	name         string

	// kernelHandle workgroup sizes and offsets
	offsets         [2]uint64
	globalWorkSizes [2]uint64
	localWorkSizes  [2]uint64
}

// Free any allocated resources used by this kernel.
func (k *Kernel) Release() {
	if k.kernelHandle != nil {
		cl.ReleaseKernel(k.kernelHandle)
		k.kernelHandle = nil
	}
}

// Bind arguments to kernelHandle.
func (k *Kernel) SetArgs(args ...interface{}) error {
	var errCode cl.ErrorCode
	for argIndex, arg := range args {
		switch t := arg.(type) {
		case cl.Mem:
			errCode = cl.SetKernelArg(k.kernelHandle, uint32(argIndex), 8, unsafe.Pointer(&t))
		default:
			return fmt.Errorf(
				"opencl device (%s): could not set arg %d for kernelHandle %s; unsupported arg type: %s",
				k.device.Name,
				argIndex,
				k.name,
				reflect.TypeOf(t).Name(),
			)
		}

		if errCode != cl.SUCCESS {
			return fmt.Errorf(
				"opencl device (%s): could not set arg %d for kernelHandle %s (errCode %d)",
				k.device.Name,
				argIndex,
				k.name,
				errCode,
			)
		}
	}

	return nil
}

// Execute 1D kernelHandle.
func (k *Kernel) Exec1D(offset, globalWorkSize, localWorkSize int) (time.Duration, error) {
	var errCode cl.ErrorCode
	var offsetPtr *uint64 = nil

	// Setup work params
	if offset > 0 {
		k.offsets[0] = uint64(offset)
		offsetPtr = (*uint64)(unsafe.Pointer(&k.offsets[0]))
	}
	k.globalWorkSizes[0] = uint64(globalWorkSize)
	k.localWorkSizes[0] = uint64(localWorkSize)

	// Run kernelHandle
	tick := time.Now()
	errCode = cl.EnqueueNDRangeKernel(
		k.device.cmdQueue,
		k.kernelHandle,
		1,
		offsetPtr,
		(*uint64)(unsafe.Pointer(&k.globalWorkSizes[0])),
		(*uint64)(unsafe.Pointer(&k.localWorkSizes[0])),
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		return time.Duration(0), fmt.Errorf("opencl device (%s): unable to execute kernelHandle %s (errCode %d)", k.device.Name, k.name, errCode)
	}

	// Wait for the kernelHandle to complete
	errCode = cl.Finish(k.device.cmdQueue)
	if errCode != cl.SUCCESS {
		return time.Duration(0), fmt.Errorf("opencl device (%s): kernelHandle %s did not complete successfully (errCode %d)", k.device.Name, k.name, errCode)
	}

	return time.Since(tick), nil
}

// Execute 2D kernelHandle.
func (k *Kernel) Exec2D(offsetX, offsetY, globalWorkSizeX, globalWorkSizeY, localWorkSizeX, localWorkSizeY int) (time.Duration, error) {
	var errCode cl.ErrorCode
	var offsetPtr *uint64 = nil

	// Setup work params
	if offsetX > 0 || offsetY > 0 {
		k.offsets[0] = uint64(offsetX)
		k.offsets[1] = uint64(offsetY)
		offsetPtr = (*uint64)(unsafe.Pointer(&k.offsets[0]))
	}
	k.globalWorkSizes[0], k.globalWorkSizes[1] = uint64(globalWorkSizeX), uint64(globalWorkSizeY)
	k.localWorkSizes[0], k.localWorkSizes[1] = uint64(localWorkSizeX), uint64(localWorkSizeY)

	// Run kernelHandle
	tick := time.Now()
	errCode = cl.EnqueueNDRangeKernel(
		k.device.cmdQueue,
		k.kernelHandle,
		2,
		offsetPtr,
		(*uint64)(unsafe.Pointer(&k.globalWorkSizes[0])),
		(*uint64)(unsafe.Pointer(&k.localWorkSizes[0])),
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		return time.Duration(0), fmt.Errorf("opencl device (%s): unable to execute kernelHandle %s (errCode %d)", k.device.Name, k.name, errCode)
	}

	// Wait for the kernelHandle to complete
	errCode = cl.Finish(k.device.cmdQueue)
	if errCode != cl.SUCCESS {
		return time.Duration(0), fmt.Errorf("opencl device (%s): kernelHandle %s did not complete successfully (errCode %d)", k.device.Name, k.name, errCode)
	}

	return time.Since(tick), nil
}
