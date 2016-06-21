package integrator

import (
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/achilleasa/go-pathtrace/log"
	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/scene/reader"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/go-pathtrace/types"
)

func BenchmarkCpuGeneratePrimaryRays128(b *testing.B) {
	benchmarkGeneratePrimaryRays("CPU", 128, b)
}

func BenchmarkCpuGeneratePrimaryRays256(b *testing.B) {
	benchmarkGeneratePrimaryRays("CPU", 256, b)
}

func BenchmarkCpuGeneratePrimaryRays512(b *testing.B) {
	benchmarkGeneratePrimaryRays("CPU", 512, b)
}

func BenchmarkCpuGeneratePrimaryRays1024(b *testing.B) {
	benchmarkGeneratePrimaryRays("CPU", 1024, b)
}

func BenchmarkIrisGpuGeneratePrimaryRays128(b *testing.B) {
	benchmarkGeneratePrimaryRays("Iris", 128, b)
}

func BenchmarkIrisGpuGeneratePrimaryRays256(b *testing.B) {
	benchmarkGeneratePrimaryRays("Iris", 256, b)
}

func BenchmarkIrisGpuGeneratePrimaryRays512(b *testing.B) {
	benchmarkGeneratePrimaryRays("Iris", 512, b)
}

func BenchmarkIrisGpuGeneratePrimaryRays1024(b *testing.B) {
	benchmarkGeneratePrimaryRays("Iris", 1024, b)
}

func BenchmarkAMDGpuGeneratePrimaryRays128(b *testing.B) {
	benchmarkGeneratePrimaryRays("AMD", 128, b)
}

func BenchmarkAMDGpuGeneratePrimaryRays256(b *testing.B) {
	benchmarkGeneratePrimaryRays("AMD", 256, b)
}

func BenchmarkAMDGpuGeneratePrimaryRays512(b *testing.B) {
	benchmarkGeneratePrimaryRays("AMD", 512, b)
}

func BenchmarkAMDGpuGeneratePrimaryRays1024(b *testing.B) {
	benchmarkGeneratePrimaryRays("AMD", 1024, b)
}

func benchmarkGeneratePrimaryRays(devName string, frameSize uint32, b *testing.B) {
	log.DefaultSink = ioutil.Discard
	defer func() {
		log.DefaultSink = os.Stdout
	}()

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

func BenchmarkCpuRayIntersectionTest128(b *testing.B) {
	benchmarkRayIntersectionTest("CPU", 128, b)
}

func BenchmarkCpuRayIntersectionTest256(b *testing.B) {
	benchmarkRayIntersectionTest("CPU", 256, b)
}

func BenchmarkCpuRayIntersectionTest512(b *testing.B) {
	benchmarkRayIntersectionTest("CPU", 512, b)
}

func BenchmarkCpuRayIntersectionTest1024(b *testing.B) {
	benchmarkRayIntersectionTest("CPU", 1024, b)
}

func BenchmarkIrisGpuRayIntersectionTest128(b *testing.B) {
	benchmarkRayIntersectionTest("Iris", 128, b)
}

func BenchmarkIrisGpuRayIntersectionTest256(b *testing.B) {
	benchmarkRayIntersectionTest("Iris", 256, b)
}

func BenchmarkIrisGpuRayIntersectionTest512(b *testing.B) {
	benchmarkRayIntersectionTest("Iris", 512, b)
}

func BenchmarkIrisGpuRayIntersectionTest1024(b *testing.B) {
	benchmarkRayIntersectionTest("Iris", 1024, b)
}

func BenchmarkAMDGpuRayIntersectionTest128(b *testing.B) {
	benchmarkRayIntersectionTest("AMD", 128, b)
}

func BenchmarkAMDGpuRayIntersectionTest256(b *testing.B) {
	benchmarkRayIntersectionTest("AMD", 256, b)
}

func BenchmarkAMDGpuRayIntersectionTest512(b *testing.B) {
	benchmarkRayIntersectionTest("AMD", 512, b)
}

func BenchmarkAMDGpuRayIntersectionTest1024(b *testing.B) {
	benchmarkRayIntersectionTest("AMD", 1024, b)
}

// Benchmark intersection test for a blockDim * blockDim rays.
func benchmarkRayIntersectionTest(devName string, blockDim uint32, b *testing.B) {
	log.DefaultSink = ioutil.Discard
	defer func() {
		log.DefaultSink = os.Stdout
	}()

	in := createBenchIntegrator(b, devName)
	defer in.Close()

	sc := createBenchScene(b)

	err := in.UploadSceneData(sc)
	if err != nil {
		b.Fatal(err)
	}
	blockReq := &tracer.BlockRequest{
		FrameW: blockDim,
		FrameH: blockDim,
		BlockY: 0,
		BlockH: blockDim,
	}

	camera := setupCamera(types.Vec3{0, 0, 2}, types.Vec3{0, 0, 0}, blockReq)
	err = in.UploadCameraData(camera)
	if err != nil {
		b.Fatal(err)
	}

	err = in.ResizeOutputFrame(blockReq.FrameW, blockReq.FrameH)
	if err != nil {
		b.Fatal(err)
	}

	// Generate rays
	type ray struct {
		origin types.Vec4
		dir    types.Vec4
	}
	rays := make([]ray, blockDim*blockDim)
	var rayIndex uint32
	for rayIndex = 0; rayIndex < blockDim*blockDim; rayIndex++ {
		rays[rayIndex] = ray{
			origin: types.Vec4{
				-1.5 + rand.Float32(),
				-1.5 + rand.Float32(),
				2,
				math.MaxFloat32,
			},
			dir: types.Vec4{0, 0, -1, 0},
		}
	}

	err = in.buffers.Rays.WriteData(rays, 0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err = in.RayIntersectionTest(uint32(len(rays)))
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

func createBenchScene(b *testing.B) *scene.Scene {
	_, thisFile, _, _ := runtime.Caller(0)
	thisDir := filepath.Dir(thisFile)

	s, err := reader.ReadScene(filepath.Join(thisDir, "fixtures/cube.obj"))
	if err != nil {
		b.Fatal(err)
	}
	return s
}
