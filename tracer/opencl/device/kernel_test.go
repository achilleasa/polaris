package device

import (
	"testing"

	"github.com/achilleasa/gopencl/v1.2/cl"
)

func TestKernelExec1DWithAutoLocalWorkSize(t *testing.T) {
	dev, err := createCpuTestDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	kernel, err := dev.Kernel("square")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Release()

	dataSize := 32
	dataIn := make([]int32, dataSize)
	dataOut := make([]int32, dataSize)
	for i := 0; i < dataSize; i++ {
		dataIn[i] = int32(i)
	}

	bufIn := dev.Buffer("in")
	defer bufIn.Release()
	err = bufIn.AllocateAndWriteData(dataIn, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	bufOut := dev.Buffer("out")
	defer bufOut.Release()
	err = bufOut.AllocateToFitData(dataOut, cl.MEM_READ_WRITE)

	if err != nil {
		t.Fatal(err)
	}

	var size uint32 = uint32(dataSize)
	err = kernel.SetArgs(
		bufIn,
		bufOut,
		size,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = kernel.Exec1D(0, dataSize, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch and validate output
	bufOut.ReadData(0, 0, 0, dataOut)
	for i := 0; i < dataSize; i++ {
		expValue := dataIn[i] * dataIn[i]
		if dataOut[i] != expValue {
			t.Fatalf("[item %d] expected squared value of %d to be %d; got %d", i, dataIn[i], expValue, dataOut[i])
		}
	}
}

func TestKernelExec1D(t *testing.T) {
	dev, err := createCpuTestDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	kernel, err := dev.Kernel("square")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Release()

	dataSize := 32
	dataIn := make([]int32, dataSize)
	dataOut := make([]int32, dataSize)
	for i := 0; i < dataSize; i++ {
		dataIn[i] = int32(i)
	}

	bufIn := dev.Buffer("in")
	defer bufIn.Release()
	err = bufIn.AllocateAndWriteData(dataIn, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	bufOut := dev.Buffer("out")
	defer bufOut.Release()
	err = bufOut.AllocateToFitData(dataOut, cl.MEM_READ_WRITE)

	if err != nil {
		t.Fatal(err)
	}

	var size uint32 = uint32(dataSize)
	err = kernel.SetArgs(
		bufIn,
		bufOut,
		size,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = kernel.Exec1D(0, dataSize, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch and validate output
	bufOut.ReadData(0, 0, 0, dataOut)
	for i := 0; i < dataSize; i++ {
		expValue := dataIn[i] * dataIn[i]
		if dataOut[i] != expValue {
			t.Fatalf("[item %d] expected squared value of %d to be %d; got %d", i, dataIn[i], expValue, dataOut[i])
		}
	}
}
func TestKernelExec2DWithAutoLocalWorkSize(t *testing.T) {
	dev, err := createCpuTestDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	kernel, err := dev.Kernel("mapBlock")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Release()

	dataWidth := 8
	dataHeight := 8

	dataIn := make([]int32, dataWidth*dataHeight)
	dataOut := make([]int32, dataWidth*dataHeight)
	for i := 0; i < dataWidth*dataHeight; i++ {
		dataIn[i] = int32(i)
	}

	bufIn := dev.Buffer("in")
	defer bufIn.Release()
	err = bufIn.AllocateAndWriteData(dataIn, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	bufOut := dev.Buffer("out")
	defer bufOut.Release()
	err = bufOut.AllocateToFitData(dataOut, cl.MEM_READ_WRITE)

	if err != nil {
		t.Fatal(err)
	}

	var size uint32 = uint32(dataWidth * dataHeight)
	err = kernel.SetArgs(
		bufIn,
		bufOut,
		size,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = kernel.Exec2D(0, 0, dataWidth, dataHeight, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch and validate output
	bufOut.ReadData(0, 0, 0, dataOut)
	for i := 0; i < dataWidth*dataHeight; i++ {
		expValue := dataIn[i]
		if dataOut[i] != expValue {
			t.Fatalf("[item %d] expected squared value of %d to be %d; got %d", i, dataIn[i], expValue, dataOut[i])
		}
	}
}

func TestKernelExec2D(t *testing.T) {
	dev, err := createCpuTestDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	kernel, err := dev.Kernel("mapBlock")
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Release()

	dataWidth := 8
	dataHeight := 8

	dataIn := make([]int32, dataWidth*dataHeight)
	dataOut := make([]int32, dataWidth*dataHeight)
	for i := 0; i < dataWidth*dataHeight; i++ {
		dataIn[i] = int32(i)
	}

	bufIn := dev.Buffer("in")
	defer bufIn.Release()
	err = bufIn.AllocateAndWriteData(dataIn, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	bufOut := dev.Buffer("out")
	defer bufOut.Release()
	err = bufOut.AllocateToFitData(dataOut, cl.MEM_READ_WRITE)

	if err != nil {
		t.Fatal(err)
	}

	var size uint32 = uint32(dataWidth * dataHeight)
	err = kernel.SetArgs(
		bufIn,
		bufOut,
		size,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = kernel.Exec2D(0, 0, dataWidth, dataHeight, 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch and validate output
	bufOut.ReadData(0, 0, 0, dataOut)
	for i := 0; i < dataWidth*dataHeight; i++ {
		expValue := dataIn[i]
		if dataOut[i] != expValue {
			t.Fatalf("[item %d] expected squared value of %d to be %d; got %d", i, dataIn[i], expValue, dataOut[i])
		}
	}
}
