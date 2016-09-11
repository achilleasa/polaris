package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"

	"github.com/achilleasa/go-pathtrace/asset/scene/reader"
	"github.com/achilleasa/go-pathtrace/renderer"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

const (
	// Coefficients for converting delta cursor movements to yaw/pitch camera angles.
	mouseSensitivityX float32 = 0.005
	mouseSensitivityY float32 = 0.005

	// Camera movement speed
	cameraMoveSpeed float32 = 0.05
)

// Render a still frame.
func RenderFrame(ctx *cli.Context) error {
	setupLogging(ctx)

	opts := renderer.Options{
		FrameW:          uint32(ctx.Int("width")),
		FrameH:          uint32(ctx.Int("height")),
		SamplesPerPixel: uint32(ctx.Int("spp")),
		Exposure:        float32(ctx.Float64("exposure")),
		NumBounces:      uint32(ctx.Int("num-bounces")),
		MinBouncesForRR: uint32(ctx.Int("rr-bounces")),
		//
		BlackListedDevices: ctx.StringSlice("blacklist"),
		ForcePrimaryDevice: ctx.String("force-primary"),
	}

	if opts.MinBouncesForRR == 0 || opts.MinBouncesForRR >= opts.NumBounces {
		logger.Notice("disabling RR for path elimination")
		opts.MinBouncesForRR = opts.NumBounces + 1
	}

	// Load scene
	if ctx.NArg() != 1 {
		return errors.New("missing scene file argument")
	}

	sc, err := reader.ReadScene(ctx.Args().First())
	if err != nil {
		return err
	}

	// Update projection matrix
	sc.Camera.SetupProjection(float32(opts.FrameW) / float32(opts.FrameH))

	// Setup tracing pipeline
	pipeline := opencl.DefaultPipeline(opencl.NoDebug)
	pipeline.PostProcess = append(pipeline.PostProcess, opencl.SaveFrameBuffer(ctx.String("out")))

	// Create renderer
	r, err := renderer.NewDefault(sc, tracer.NaiveScheduler(), pipeline, opts)
	if err != nil {
		return err
	}
	defer r.Close()

	err = r.Render()
	if err != nil {
		return err
	}

	// Display stats
	displayFrameStats(r.Stats())

	return err
}

func displayFrameStats(stats renderer.FrameStats) {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Device", "Primary", "Block height", "% of frame", "Render time"})
	for _, stat := range stats.Tracers {
		table.Append([]string{
			stat.Id,
			fmt.Sprintf("%t", stat.IsPrimary),
			fmt.Sprintf("%d", stat.BlockH),
			fmt.Sprintf("%02.1f %%", stat.FramePercent),
			fmt.Sprintf("%s", stat.RenderTime),
		})
	}
	table.SetFooter([]string{"", "", "", "TOTAL", fmt.Sprintf("%s", stats.RenderTime)})

	table.Render()
	logger.Noticef("frame statistics\n%s", buf.String())
}

// Render scene using an interactive opengl view.
func RenderInteractive(ctx *cli.Context) error {
	runtime.LockOSThread()
	setupLogging(ctx)

	opts := renderer.Options{
		FrameW:          uint32(ctx.Int("width")),
		FrameH:          uint32(ctx.Int("height")),
		SamplesPerPixel: uint32(ctx.Int("spp")),
		Exposure:        float32(ctx.Float64("exposure")),
		NumBounces:      uint32(ctx.Int("num-bounces")),
		MinBouncesForRR: uint32(ctx.Int("rr-bounces")),
		//
		BlackListedDevices: ctx.StringSlice("blacklist"),
		ForcePrimaryDevice: ctx.String("force-primary"),
	}

	if opts.MinBouncesForRR == 0 || opts.MinBouncesForRR >= opts.NumBounces {
		logger.Notice("disabling RR for path elimination")
		opts.MinBouncesForRR = opts.NumBounces + 1
	}

	// Load scene
	if ctx.NArg() != 1 {
		return errors.New("missing scene file argument")
	}

	sc, err := reader.ReadScene(ctx.Args().First())
	if err != nil {
		return err
	}

	// Update projection matrix
	sc.Camera.SetupProjection(float32(opts.FrameW) / float32(opts.FrameH))

	// Setup tracing pipeline
	pipeline := opencl.DefaultPipeline(opencl.NoDebug)

	// Create renderer
	r, err := renderer.NewInteractive(sc, tracer.NaiveScheduler(), pipeline, opts)
	if err != nil {
		return err
	}

	// enter main loop
	return r.Render()
}
