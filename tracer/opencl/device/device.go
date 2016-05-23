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
	var errPtr *int32

	// Already initialized
	if d.ctx != nil {
		return nil
	}

	// Create context
	d.ctx = cl.CreateContext(nil, 1, &d.Id, nil, nil, errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		defer d.Close()
		return fmt.Errorf("opencl device (%s): could not create opencl context (errCode %d)", d.Name, cl.ErrorCode(*errPtr))
	}

	// Create command queue
	d.cmdQueue = cl.CreateCommandQueue(*d.ctx, d.Id, 0, errPtr)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		defer d.Close()
		return fmt.Errorf("opencl device (%s): could not create opencl context (errCode %d)", d.Name, cl.ErrorCode(*errPtr))
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
		errPtr,
	)
	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		defer d.Close()
		return fmt.Errorf("opencl device (%s): could not create program (errCode %d)", d.Name, cl.ErrorCode(*errPtr))
	}

	errCode := cl.BuildProgram(
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
		return fmt.Errorf("opencl device (%s): could not build kernel (errCode %d):\n%s", d.Name, errCode, string(data[0:dataLen-1]))
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
	var errPtr *int32
	kernelHandle := cl.CreateKernel(
		d.program,
		cl.Str(name+"\x00"),
		errPtr,
	)

	if errPtr != nil && cl.ErrorCode(*errPtr) != cl.SUCCESS {
		return nil, fmt.Errorf("opencl device (%s): could not load kernelHandle %s (errCode %d)", d.Name, name, cl.ErrorCode(*errPtr))
	}

	return &Kernel{
		device:       d,
		kernelHandle: kernelHandle,
		name:         name,
	}, nil
}

// Detect device speed.
func (d *Device) detectSpeed() error {
	// Calculate theoretical device speed as: compute units * 2ops/cycle * clock speed
	errCode := cl.GetDeviceInfo(d.Id, cl.DEVICE_MAX_COMPUTE_UNITS, 4, unsafe.Pointer(&d.compUnits), nil)
	if errCode != cl.SUCCESS {
		return fmt.Errorf("opencl device (%s): could not query MAX_COMPUTE_UNITS (errCode %d)", d.Name, errCode)
	}
	errCode = cl.GetDeviceInfo(d.Id, cl.DEVICE_MAX_CLOCK_FREQUENCY, 4, unsafe.Pointer(&d.clockSpeed), nil)
	if errCode != cl.SUCCESS {
		return fmt.Errorf("opencl device (%s): could not query MAX_CLOCK_FREQUENCY (errCode %d)", d.Name, errCode)
	}
	var opsPerCycle uint32 = 2
	if d.Type == CpuDevice {
		opsPerCycle = 4
	}
	d.Speed = d.compUnits * opsPerCycle * d.clockSpeed / 1000

	return nil
}
