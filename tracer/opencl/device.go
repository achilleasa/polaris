package opencl

import (
	"fmt"
	"regexp"

	"github.com/hydroflame/gopencl/v1.2/cl"
)

type DeviceType uint8

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

// Information about an opencl-supported device.
type Device struct {
	Name string
	Id   cl.DeviceId
	Type DeviceType
}

func (d Device) String() string {
	return fmt.Sprintf("Name: %s\nType: %s", d.Name, d.Type.String())
}

// A list of devices.
type DeviceList []Device
