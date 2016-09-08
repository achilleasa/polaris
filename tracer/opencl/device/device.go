package device

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"unsafe"

	"github.com/achilleasa/gopencl/v1.2/cl"
)

type DeviceType uint8

// Supported device types.
const (
	CpuDevice   DeviceType = 1 << iota
	GpuDevice              = 1 << iota
	OtherDevice            = 1 << iota
	AllDevices             = 0xFF
)

var (
	indentRegex = regexp.MustCompile("(?m)^")
)

func (dt DeviceType) String() string {
	switch dt {
	case CpuDevice:
		return "CPU"
	case GpuDevice:
		return "GPU"
	case OtherDevice:
		return "Other"
	}
	panic("opencl: unsupported device type")
}

// Wrapper around opencl-supported devices.
type Device struct {
	Name string
	Id   cl.DeviceId
	Type DeviceType

	compUnits  uint32
	clockSpeed uint32

	// Speed estimate in GFlops.
	Speed uint32

	// Opencl handles; allocated when device is initialized.
	ctx      *cl.Context
	cmdQueue cl.CommandQueue
	program  cl.Program
}

// A list of devices.
type DeviceList []Device

// Implements Stringer.
func (d Device) String() string {
	return fmt.Sprintf(
		"Name: %s\nType: %s\nSpecs: %d computation units, %d Mhz clock, %d GFlops approximate speed",
		d.Name,
		d.Type.String(),
		d.compUnits,
		d.clockSpeed,
		d.Speed,
	)
}

// Initialize device.
func (d *Device) Init(programFile string) error {
	var errCode cl.ErrorCode

	// Already initialized
	if d.ctx != nil {
		return nil
	}

	// Create context
	d.ctx = cl.CreateContext(nil, 1, &d.Id, nil, nil, (*int32)(&errCode))
	if errCode != cl.SUCCESS {
		defer d.Close()
		return fmt.Errorf("opencl device (%s): could not create opencl context (error: %s; code %d)", d.Name, ErrorName(errCode), errCode)
	}

	// Create command queue
	d.cmdQueue = cl.CreateCommandQueue(*d.ctx, d.Id, 0, (*int32)(&errCode))
	if errCode != cl.SUCCESS {
		defer d.Close()
		return fmt.Errorf("opencl device (%s): could not create opencl context (error: %s; code %d)", d.Name, ErrorName(errCode), errCode)
	}

	// Load program source
	absProgramPath, err := filepath.Abs(programFile)
	if err != nil {
		defer d.Close()
		return err
	}

	f, err := os.Open(absProgramPath)
	if err != nil {
		defer d.Close()
		return err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		defer d.Close()
		return err
	}
	progSrc := cl.Str(string(data) + "\x00")

	// Create and build program
	d.program = cl.CreateProgramWithSource(
		*d.ctx,
		1,
		&progSrc,
		nil,
		(*int32)(&errCode),
	)
	if errCode != cl.SUCCESS {
		defer d.Close()
		return fmt.Errorf("opencl device (%s): could not create program (error: %s; code %d)", d.Name, ErrorName(errCode), errCode)
	}

	errCode = cl.BuildProgram(
		d.program,
		1,
		&d.Id,
		cl.Str(fmt.Sprintf("-I %s\x00", filepath.Dir(absProgramPath))),
		nil,
		nil,
	)
	if errCode != cl.SUCCESS {
		var dataLen uint64
		data := make([]byte, 120000)

		cl.GetProgramBuildInfo(d.program, d.Id, cl.PROGRAM_BUILD_LOG, uint64(len(data)), unsafe.Pointer(&data[0]), &dataLen)
		defer d.Close()
		return fmt.Errorf("opencl device (%s): could not build kernel (error: %s; code %d):\n%s", d.Name, ErrorName(errCode), errCode, string(data[0:dataLen-1]))
	}

	return nil
}

// Shut down the device.
func (d *Device) Close() {
	if d.program != nil {
		cl.ReleaseProgram(d.program)
		d.program = nil
	}

	if d.cmdQueue != nil {
		cl.ReleaseCommandQueue(d.cmdQueue)
		d.cmdQueue = nil
	}

	if d.ctx != nil {
		cl.ReleaseContext(d.ctx)
		d.ctx = nil
	}
}

// Load kernel by name.
func (d *Device) Kernel(name string) (*Kernel, error) {
	var errCode cl.ErrorCode
	kernelHandle := cl.CreateKernel(
		d.program,
		cl.Str(name+"\x00"),
		(*int32)(&errCode),
	)

	if errCode != cl.SUCCESS {
		return nil, fmt.Errorf("opencl device (%s): could not load kernel %s (error: %s; code %d)", d.Name, name, ErrorName(errCode), errCode)
	}

	return &Kernel{
		device:       d,
		kernelHandle: kernelHandle,
		name:         name,
	}, nil
}

