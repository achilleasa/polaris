package integrator

import (
	"testing"

	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/go-pathtrace/types"
)

func BenchmarkCpuGeneratePrimaryRays128(b *testing.B) {
	benchmarkGeneratePrimaryRays("CPU", 1024, b)
}

func BenchmarkCpuGeneratePrimaryRays256(b *testing.B) {
	benchmarkGeneratePrimaryRays("CPU", 1024, b)
}

func BenchmarkCpuGeneratePrimaryRays512(b *testing.B) {
	benchmarkGeneratePrimaryRays("CPU", 1024, b)
}

func BenchmarkCpuGeneratePrimaryRays1024(b *testing.B) {
	benchmarkGeneratePrimaryRays("CPU", 1024, b)
}

func BenchmarkIrisGpuGeneratePrimaryRays128(b *testing.B) {
	benchmarkGeneratePrimaryRays("Iris", 1024, b)
}

func BenchmarkIrisGpuGeneratePrimaryRays256(b *testing.B) {
	benchmarkGeneratePrimaryRays("Iris", 1024, b)
}

func BenchmarkIrisGpuGeneratePrimaryRays512(b *testing.B) {
	benchmarkGeneratePrimaryRays("Iris", 1024, b)
}

func BenchmarkIrisGpuGeneratePrimaryRays1024(b *testing.B) {
	benchmarkGeneratePrimaryRays("Iris", 1024, b)
}

func BenchmarkAMDGpuGeneratePrimaryRays128(b *testing.B) {
	benchmarkGeneratePrimaryRays("AMD", 1024, b)
}

func BenchmarkAMDGpuGeneratePrimaryRays256(b *testing.B) {
	benchmarkGeneratePrimaryRays("AMD", 1024, b)
}

func BenchmarkAMDGpuGeneratePrimaryRays512(b *testing.B) {
	benchmarkGeneratePrimaryRays("AMD", 1024, b)
}

func BenchmarkAMDGpuGeneratePrimaryRays1024(b *testing.B) {
	benchmarkGeneratePrimaryRays("AMD", 1024, b)
}

func benchmarkGeneratePrimaryRays(devName string, frameSize uint32, b *testing.B) {
	in := createBenchIntegrator(b, devName)
	defer in.Close()

	blockReq := &tracer.BlockRequest{
		FrameW: frameSize,
		FrameH: frameSize,
		BlockY: 0,
		BlockH: frameSize,
	}

	camera := setupCamera(types.Vec3{0, 0, 1}, types.Vec3{0, 0, 0}, blockReq)
	err := in.UploadCameraData(camera)
	if err != nil {
		b.Fatal(err)
	}

	err = in.ResizeOutputFrame(blockReq.FrameW, blockReq.FrameH)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err = in.GeneratePrimaryRays(blockReq)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createBenchIntegrator(b *testing.B, devName string) *MonteCarlo {
	devList, err := device.SelectDevices(device.AllDevices, devName)
	if err != nil {
		b.Fatal(err)
	}

	if len(devList) == 0 {
		b.Skipf("could not detect any opencl device matching '%s'", devName)
	}

	in := NewMonteCarloIntegrator(devList[0])
	err = in.Init()
	if err != nil {
		b.Fatal(err)
	}

	return in
}
