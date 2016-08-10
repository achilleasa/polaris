package cmd

import (
	"fmt"

	"github.com/achilleasa/go-pathtrace/scene/reader"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl"
	"github.com/achilleasa/go-pathtrace/tracer/opencl/device"
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

	// Load scene
	sc, err := reader.ReadScene(ctx.Args().First())

	if err != nil {
		logger.Error(err)
		return err
	}

	// Update projection matrix
	sc.Camera.SetupProjection(float32(frameW) / float32(frameH))

	// Setup tracer
	dev, err := findDevice([]string{ /*"CPU", */ "AMD", "CPU", "Iris", "CPU"})
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Noticef(`using device "%s"`, dev.Name)

	// Setup pipeline
	pipeline := opencl.DefaultPipeline(
		opencl.PrimaryRayIntersectionDepth|opencl.PrimaryRayIntersectionNormals|
			opencl.VisibleEmissiveSamples|opencl.Throughput|opencl.Accumulator|
			opencl.FrameBuffer,
		6,
		1.5,
	)

	tr, _ := opencl.NewTracer("tr-0", dev, pipeline)
	err = tr.Init()
	if err != nil {
		logger.Error(err)
		return fmt.Errorf("error initializing opencl device")
	}
	defer tr.Close()

	// Queue state changes
	tr.UpdateState(tracer.Synchronous, tracer.FrameDimensions, [2]uint32{frameW, frameH})
	tr.UpdateState(tracer.Synchronous, tracer.SceneData, sc)
	tr.UpdateState(tracer.Synchronous, tracer.CameraData, sc.Camera)

	// Setup block
	blockReq := &tracer.BlockRequest{
		FrameW:          frameW,
		FrameH:          frameH,
		BlockX:          0,
		BlockY:          0,
		BlockW:          frameW,
		BlockH:          frameH,
		SamplesPerPixel: 1,
	}

	// Render
	_, err = tr.Trace(blockReq)
	if err != nil {
		logger.Error(err)
		return err
	}
	_, err = tr.SyncFramebuffer(blockReq)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}
