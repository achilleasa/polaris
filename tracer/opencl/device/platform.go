package device

import (
	"bytes"
	"fmt"
	"strings"
	"unsafe"

	"github.com/achilleasa/gopencl/v1.2/cl"
)

const (
	platformBufferSize = 100
	deviceBufferSize   = 100
	dataBufferSize     = 1024
)

// Information about a system's opencl platform and supported devices.
type PlatformInfo struct {
	Profile    string
	Version    string
	Name       string
	Vendor     string
	Extensions string
	Devices    []*Device
}

func (pl PlatformInfo) String() string {
	var buf bytes.Buffer

	buf.WriteString(
		fmt.Sprintf(
			"Version:    %s\nName:       %s\nVendor:     %s\nExtensions: %s\nDevices:\n",
			pl.Version,
			pl.Name,
			pl.Vendor,
			pl.Extensions,
		),
	)

	for dIdx, d := range pl.Devices {
		buf.WriteString(fmt.Sprintf("  Device %02d:\n", dIdx))
		buf.WriteString(indentRegex.ReplaceAllString(d.String(), "    "))
		buf.WriteString("\n\n")
	}

	return buf.String()
}

// Get information about supported opencl platforms and devices.
func GetPlatformInfo() ([]PlatformInfo, error) {

	pids := make([]cl.PlatformID, platformBufferSize)
	data := make([]byte, dataBufferSize)
	dataLen := uint64(0)

	devices := make([]cl.DeviceId, deviceBufferSize)
	deviceCount := uint32(0)

	pidCount := uint32(0)
	cl.GetPlatformIDs(uint32(len(pids)), &pids[0], &pidCount)

	infoList := make([]PlatformInfo, int(pidCount))
	for pIdx := 0; pIdx < int(pidCount); pIdx++ {
		infoList[pIdx].Devices = make([]*Device, 0)

		dataLen = 0
		cl.GetPlatformInfo(pids[pIdx], cl.PLATFORM_PROFILE, dataBufferSize, unsafe.Pointer(&data[0]), &dataLen)
		infoList[pIdx].Profile = string(data[0 : dataLen-1])

		cl.GetPlatformInfo(pids[pIdx], cl.PLATFORM_VERSION, dataBufferSize, unsafe.Pointer(&data[0]), &dataLen)
		infoList[pIdx].Version = string(data[0 : dataLen-1])

		cl.GetPlatformInfo(pids[pIdx], cl.PLATFORM_NAME, dataBufferSize, unsafe.Pointer(&data[0]), &dataLen)
		infoList[pIdx].Name = string(data[0 : dataLen-1])

		cl.GetPlatformInfo(pids[pIdx], cl.PLATFORM_VENDOR, dataBufferSize, unsafe.Pointer(&data[0]), &dataLen)
		infoList[pIdx].Vendor = string(data[0 : dataLen-1])

		cl.GetPlatformInfo(pids[pIdx], cl.PLATFORM_EXTENSIONS, dataBufferSize, unsafe.Pointer(&data[0]), &dataLen)
		infoList[pIdx].Extensions = string(data[0 : dataLen-1])

		// Enumerate CPU devices
		deviceCount = 0
		cl.GetDeviceIDs(pids[pIdx], cl.DEVICE_TYPE_CPU, uint32(deviceBufferSize), &devices[0], &deviceCount)
		for dIdx := 0; dIdx < int(deviceCount); dIdx++ {
			cl.GetDeviceInfo(devices[dIdx], cl.DEVICE_NAME, dataBufferSize, unsafe.Pointer(&data[0]), &dataLen)
			infoList[pIdx].Devices = append(
				infoList[pIdx].Devices,
				&Device{
					Name: string(data[0 : dataLen-1]),
					Id:   devices[dIdx],
					Type: CpuDevice,
				},
			)
		}

		// Enumerate GPU devices
		deviceCount = 0
		cl.GetDeviceIDs(pids[pIdx], cl.DEVICE_TYPE_GPU, uint32(deviceBufferSize), &devices[0], &deviceCount)
		for dIdx := 0; dIdx < int(deviceCount); dIdx++ {
			cl.GetDeviceInfo(devices[dIdx], cl.DEVICE_NAME, dataBufferSize, unsafe.Pointer(&data[0]), &dataLen)
			infoList[pIdx].Devices = append(
				infoList[pIdx].Devices,
				&Device{
					Name: string(data[0 : dataLen-1]),
					Id:   devices[dIdx],
					Type: GpuDevice,
				},
			)
		}

		// Enumerate speed for all platform devices
		for _, dev := range infoList[pIdx].Devices {
			err := dev.detectSpeed()
			if err != nil {
				return nil, err
			}
		}
	}

	return infoList, nil
}

// Scan all available opencl platforms and select devices that match the given query.
func SelectDevices(typeMask DeviceType, matchName string) ([]*Device, error) {
	platforms, err := GetPlatformInfo()
	if err != nil {
		return nil, err
	}
	list := make([]*Device, 0)
	for _, p := range platforms {
		for _, d := range p.Devices {
			// Match type
			if d.Type&typeMask != d.Type {
				continue
			}

			// Match name
			if matchName != "" && !strings.Contains(d.Name, matchName) {
				continue
			}

			list = append(list, d)
		}
	}
	return list, nil
}
