package cmd

import (
	"fmt"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/scene/reader"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/integrator"
	"github.com/achilleasa/go-pathtrace/types"
	"github.com/urfave/cli"
)

const (
	frameW uint32 = 1024
	frameH uint32 = 1024
)

func findDevice(names []string) (*device.Device, error) {
	for _, name := range names {
		devList, err := device.SelectDevices(device.AllDevices, name)
		if err != nil {
			logger.Error(err)
			return nil, err
		}

		if len(devList) != 0 {
			return devList[0], nil
		}
	}

	return nil, fmt.Errorf("no suitable device found")
}

func Debug(ctx *cli.Context) error {
	setupLogging(ctx)

	dev, err := findDevice([]string{"AMD", "Iris", "CPU"})
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Noticef(`using device "%s"`, dev.Name)

	in := integrator.NewMonteCarloIntegrator(dev)
	err = in.Init()
	if err != nil {
		logger.Error(err)
		return err
	}
	defer in.Close()

	sc, err := reader.ReadScene("../go-pathtrace-scenes/crytek-sponza/sponza.zip")
	if err != nil {
		logger.Error(err)
		return err
	}

	err = in.UploadSceneData(sc)
	if err != nil {
		logger.Error(err)
		return err
	}

	blockReq := &tracer.BlockRequest{
		FrameW: frameW,
		FrameH: frameH,
		BlockY: 0,
		BlockH: frameH,
	}

	camera := scene.NewCamera(45)
	camera.Position = types.Vec3{-1053.478, 92.0336, -22.42906}
	camera.Up = types.Vec3{0, 1, 0}
	camera.LookAt = types.Vec3{-1, 0, 0}

	camera.SetupProjection(float32(blockReq.FrameW) / float32(blockReq.FrameH))
	camera.Update()

	err = in.UploadCameraData(camera)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = in.ResizeOutputFrame(blockReq.FrameW, blockReq.FrameH)
	if err != nil {
		logger.Error(err)
		return err
	}

	_, err = in.GeneratePrimaryRays(blockReq)
	if err != nil {
		logger.Error(err)
		return err
	}

	elapsedTime, err := in.RayIntersectionQuery(blockReq.FrameW * blockReq.FrameH)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Noticef("Elapsed time: %d ms", elapsedTime.Nanoseconds()/1e6)

	err = in.DebugIntersections(blockReq.FrameW, blockReq.FrameH, "debug.png")
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}
