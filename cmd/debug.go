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

	// Setup tracer
	dev, err := findDevice([]string{"AMD", "CPU", "Iris", "CPU"})
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Noticef(`using device "%s"`, dev.Name)

	tr, _ := opencl.NewTracer("tr-0", dev)
	err = tr.Init(
		frameW,
		frameH,
		// Stages
		opencl.ClearAccumulator(),
		opencl.PerspectiveCamera(),
		opencl.MonteCarloIntegrator(3),
		opencl.TonemapSimpleReinhard(3.5),
		opencl.DebugFrameBuffer("debug-fb.png"),
	)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer tr.Close()

	blockReq := tracer.BlockRequest{
		FrameW:   frameW,
		FrameH:   frameH,
		BlockY:   0,
		BlockH:   frameH,
		DoneChan: make(chan uint32, 0),
		ErrChan:  make(chan error, 0),
	}

	// Upload data
	err = tr.UploadSceneData(sc)
	if err != nil {
		logger.Error(err)
		return err
	}

	sc.Camera.SetupProjection(float32(blockReq.FrameW) / float32(blockReq.FrameH))
	err = tr.UploadCameraData(sc.Camera)
	if err != nil {
		logger.Error(err)
		return err
	}

	// Run pipeline
	tr.Enqueue(blockReq)
	select {
	case <-blockReq.DoneChan:
		logger.Notice("Done")
		return nil
	case err := <-blockReq.ErrChan:
		logger.Error(err)
		return err
	}
}
