package device

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/achilleasa/gopencl/v1.2/cl"
)

type Buffer struct {
	// Handle to opencl buffer.
	bufHandle cl.Mem

	// Associated Device.
	device *Device

	// A name for identifying the buffer.
	name string

	// Allocated size.
	size int
}

// Get buffer size.
func (b *Buffer) Size() int {
	return b.size
}

// Allocate a buffer with the given size and flags.
func (b *Buffer) Allocate(size int, flags cl.MemFlags) error {
	var errPtr *int32

	// If the buffer is alreay allocated release it
	b.Release()

	b.bufHandle = cl.CreateBuffer(
		*b.device.ctx,
		flags,
		cl.MemFlags(size),
		nil,
		errPtr,
	)

	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		return fmt.Errorf("opencl device (%s): could not allocate buffer %s of size %d (errCode %d)", b.device.Name, b.name, size, cl.ErrorCode(*errPtr))
	}

	b.size = size

	return nil
}

// Allocate a buffer with enough capacity to fit the given data.
func (b *Buffer) AllocateToFitData(data interface{}, flags cl.MemFlags) error {
	var errPtr *int32

	// If the buffer is alreay allocated release it
	b.Release()

	_, dataLen := getSliceData(data)

	b.bufHandle = cl.CreateBuffer(
		*b.device.ctx,
		flags,
		cl.MemFlags(dataLen),
		nil,
		errPtr,
	)

	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		return fmt.Errorf("opencl device (%s): could not allocate buffer %s of size %d (errCode %d)", b.device.Name, b.name, dataLen, cl.ErrorCode(*errPtr))
	}

	b.size = dataLen

	return nil
}

// Allocate a buffer with the given flags that is large enough to hold the given data
// and have opencl copy the data from the host pointer. The behavior of this method is
// undefined if a non-slice argument is passed or the argument does not use contiguous
// memory.
func (b *Buffer) AllocateAndWriteData(data interface{}, flags cl.MemFlags) error {
	var errPtr *int32

	// If the buffer is alreay allocated release it
	b.Release()

	dataPtr, dataLen := getSliceData(data)

	b.bufHandle = cl.CreateBuffer(
		*b.device.ctx,
		flags|cl.MEM_USE_HOST_PTR,
		cl.MemFlags(dataLen),
		dataPtr,
		errPtr,
	)

	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		return fmt.Errorf("opencl device (%s): could not allocate buffer %s of size %d (errCode %d)", b.device.Name, b.name, dataLen, cl.ErrorCode(*errPtr))
	}

	b.size = dataLen

	return nil
}

// Write data to the device buffer. The behavior of this method is undefined
// if a non-slice argument is passed or the argument does not use contiguous
// memory. A byte offset may also be specified to adjust the actual data copied.
func (b *Buffer) WriteData(data interface{}, offset int) error {

	dataPtr, dataLen := getSliceData(data)

	if dataLen > b.size {
		return fmt.Errorf("opencl device(%s): insufficient buffer space (%d) in %s for copying data of length %d", b.device.Name, b.size, b.name, dataLen)
	}

	errCode := cl.EnqueueWriteBuffer(
		b.device.cmdQueue,
		b.bufHandle,
		cl.TRUE,
		uint64(offset),
		uint64(dataLen-offset),
		dataPtr,
		0,
		nil,
		nil,
	)

	if errCode != cl.SUCCESS {
		return fmt.Errorf("opencl device(%s): error copying host data to device buffer %s (errCode %d)", b.device.Name, b.name, errCode)
	}

	return nil
}

// Read data from device buffer into the supplied host buffer. The behavior of
// this method is undefined if a non-slice argument is passed or if the argument
// does not use contiguous memory.
//
// If size is <= 0 then ReadData will read the entire bufer. Both src and dst
// offsets are specified in bytes.
func (b *Buffer) ReadData(srcOffset, dstOffset, size int, hostBuffer interface{}) error {
	if size <= 0 {
		size = b.size
	}

	dataPtr, _ := getSliceData(hostBuffer)

	errCode := cl.EnqueueReadBuffer(
		b.device.cmdQueue,
		b.bufHandle,
		cl.TRUE,
		uint64(srcOffset),
		uint64(size),
		unsafe.Pointer(uintptr(dataPtr)+uintptr(dstOffset)),
		0,
		nil,
		nil,
	)

	if errCode != cl.SUCCESS {
		return fmt.Errorf("opencl device(%s): error copying device data from %s to host buffer (errCode %d)", b.device.Name, b.name, errCode)
	}

	return nil
}

// Release buffer.
func (b *Buffer) Release() {
	if b.bufHandle != nil {
		cl.ReleaseMemObject(b.bufHandle)
		b.bufHandle = nil
	}
}

// Get opencl buffer handle.
func (b *Buffer) Handle() cl.Mem {
	return b.bufHandle
}

// Given an interface{} containing a slice return a pointer to its data and its length.
func getSliceData(data interface{}) (unsafe.Pointer, int) {
	reflVal := reflect.ValueOf(data)

	if reflVal.Kind() != reflect.Slice {
		panic("getSliceData: this function only supports slices")
	}

	sliceElemCount := reflVal.Len()
	if sliceElemCount == 0 {
		panic("getSliceData: supplied slice object is empty")
	}

	return unsafe.Pointer(reflVal.Index(0).Addr().Pointer()),
		sliceElemCount * int(reflect.TypeOf(data).Elem().Size())
}
