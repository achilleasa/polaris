package cmd

import (
	"image/png"
	"log"
	"os"
	"strings"
	"time"

	"github.com/achilleasa/go-pathtrace/renderer"
	"github.com/achilleasa/go-pathtrace/scene/io"
	"github.com/achilleasa/go-pathtrace/tracer/opencl"
	"github.com/codegangsta/cli"
)

// Return the available opencl devices after applying the blacklist filters.
func filteredDeviceList(ctx *cli.Context) []opencl.Device {
	filteredList := make([]opencl.Device, 0)
	blackList := ctx.StringSlice("blacklist")

	var keep bool
	for _, platformInfo := range opencl.GetPlatformInfo() {
		for _, device := range platformInfo.Devices {
			keep = true
			for _, text := range blackList {
				if strings.Contains(device.Name, text) {
					keep = false
					break
				}
			}
			if keep {
				filteredList = append(filteredList, device)
			}
		}
	}

	return filteredList
}

// Render still frame.
func RenderFrame(ctx *cli.Context) {
	// Get render params
	frameW := uint32(ctx.Int("width"))
	frameH := uint32(ctx.Int("height"))
	spp := uint32(ctx.Int("spp"))
	imgFile := ctx.String("out")

	if ctx.NArg() == 0 {
		logger.Print("error: missing scene file argument")
	}

	// Get list of opencl devices to use
	deviceList := filteredDeviceList(ctx)
	if len(deviceList) == 0 {
		logger.Print("error: no available opencl devices")
		os.Exit(1)
	}

	// Load scene
	sc, err := io.ReadScene(ctx.Args().First())
	if err != nil {
		logger.Printf("error: %s", err.Error())
		return
	}

	// Init renderer
	r := renderer.NewRenderer(frameW, frameH, sc)
	logger.Print("compiling opencl kernels and uploading scene data")
	start := time.Now()
	attachedTracers := 0
	for _, device := range deviceList {
		tr, err := device.GetTracer()
		if err != nil {
			logger.Printf("skipping device %s due to init error %s", device.Name, err.Error())
			continue
		}

		err = r.AddTracer(tr)
		logger.Printf(` attaching tracer for device "%s"`, device.Name)
		if err != nil {
			logger.Printf("skipping device %s due to renderer attachment error %s", device.Name, err.Error())
			continue
		}
		attachedTracers++
	}
	if attachedTracers == 0 {
		logger.Printf("error: no tracers attached")
		os.Exit(1)
	}
	logger.Printf("setup %d traces in %d ms", attachedTracers, time.Since(start).Nanoseconds()/1000000)

	// Setup camera
	sc.Camera.SetupProjection(float32(frameW) / float32(frameH))
	r.UpdateCamera()

	// Render frame
	log.Print("rendering frame")
	start = time.Now()
	frame, err := r.Render(renderer.SamplesPerPixel(spp))
	if err != nil {
		logger.Printf("error rendering frame: %s", err.Error())
		os.Exit(1)
	}
	logger.Printf("rendered frame in %d ms", time.Since(start).Nanoseconds()/1000000)

	// Export PNG
	f, err := os.Create(imgFile)
	if err != nil {
		logger.Printf("error rendering frame: %s", err.Error())
		os.Exit(1)
	}
	defer f.Close()
	err = png.Encode(f, frame)
	if err != nil {
		logger.Printf("error encoding png file: %s", err.Error())
		os.Exit(1)
	}
}
