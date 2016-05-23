package device

import (
	"strings"
	"testing"
)

func TestSelectDevices(t *testing.T) {
	devList, err := SelectDevices(CpuDevice, "CPU")
	if err != nil {
		t.Fatal(err)
	}
	if len(devList) != 1 {
		t.Fatalf("expected to get 1 CPU opencl device; got %d; check that openCL drivers are installed", len(devList))
	}
}

func TestDeviceInit(t *testing.T) {
	dev, err := createCpuTestDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	if !strings.Contains(dev.Name, "CPU") {
		t.Fatalf("expected CPU device name '%s' to contain 'CPU'", dev.Name)
	}

	if dev.Type.String() != "CPU" {
		t.Fatalf("expected device type to be CpuDevice; got %s", dev.Type.String())
	}
}

func createCpuTestDevice() (*Device, error) {
	devList, err := SelectDevices(CpuDevice, "CPU")
	if err != nil {
		return nil, err
	}

	return devList[0], devList[0].Init("test.cl")
}