// Create an empty buffer.
func (d *Device) Buffer(name string) *Buffer {
	return &Buffer{
		device: d,
		name:   name,
	}
}

// Detect device speed.
func (d *Device) detectSpeed() error {
	// Calculate theoretical device speed as: compute units * 2ops/cycle * clock speed
	errCode := cl.GetDeviceInfo(d.Id, cl.DEVICE_MAX_COMPUTE_UNITS, 4, unsafe.Pointer(&d.compUnits), nil)
	if errCode != cl.SUCCESS {
		return fmt.Errorf("opencl device (%s): could not query MAX_COMPUTE_UNITS (error: %s; code %d)", d.Name, ErrorName(errCode), errCode)
	}
	errCode = cl.GetDeviceInfo(d.Id, cl.DEVICE_MAX_CLOCK_FREQUENCY, 4, unsafe.Pointer(&d.clockSpeed), nil)
	if errCode != cl.SUCCESS {
		return fmt.Errorf("opencl device (%s): could not query MAX_CLOCK_FREQUENCY (error: %s; code %d)", d.Name, ErrorName(errCode), errCode)
	}
	d.Speed = d.compUnits * d.clockSpeed / 1000

	return nil
}

// Return a textual description of an opencl error code.
func ErrorName(errCode cl.ErrorCode) string {
	switch errCode {
	case 0:
		return "SUCCESS"
	case -1:
		return "DEVICE_NOT_FOUND"
	case -2:
		return "DEVICE_NOT_AVAILABLE"
	case -3:
		return "COMPILER_NOT_AVAILABLE"
	case -4:
		return "MEM_OBJECT_ALLOCATION_FAILURE"
	case -5:
		return "OUT_OF_RESOURCES"
	case -6:
		return "OUT_OF_HOST_MEMORY"
	case -7:
		return "PROFILING_INFO_NOT_AVAILABLE"
	case -8:
		return "MEM_COPY_OVERLAP"
	case -9:
		return "IMAGE_FORMAT_MISMATCH"
	case -10:
		return "IMAGE_FORMAT_NOT_SUPPORTED"
	case -11:
		return "BUILD_PROGRAM_FAILURE"
	case -12:
		return "MAP_FAILURE"
	case -30:
		return "INVALID_VALUE"
	case -31:
		return "INVALID_DEVICE_TYPE"
	case -32:
		return "INVALID_PLATFORM"
	case -33:
		return "INVALID_DEVICE"
	case -34:
		return "INVALID_CONTEXT"
	case -35:
		return "INVALID_QUEUE_PROPERTIES"
	case -36:
		return "INVALID_COMMAND_QUEUE"
	case -37:
		return "INVALID_HOST_PTR"
	case -38:
		return "INVALID_MEM_OBJECT"
	case -39:
		return "INVALID_IMAGE_FORMAT_DESCRIPTOR"
	case -40:
		return "INVALID_IMAGE_SIZE"
	case -41:
		return "INVALID_SAMPLER"
	case -42:
		return "INVALID_BINARY"
	case -43:
		return "INVALID_BUILD_OPTIONS"
	case -44:
		return "INVALID_PROGRAM"
	case -45:
		return "INVALID_PROGRAM_EXECUTABLE"
	case -46:
		return "INVALID_KERNEL_NAME"
	case -47:
		return "INVALID_KERNEL_DEFINITION"
	case -48:
		return "INVALID_KERNEL"
	case -49:
		return "INVALID_ARG_INDEX"
	case -50:
		return "INVALID_ARG_VALUE"
	case -51:
		return "INVALID_ARG_SIZE"
	case -52:
		return "INVALID_KERNEL_ARGS"
	case -53:
		return "INVALID_WORK_DIMENSION"
	case -54:
		return "INVALID_WORK_GROUP_SIZE"
	case -55:
		return "INVALID_WORK_ITEM_SIZE"
	case -56:
		return "INVALID_GLOBAL_OFFSET"
	case -57:
		return "INVALID_EVENT_WAIT_LIST"
	case -58:
		return "INVALID_EVENT"
	case -59:
		return "INVALID_OPERATION"
	case -60:
		return "INVALID_GL_OBJECT"
	case -61:
		return "INVALID_BUFFER_SIZE"
	case -62:
		return "INVALID_MIP_LEVEL"
	case -63:
		return "INVALID_GLOBAL_WORK_SIZE"
	default:
		return fmt.Sprintf("unknown error code %d", errCode)
	}
}
