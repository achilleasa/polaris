package integrator

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/scene/reader"
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
			index := morton2d(x, y)
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
			expPixelIndex := (y * blockReq.FrameW * 4) + x
			if p.pixelIndex != expPixelIndex {
				t.Fatalf("path(%d, %d) expected pixel index to be %d; got %d", x, y, expPixelIndex, p.pixelIndex)
			}
		}
	}
}

func TestRayIntersectionTest(t *testing.T) {
	in := createTestIntegrator(t, "CPU")
	defer in.Close()

	sc := createTestScene(t)

	err := in.UploadSceneData(sc)
	if err != nil {
		t.Fatal(err)
	}

	blockReq := &tracer.BlockRequest{
		FrameW: 10,
		FrameH: 1,
		BlockY: 0,
		BlockH: 1,
	}

	camera := setupCamera(types.Vec3{0, 0, 2}, types.Vec3{0, 0, 0}, blockReq)
	err = in.UploadCameraData(camera)
	if err != nil {
		t.Fatal(err)
	}

	err = in.ResizeOutputFrame(blockReq.FrameW, blockReq.FrameH)
	if err != nil {
		t.Fatal(err)
	}

	// Generate rays
	type ray struct {
		origin types.Vec4
		dir    types.Vec4
	}
	rays := []ray{
		{
			origin: types.Vec4{0.25, 0.25, 2, math.MaxFloat32},
			dir:    types.Vec4{0, 0, -1, 0},
		},
		// This ray should intersect the translated cube instance
		{
			origin: types.Vec4{-1.0, 0, 2, math.MaxFloat32},
			dir:    types.Vec4{0, 0, -1, 0},
		},
		// This ray should miss
		{
			origin: types.Vec4{-1.0, 1.0, 2, math.MaxFloat32},
			dir:    types.Vec4{0, 0, -1, 0},
		},
		// This ray should miss due to max ray dist (W origin coord)
		{
			origin: types.Vec4{0.25, 0.25, 2, 1.49},
			dir:    types.Vec4{0, 0, -1, 0},
		},
	}
	err = in.buffers.Rays.WriteData(rays, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = in.RayIntersectionTest(uint32(len(rays)))
	if err != nil {
		t.Fatal(err)
	}

	hitFlags := readIntoSlice(t, in.buffers.HitFlags, make([]uint32, 0)).([]uint32)
	expResults := []uint32{1, 1, 0, 0}
	for resIndex, expRes := range expResults {
		if hitFlags[resIndex] != expRes {
			t.Fatalf("[ray %d] expected hit flag to be %d; got %d", resIndex, expRes, hitFlags[resIndex])
		}
	}
}

func TestRayIntersectionQuery(t *testing.T) {
	in := createTestIntegrator(t, "CPU")
	defer in.Close()

	sc := createTestScene(t)

	err := in.UploadSceneData(sc)
	if err != nil {
		t.Fatal(err)
	}

	blockReq := &tracer.BlockRequest{
		FrameW: 10,
		FrameH: 1,
		BlockY: 0,
		BlockH: 1,
	}

	camera := setupCamera(types.Vec3{0, 0, 2}, types.Vec3{0, 0, 0}, blockReq)
	err = in.UploadCameraData(camera)
	if err != nil {
		t.Fatal(err)
	}

	err = in.ResizeOutputFrame(blockReq.FrameW, blockReq.FrameH)
	if err != nil {
		t.Fatal(err)
	}

	// Generate rays
	type ray struct {
		origin types.Vec4
		dir    types.Vec4
	}
	rays := []ray{
		{
			origin: types.Vec4{0.25, 0.25, 2, math.MaxFloat32},
			dir:    types.Vec4{0, 0, -1, 0},
		},
		// This ray should intersect the translated cube instance
		{
			origin: types.Vec4{-1.0, 0, 2, math.MaxFloat32},
			dir:    types.Vec4{0, 0, -1, 0},
		},
		// This ray should miss
		{
			origin: types.Vec4{-1.0, 1.0, 2, math.MaxFloat32},
			dir:    types.Vec4{0, 0, -1, 0},
		},
		// This ray should miss due to max ray dist (W origin coord)
		{
			origin: types.Vec4{0.25, 0.25, 2, 1.49},
			dir:    types.Vec4{0, 0, -1, 0},
		},
	}
	err = in.buffers.Rays.WriteData(rays, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = in.RayIntersectionQuery(uint32(len(rays)))
	if err != nil {
		t.Fatal(err)
	}

	hitFlags := readIntoSlice(t, in.buffers.HitFlags, make([]uint32, 0)).([]uint32)
	expResults := []uint32{1, 1, 0, 0}
	for resIndex, expRes := range expResults {
		if hitFlags[resIndex] != expRes {
			t.Fatalf("[ray %d] expected hit flag to be %d; got %d", resIndex, expRes, hitFlags[resIndex])
		}
	}

	type intersection struct {
		wuvt         types.Vec4
		meshInstance uint32
		triIndex     uint32
		_padding1    uint32
		_padding2    uint32
	}
	intersections := readIntoSlice(t, in.buffers.Intersections, make([]intersection, 0)).([]intersection)
	expIntersections := []intersection{
		{
			wuvt:         types.Vec4{0, 0, 0, 1.5},
			meshInstance: 0,
			triIndex:     3,
		},
		{
			wuvt:         types.Vec4{0, 0, 0, 1.5},
			meshInstance: 1,
			triIndex:     2,
		},
	}
	for expIndex, exp := range expIntersections {
		target := intersections[expIndex]

		// Test barycentric coordinate calculations
		// Test that w = 1.0 - (u+v)
		expW := float64(1.0 - (target.wuvt[1] + target.wuvt[2]))
		if math.Abs(expW-float64(target.wuvt[0])) > 0.001 {
			t.Fatalf("[intersection %d] expected w barycentric coord to be %f; got %f", expIndex, expW, target.wuvt[0])
		}

		// Test distance to hit
		if math.Abs(float64(exp.wuvt[3]-target.wuvt[3])) > 0.001 {
			t.Fatalf("[intersection %d] expected t to be %f; got %f", expIndex, exp.wuvt[3], target.wuvt[3])
		}

		if target.meshInstance != exp.meshInstance {
			t.Fatalf("[intersection %d] expected intersected mesh instance to be %d; got %d", expIndex, exp.meshInstance, target.meshInstance)
		}

		if target.triIndex != exp.triIndex {
			t.Fatalf("[intersection %d] expected intersected triangle index to be %d; got %d", expIndex, exp.triIndex, target.triIndex)
		}
	}
}

func morton2d(x uint32, y uint32) uint32 {
	mortonMasks2D := [5]uint32{0x0000FFFF, 0x00FF00FF, 0x0F0F0F0F, 0x33333333, 0x55555555}

	x = (x | x<<16) & mortonMasks2D[0]
	x = (x | x<<8) & mortonMasks2D[1]
	x = (x | x<<4) & mortonMasks2D[2]
	x = (x | x<<2) & mortonMasks2D[3]
	x = (x | x<<1) & mortonMasks2D[4]

	y = (y | y<<16) & mortonMasks2D[0]
	y = (y | y<<8) & mortonMasks2D[1]
	y = (y | y<<4) & mortonMasks2D[2]
	y = (y | y<<2) & mortonMasks2D[3]
	y = (y | y<<1) & mortonMasks2D[4]

	return x | (y << 1)
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

func createTestScene(t *testing.T) *scene.Scene {
	_, thisFile, _, _ := runtime.Caller(0)
	thisDir := filepath.Dir(thisFile)

	s, err := reader.ReadScene(filepath.Join(thisDir, "fixtures/cube.obj"))
	if err != nil {
		t.Fatal(err)
	}
	return s
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
