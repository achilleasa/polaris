package device

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/achilleasa/gopencl/v1.2/cl"
)

func TestBufferAllocate(t *testing.T) {
	dev, err := createCpuDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	buf := dev.Buffer("test")
	defer buf.Release()
	err = buf.Allocate(128, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	expSize := 128
	if buf.Size() != expSize {
		t.Fatalf("expected buffer size to be %d; got %d", expSize, buf.Size())
	}
}

func TestBufferAllocateToFitData(t *testing.T) {
	dev, err := createCpuDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	data := make([]float64, 128)

	buf := dev.Buffer("test")
	defer buf.Release()
	err = buf.AllocateToFitData(data, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	expSize := len(data) * int(unsafe.Sizeof(data[0]))
	if buf.Size() != expSize {
		t.Fatalf("expected buffer size to be %d; got %d", expSize, buf.Size())
	}
}

func TestBufferAllocateAndWriteData(t *testing.T) {
	dev, err := createCpuDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	data := make([]byte, 128)
	for i := 0; i < 128; i++ {
		data[i] = byte(i)
	}

	buf := dev.Buffer("test")
	defer buf.Release()
	err = buf.AllocateAndWriteData(data, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	expSize := len(data)
	if buf.Size() != expSize {
		t.Fatalf("expected buffer size to be %d; got %d", expSize, buf.Size())
	}
}

func TestDataReadWriteWithArrayTargets(t *testing.T) {
	dev, err := createCpuDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	var data [128]byte
	for i := 0; i < 128; i++ {
		data[i] = byte(i)
	}

	buf := dev.Buffer("test")
	defer buf.Release()
	err = buf.Allocate(128, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	// We need to convert the array into a slice
	err = buf.WriteData(data[:], 0)
	if err != nil {
		t.Fatal(err)
	}

	dataOut := make([]byte, 128)
	err = buf.ReadData(0, 0, 0, dataOut)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(data[:], dataOut) {
		t.Fatal("read data does not match written data")
	}
}

func TestDataReadWrite(t *testing.T) {
	dev, err := createCpuDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	data := make([]byte, 128)
	for i := 0; i < 128; i++ {
		data[i] = byte(i)
	}

	buf := dev.Buffer("test")
	defer buf.Release()
	err = buf.Allocate(128, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	err = buf.WriteData(data, 0)
	if err != nil {
		t.Fatal(err)
	}

	dataOut := make([]byte, 128)
	err = buf.ReadData(0, 0, 0, dataOut)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(data, dataOut) {
		t.Fatal("read data does not match written data")
	}
}

func TestDataReadWriteWithStructSlices(t *testing.T) {
	dev, err := createCpuDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	type foo struct {
		x    float32
		name string
	}

	numFoos := 10
	data := make([]foo, numFoos)
	for i := 0; i < numFoos; i++ {
		data[i].x = float32(i)
		data[i].name = fmt.Sprintf("%d", i)
	}

	buf := dev.Buffer("test")
	defer buf.Release()
	err = buf.Allocate(len(data)*int(unsafe.Sizeof(data[0])), cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	err = buf.WriteData(data, 0)
	if err != nil {
		t.Fatal(err)
	}

	dataOut := make([]foo, numFoos)
	err = buf.ReadData(0, 0, 0, dataOut)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(data, dataOut) {
		t.Fatal("read data does not match written data")
	}
}

func TestDataReadWriteOffsets(t *testing.T) {
	dev, err := createCpuDevice()
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	data := make([]byte, 128)
	for i := 0; i < 128; i++ {
		data[i] = byte(i)
	}

	buf := dev.Buffer("test")
	defer buf.Release()
	err = buf.Allocate(128, cl.MEM_READ_WRITE)
	if err != nil {
		t.Fatal(err)
	}

	err = buf.WriteData(data, 64)
	if err != nil {
		t.Fatal(err)
	}

	dataOut := make([]byte, 128)
	err = buf.ReadData(64, 0, 64, dataOut)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(data[:64], dataOut[:64]) {
		t.Fatal("read data does not match written data")
	}
}

func TestGetSliceData(t *testing.T) {
	data := make([]int32, 32)
	_, dataLen := getSliceData(data)

	expSize := 4 * 32
	if dataLen != expSize {
		t.Fatalf("expected datalen to be %d; got %d", expSize, dataLen)
	}
}

func createCpuDevice() (*Device, error) {
	devList, err := SelectDevices(CpuDevice, "CPU")
	if err != nil {
		return nil, err
	}

	dev := devList[0]
	err = dev.Init("test.cl")
	if err != nil {
		return nil, err
	}

	return dev, nil
}
