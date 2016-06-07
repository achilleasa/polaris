package integrator

import (
	"testing"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/go-pathtrace/types"
)

func TestGeneratePrimaryRays(t *testing.T) {
	in := createTestIntegrator(t, "CPU")
	defer in.Close()

	blockReq := &tracer.BlockRequest{
		FrameW: 2,
		FrameH: 2,
		BlockY: 0,
		BlockH: 2,
	}

	camera := setupCamera(types.Vec3{0, 0, 1}, types.Vec3{0, 0, 0}, blockReq)
	err := in.UploadCameraData(camera)
	if err != nil {
		t.Fatal(err)
	}

	err = in.ResizeOutputFrame(blockReq.FrameW, blockReq.FrameH)
	if err != nil {
		t.Fatal(err)
	}

	_, err = in.GeneratePrimaryRays(blockReq)
	if err != nil {
		t.Fatal(err)
	}

	// read back and verify ray and path data
	type ray struct {
		origin types.Vec4
		dir    types.Vec4
	}
	type path struct {
		throughput types.Vec4
		status     uint32
		pixelIndex uint32
		reserved   [2]uint32
	}
	rayData := readIntoSlice(t, in.buffers.Rays, make([]ray, 0)).([]ray)
	pathData := readIntoSlice(t, in.buffers.Paths, make([]path, 0)).([]path)

	var x, y uint32
	for y = 0; y < blockReq.FrameH; y++ {
		for x = 0; x < blockReq.FrameW; x++ {
			index := (y * blockReq.FrameW) + x
			r := rayData[index]
			if !types.ApproxEqual(r.origin.Vec3(), in.cameraEyePos, 0.001) {
				t.Fatalf("ray(%d, %d) expected eye to be %v; got %v", x, y, in.cameraEyePos, r.origin.Vec3())
			}

			// origin.w should be -1 (no dist limit)
			if r.origin[3] != -1.0 {
				t.Fatalf("ray(%d, %d) expected origin.w to be -1.0; got %f", x, y, r.origin[3])
			}

			// dir should be a trilinear blend of the frustrum corners
			tx := float32(x) / float32(blockReq.FrameW)
			ty := float32(y) / float32(blockReq.FrameH)
			lVec := camera.Frustrum[0].Mul(1.0 - ty).Vec3().Add(camera.Frustrum[2].Mul(ty).Vec3())
			rVec := camera.Frustrum[1].Mul(1.0 - ty).Vec3().Add(camera.Frustrum[3].Mul(ty).Vec3())
			expDir := lVec.Mul(1.0 - tx).Add(rVec.Mul(tx)).Normalize()

			if !types.ApproxEqual(r.dir.Vec3(), expDir, 0.001) {
				t.Fatalf("ray(%d, %d) expected dir to be %v; got %v", x, y, expDir, r.dir.Vec3())
			}

			p := pathData[index]
			if p.status != 1 {
				t.Fatalf("path(%d, %d) exepected status to be active(1); got %d", x, y, p.status)
			}
			if p.pixelIndex != index {
				t.Fatalf("path(%d, %d) expected pixel index to be %d; got %d", x, y, index, p.pixelIndex)
			}
		}
	}
}

func setupCamera(eye, look types.Vec3, blockReq *tracer.BlockRequest) *scene.Camera {
	camera := scene.NewCamera(45)
	camera.Position = eye
	camera.Up = types.Vec3{0, 1, 0}
	camera.LookAt = look

	camera.SetupProjection(float32(blockReq.FrameW) / float32(blockReq.FrameH))
	camera.Update()

	return camera
}

func createTestIntegrator(t *testing.T, devName string) *MonteCarlo {
	devList, err := device.SelectDevices(device.AllDevices, devName)
	if err != nil {
		t.Fatal(err)
	}

	if len(devList) == 0 {
		t.Fatal("could not detect CPU opencl device")
	}

	in := NewMonteCarloIntegrator(devList[0])
	err = in.Init()
	if err != nil {
		t.Fatal(err)
	}

	return in
}

func readIntoSlice(t *testing.T, buf *device.Buffer, sliceType interface{}) interface{} {
	d, err := buf.ReadDataIntoSlice(sliceType)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
