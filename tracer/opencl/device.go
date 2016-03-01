package opencl

import (
	"fmt"
	"regexp"

	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/gopencl/v1.2/cl"
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

// Get speed estimate for device compared to a baseline (cpu) implementation
func (d Device) SpeedEstimate() float32 {
	if d.Type == GpuDevice {
		// Assume that GPU has 10x power of a CPU device
		return 10.0
	}
	return 1.0
}

// Create a opencl tracer for a specific device.
func (d Device) GetTracer() (tracer.Tracer, error) {
	return newTracer(d.Name, d)
}

func (d Device) String() string {
	return fmt.Sprintf("Name: %s\nType: %s", d.Name, d.Type.String())
}

// A list of devices.
type DeviceList []Device

// Create opencl tracers for devices matching type flags
func (dl DeviceList) GetTracers(typeMask DeviceType) ([]tracer.Tracer, []error) {
	list := make([]tracer.Tracer, 0)
	errList := make([]error, 0)

	for _, d := range dl {
		if d.Type&typeMask == d.Type {
			tr, err := d.GetTracer()
			if err != nil {
				errList = append(errList, fmt.Errorf("%s: %s", d.Name, err.Error()))
			} else {
				list = append(list, tr)
			}
		}
	}
	return list, errList
}
