package device

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/achilleasa/go-pathtrace/types"
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
		// We can't use the captured type from the switch
		// like switch t := arg.(type) as we get back an
		// interface and we need to obtain a pointer to the underlying data.
		switch arg.(type) {
		case *Buffer:
			bufHandle := arg.(*Buffer).Handle()
			errCode = cl.SetKernelArg(k.kernelHandle, uint32(argIndex), 8, unsafe.Pointer(&bufHandle))
		case int32:
			v := arg.(int32)
			errCode = cl.SetKernelArg(k.kernelHandle, uint32(argIndex), 4, unsafe.Pointer(&v))
		case uint32:
			v := arg.(uint32)
			errCode = cl.SetKernelArg(k.kernelHandle, uint32(argIndex), 4, unsafe.Pointer(&v))
		case float32:
			v := arg.(float32)
			errCode = cl.SetKernelArg(k.kernelHandle, uint32(argIndex), 4, unsafe.Pointer(&v))
		case types.Vec2:
			v := arg.(types.Vec2)
			errCode = cl.SetKernelArg(k.kernelHandle, uint32(argIndex), 8, unsafe.Pointer(&v[0]))
		case types.Vec3:
			v := arg.(types.Vec3)
			errCode = cl.SetKernelArg(k.kernelHandle, uint32(argIndex), 12, unsafe.Pointer(&v[0]))
		case types.Vec4:
			v := arg.(types.Vec4)
			errCode = cl.SetKernelArg(k.kernelHandle, uint32(argIndex), 16, unsafe.Pointer(&v[0]))
		default:
			return fmt.Errorf(
				"opencl device (%s): could not set arg %d for kernelHandle %s; unsupported arg type: %s",
				k.device.Name,
				argIndex,
				k.name,
				reflect.TypeOf(arg).Name(),
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

// Execute 1D kernelHandle. If localWorSize is equal to 0 then the opencl implementation
// will pick the optimal worksize split for the underlying hardware.
func (k *Kernel) Exec1D(offset, globalWorkSize, localWorkSize int) (time.Duration, error) {
	var errCode cl.ErrorCode
	var offsetPtr *uint64 = nil
	var localSizePtr *uint64 = nil

	// Setup work params
	if offset > 0 {
		k.offsets[0] = uint64(offset)
		offsetPtr = (*uint64)(unsafe.Pointer(&k.offsets[0]))
	}
	k.globalWorkSizes[0] = uint64(globalWorkSize)
	if localWorkSize != 0 {
		k.localWorkSizes[0] = uint64(localWorkSize)

		localSizePtr = (*uint64)(unsafe.Pointer(&k.localWorkSizes[0]))
	}

	// Run kernelHandle
	tick := time.Now()
	errCode = cl.EnqueueNDRangeKernel(
		k.device.cmdQueue,
		k.kernelHandle,
		1,
		offsetPtr,
		(*uint64)(unsafe.Pointer(&k.globalWorkSizes[0])),
		localSizePtr,
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		return time.Duration(0), fmt.Errorf("opencl device (%s): unable to execute kernel %s (errCode %d)", k.device.Name, k.name, errCode)
	}

	// Wait for the kernelHandle to complete
	errCode = cl.Finish(k.device.cmdQueue)
	if errCode != cl.SUCCESS {
		return time.Duration(0), fmt.Errorf("opencl device (%s): kernel %s did not complete successfully (errCode %d)", k.device.Name, k.name, errCode)
	}

	return time.Since(tick), nil
}

// Execute 2D kernelHandle. If both localWorkSizeX and localWorkSizeY are 0 then the opencl implementation
// will pick the optimal local worksize split for the underlying hardware.
func (k *Kernel) Exec2D(offsetX, offsetY, globalWorkSizeX, globalWorkSizeY, localWorkSizeX, localWorkSizeY int) (time.Duration, error) {
	var errCode cl.ErrorCode
	var offsetPtr *uint64 = nil
	var localSizePtr *uint64 = nil

	// Setup work params
	if offsetX > 0 || offsetY > 0 {
		k.offsets[0] = uint64(offsetX)
		k.offsets[1] = uint64(offsetY)
		offsetPtr = (*uint64)(unsafe.Pointer(&k.offsets[0]))
	}
	k.globalWorkSizes[0], k.globalWorkSizes[1] = uint64(globalWorkSizeX), uint64(globalWorkSizeY)
	if localWorkSizeX != 0 && localWorkSizeY != 0 {
		k.localWorkSizes[0], k.localWorkSizes[1] = uint64(localWorkSizeX), uint64(localWorkSizeY)
		localSizePtr = (*uint64)(unsafe.Pointer(&k.localWorkSizes[0]))
	}

	// Run kernelHandle
	tick := time.Now()
	errCode = cl.EnqueueNDRangeKernel(
		k.device.cmdQueue,
		k.kernelHandle,
		2,
		offsetPtr,
		(*uint64)(unsafe.Pointer(&k.globalWorkSizes[0])),
		localSizePtr,
		0,
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		return time.Duration(0), fmt.Errorf("opencl device (%s): unable to execute kernel %s (errCode %d)", k.device.Name, k.name, errCode)
	}

	// Wait for the kernelHandle to complete
	errCode = cl.Finish(k.device.cmdQueue)
	if errCode != cl.SUCCESS {
		return time.Duration(0), fmt.Errorf("opencl device (%s): kernel %s did not complete successfully (errCode %d)", k.device.Name, k.name, errCode)
	}

	return time.Since(tick), nil
}
