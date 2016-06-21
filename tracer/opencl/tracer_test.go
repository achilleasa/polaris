package opencl

import (
	"testing"

	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
)

func TestTracerBlockWorker(t *testing.T) {
	tr := createTestTracer(t)
	defer tr.Close()
}

func createTestTracer(t *testing.T) *clTracer {
	devList, err := device.SelectDevices(device.CpuDevice, "CPU")
	if err != nil {
		t.Fatal(err)
	}

	if len(devList) == 0 {
		t.Fatal("could not detect CPU opencl device")
	}

	tr, err := newTracer("test", devList[0])
	if err != nil {
		t.Fatal(err)
	}

	err = tr.Init()
	if err != nil {
		t.Fatal(err)
	}

	return tr.(*clTracer)
}
