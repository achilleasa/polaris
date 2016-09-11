package renderer

import (
	"fmt"
	"sync"

	"github.com/achilleasa/go-pathtrace/asset/scene"
	"github.com/achilleasa/go-pathtrace/tracer"
	"github.com/achilleasa/go-pathtrace/tracer/opencl"
	"github.com/achilleasa/go-pathtrace/types"
	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.1/glfw"
)

const (
	// Coefficients for converting delta cursor movements to yaw/pitch camera angles.
	mouseSensitivityX float32 = 0.005
	mouseSensitivityY float32 = 0.005

	// Camera movement speed
	cameraMoveSpeed float32 = 0.05
)

const (
	leftMouseButton  = 0
	rightMouseButton = 1
)

// An interactive opengl-based renderer.
type interactiveGLRenderer struct {
	*defaultRenderer

	accumulatedSamples uint32

	// opengl handles
	window *glfw.Window
	texFbo uint32

	// state
	lastCursorPos types.Vec2
	mousePressed  [2]bool
	camera        *scene.Camera

	// mutex for synchronizing updates
	sync.Mutex

	// Display options
	showBlockAllocations bool
}

// Create a new interactive opengl renderer using the specified block scheduler and tracing pipeline.
func NewInteractive(sc *scene.Scene, scheduler tracer.BlockScheduler, pipeline *opencl.Pipeline, opts Options) (Renderer, error) {
	// Add an extra pipeline step to copy framebuffer data to an opengl texture
	pipeline.PostProcess = append(pipeline.PostProcess, opencl.CopyFrameBufferToOpenGLTexture())

	base, err := NewDefault(sc, scheduler, pipeline, opts)
	if err != nil {
		return nil, err
	}

	r := &interactiveGLRenderer{
		defaultRenderer: base.(*defaultRenderer),
		camera:          sc.Camera,
	}

	err = r.initGL(opts)
	if err != nil {
		r.Close()
		return nil, err
	}

	return r, nil
}

func (r *interactiveGLRenderer) Close() {
	if r.window != nil {
		r.window.SetShouldClose(true)
	}
	if r != nil {
		r.Close()
	}
}

func (r *interactiveGLRenderer) initGL(opts Options) error {
	var err error
	if err = glfw.Init(); err != nil {
		return fmt.Errorf("failed to initialize glfw: %s", err.Error())
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	r.window, err = glfw.CreateWindow(int(opts.FrameW), int(opts.FrameH), "go-pathtrace", nil, nil)
	if err != nil {
		return fmt.Errorf("could not create opengl window: %s", err.Error())
	}
	r.window.MakeContextCurrent()

	if err = gl.Init(); err != nil {
		return fmt.Errorf("could not init opengl: %s", err.Error())
	}

	// Setup texture for image data
	var fbTexture uint32
	gl.GenTextures(1, &fbTexture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, fbTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(opts.FrameW), int32(opts.FrameH), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	// Attach texture to FBO
	gl.GenFramebuffers(1, &r.texFbo)
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, r.texFbo)
	gl.FramebufferTexture2D(gl.READ_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fbTexture, 0)
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)

	// Bind event callbacks
	r.window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	r.window.SetKeyCallback(r.onKeyEvent)
	r.window.SetMouseButtonCallback(r.onMouseEvent)
	r.window.SetCursorPosCallback(r.onCursorPosEvent)

	return nil
}

func (r *interactiveGLRenderer) Render() error {
	for !r.window.ShouldClose() {
		glfw.PollEvents()

		// Don't do anything if we don't require addittional samples
		if r.options.SamplesPerPixel != 0 && r.accumulatedSamples == r.defaultRenderer.options.SamplesPerPixel {
			continue
		}

		// Render next frame
		r.Lock()
		err := r.renderFrame(r.accumulatedSamples)
		r.accumulatedSamples++
		if err != nil {
			r.Unlock()
			return err
		}

		// Copy texture data to framebuffer
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, r.texFbo)
		gl.BlitFramebuffer(0, 0, int32(r.options.FrameW), int32(r.options.FrameH), 0, 0, int32(r.options.FrameW), int32(r.options.FrameH), gl.COLOR_BUFFER_BIT, gl.LINEAR)
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, 0)

		r.window.SwapBuffers()
		r.Unlock()
	}
	return nil
}

func (r *interactiveGLRenderer) onKeyEvent(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action != glfw.Press && action != glfw.Repeat {
		return
	}

	var moveDir scene.CameraDirection
	switch key {
	case glfw.KeyEscape:
		r.window.SetShouldClose(true)
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
	r.camera.Move(moveDir, speedScaler*cameraMoveSpeed)
	r.updateCamera()
}

func (r *interactiveGLRenderer) onMouseEvent(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
	if button != glfw.MouseButtonLeft && button != glfw.MouseButtonRight {
		return
	}

	r.mousePressed[leftMouseButton] = false
	r.mousePressed[rightMouseButton] = false

	if action == glfw.Press {
		xPos, yPos := w.GetCursorPos()
		r.lastCursorPos[0], r.lastCursorPos[1] = float32(xPos), float32(yPos)

		buttonIndex := leftMouseButton
		if button == glfw.MouseButtonRight {
			buttonIndex = rightMouseButton
		}

		r.mousePressed[buttonIndex] = true
	}
}

func (r *interactiveGLRenderer) onCursorPosEvent(w *glfw.Window, xPos, yPos float64) {
	if !r.mousePressed[leftMouseButton] && !r.mousePressed[rightMouseButton] {
		return
	}

	// Calculate delta movement and apply mouse sensitivity
	newPos := types.Vec2{float32(xPos), float32(yPos)}
	delta := r.lastCursorPos.Sub(newPos)
	delta[0] *= mouseSensitivityX
	delta[1] *= mouseSensitivityY
	r.lastCursorPos = newPos

	if r.mousePressed[leftMouseButton] {
		// The left mouse button rotates lookat around eye
		r.camera.Pitch = delta[1]
		r.camera.Yaw = delta[0]
		r.camera.Update()
		r.updateCamera()
	}
}

func (r *interactiveGLRenderer) updateCamera() {
	r.Lock()
	defer r.Unlock()

	for _, tr := range r.tracers {
		tr.UpdateState(tracer.Asynchronous, tracer.CameraData, r.camera)
	}

	r.accumulatedSamples = 0
}
