package cmd

import (
	"bytes"
	"errors"
	"fmt"

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

	_, err = r.Render(0)
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

/*
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

// Load scene and setup renderer.
func setupRenderer(ctx *cli.Context, invertY bool) (*renderer.Renderer, *scene.Camera) {
	// Get render params
	frameW := uint32(ctx.Int("width"))
	frameH := uint32(ctx.Int("height"))
	exposure := float32(ctx.Float64("exposure"))

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
	sc, err := reader.ReadScene(ctx.Args().First())
	if err != nil {
		logger.Printf("error: %s", err.Error())
		os.Exit(1)
	}

	// Init renderer
	r := renderer.NewRenderer(frameW, frameH, sc)
	r.Exposure = exposure
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
	logger.Printf("setup %d tracers in %d ms", attachedTracers, time.Since(start).Nanoseconds()/1000000)

	// Setup camera
	sc.Camera.InvertY = invertY
	sc.Camera.SetupProjection(float32(frameW) / float32(frameH))
	r.UpdateCamera()

	return r, sc.Camera
}

// Render still frame.
func RenderFrame(ctx *cli.Context) {
	setupLogging(ctx)

	// Get render params
	spp := uint32(ctx.Int("spp"))
	imgFile := ctx.String("out")

	// Render frame
	r, _ := setupRenderer(ctx, false)
	defer r.Close()
	logger.Print("rendering frame")
	start := time.Now()
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

	start = time.Now()
	err = png.Encode(f, frame)
	if err != nil {
		logger.Printf("error encoding png file: %s", err.Error())
		os.Exit(1)
	}
	logger.Printf("wrote frame to %s in %d ms", imgFile, time.Since(start).Nanoseconds()/1000000)
}

// Use opengl to render a continuously updating view of the renderer's frame
// buffer contents.
func RenderInteractive(ctx *cli.Context) {
	setupLogging(ctx)

	// Get render params
	frameW := int32(ctx.Int("width"))
	frameH := int32(ctx.Int("height"))

	// Init opengl
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		logger.Print("error: failed to initialize glfw: %s", err.Error())
		os.Exit(1)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	window, err := glfw.CreateWindow(int(frameW), int(frameH), "go-pathtrace", nil, nil)
	if err != nil {
		logger.Printf("error: could not create opengl window: %s", err.Error())
		os.Exit(1)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		logger.Printf("error: could not init opengl: %s", err.Error())
		os.Exit(1)
	}

	// Setup texture for image data
	var fbTexture uint32
	gl.GenTextures(1, &fbTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, fbTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, frameW, frameH, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	// Attach texture to FBO
	var texFbo uint32
	gl.GenFramebuffers(1, &texFbo)
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, texFbo)
	gl.FramebufferTexture2D(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fbTexture, 0)
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)

	// Init renderer. glBlitFrameBuffer copies the texture with the Y axis
	// inverted. To correct this, we adjust the camera frustrum corners
	// so that the scene is rendered upside-down.
	r, camera := setupRenderer(ctx, true)
	defer r.Close()

	// Enable mouse cursor and register input callbacks
	var lastCursorPos types.Vec2
	var mousePressed bool
	window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action != glfw.Press && action != glfw.Repeat {
			return
		}

		var moveDir scene.CameraDirection
		switch key {
		case glfw.KeyEscape:
			window.SetShouldClose(true)
		case glfw.KeyUp:
			moveDir = scene.Forward
		case glfw.KeyDown:
			moveDir = scene.Backward
		case glfw.KeyLeft:
			moveDir = scene.Left
		case glfw.KeyRight:
			moveDir = scene.Right
		default:
			return

		}

		// Double speed if shift is pressed
		var speedScaler float32 = 1.0
		if (mods & glfw.ModShift) == glfw.ModShift {
			speedScaler = 2.0
		}
		camera.Move(moveDir, speedScaler*cameraMoveSpeed)
		r.UpdateCamera()
	})
	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
		if button != glfw.MouseButtonLeft {
			return
		}

		if action == glfw.Press {
			xPos, yPos := w.GetCursorPos()
			lastCursorPos[0], lastCursorPos[1] = float32(xPos), float32(yPos)
			mousePressed = true
		} else {
			mousePressed = false
		}
	})
	window.SetCursorPosCallback(func(w *glfw.Window, xPos, yPos float64) {
		if !mousePressed {
			return
		}

		// Calculate delta movement and apply mouse sensitivity
		newPos := types.Vec2{float32(xPos), float32(yPos)}
		delta := lastCursorPos.Sub(newPos)
		delta[0] *= mouseSensitivityX
		delta[1] *= mouseSensitivityY
		lastCursorPos = newPos

		camera.Pitch = delta[1]
		camera.Yaw = delta[0]
		camera.Update()
		r.UpdateCamera()
	})

	// Enter render loop
	var frame *image.RGBA
	for !window.ShouldClose() {
		frame, err = r.Render(renderer.AutoSamplesPerPixel)
		if err != nil {
			logger.Printf("error rendering frame: %s", err.Error())
			os.Exit(1)
		}

		// Update texture with frame data
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, frameW, frameH, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&frame.Pix[0]))

		// Copy texture data to framebuffer
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, texFbo)
		gl.BlitFramebuffer(0, 0, frameW, frameH, 0, 0, frameW, frameH, gl.COLOR_BUFFER_BIT, gl.LINEAR)
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)

		window.SwapBuffers()
		glfw.PollEvents()
	}
}*/
