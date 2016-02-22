package renderer

import (
	"image"
	"math"
	"sync"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/go-pathtrace/tracer"
)

type Renderer struct {
	// A lock for synchronizing access to the framebuffer.
	sync.Mutex

	// Renderer frame dims.
	frameW uint
	frameH uint

	// The scene to be rendered.
	scene *scene.Scene

	// A buffered channel for receiving block completions.
	tracerDoneChan chan uint

	// A channel for receiving tracer errors.
	tracerErrChan chan error

	// A frame buffer used as each tracer's render target.
	frameBuffer []float32

	// An accumulation buffer where each frame's output is blended with
	// the previous frames' output. This enables progressive scene rendering.
	accumBuffer []float32

	// A weight defining how to blend the contents of the accumulation
	// buffer with the contents of the frame buffer. A value of 0 means
	// that the frameBuffer always overwrites the contents of the accumBuffer.
	blendWeight float32

	// A list of assigned block height for each tracer
	blockAssignment []uint

	// The list of attached tracers
	Tracers []tracer.Tracer
}

// Create a new renderer
func NewRenderer(frameW, frameH uint, sc *scene.Scene) *Renderer {
	return &Renderer{
		frameW:          frameW,
		frameH:          frameH,
		tracerDoneChan:  make(chan uint, frameH),
		tracerErrChan:   make(chan error, 0),
		frameBuffer:     make([]float32, frameW*frameH*4),
		accumBuffer:     make([]float32, frameW*frameH*4),
		blendWeight:     0,
		blockAssignment: make([]uint, 0),
		Tracers:         make([]tracer.Tracer, 0),
		scene:           sc,
	}
}

// Shutdown and cleanup renderer and all connected tracers. This function will
// block if a frame is currently being rendered.
func (r *Renderer) Close() {
	r.Lock()
	defer r.Unlock()

	for _, tr := range r.Tracers {
		tr.Close()
	}
	r.Tracers = make([]tracer.Tracer, 0)
}

// Add a tracer to the renderer's tracer pool.
func (r *Renderer) AddTracer(tr tracer.Tracer) error {
	err := tr.Attach(r.scene, r.frameBuffer, r.frameW, r.frameH)
	if err != nil {
		return err
	}
	r.Tracers = append(r.Tracers, tr)
	r.blockAssignment = append(r.blockAssignment, 0)
	return nil
}

// Clear the frame buffer. This function will block if a frame is currently
// being rendered.
func (r *Renderer) ClearFrame() {
	r.Lock()
	defer r.Unlock()

	for i := range r.frameBuffer {
		r.frameBuffer[i] = 0
		r.accumBuffer[i] = 0
	}
	r.blendWeight = 0
}

// Generate a RGBA image for the current accumulation buffer contents. If the
// rendering has completed then this function will return the final frame;
// otherwise it will return the currently rendered frame bits. This function will
// lock the renderer while it copies out frame data it shouldn't be called too
// often while a render request is in progress.
func (r *Renderer) Frame() *image.RGBA {
	r.Lock()
	defer r.Unlock()

	img := image.NewRGBA(image.Rect(0, 0, int(r.frameW), int(r.frameH)))
	pixelCount := int(r.frameW * r.frameH)
	offset := 0
	for i := 0; i < pixelCount; i++ {
		img.Pix[offset] = uint8(r.accumBuffer[offset]*255.0 + 0.5)
		img.Pix[offset+1] = uint8(r.accumBuffer[offset+1]*255.0 + 0.5)
		img.Pix[offset+2] = uint8(r.accumBuffer[offset+2]*255.0 + 0.5)
		img.Pix[offset+3] = 255 // alpha channel
		offset += 4
	}

	return img
}

// Distribute the frame rows between the pooled tracers.
func (r *Renderer) assignTracerBlocks() {
	// Get speed estimate for each tracer and distribute rows accordingly
	var totalSpeedEstimate float32 = 0.0
	for _, tr := range r.Tracers {
		totalSpeedEstimate += tr.SpeedEstimate()
	}
	scaler := float32(r.frameH) / totalSpeedEstimate

	for idx, tr := range r.Tracers {
		r.blockAssignment[idx] = uint(math.Ceil(float64(tr.SpeedEstimate() * scaler)))
	}
}

// Blend contents of the frame buffer into the accumulation buffer.
func (r *Renderer) updateAccumulationBuffer() {
	pixelCount := int(r.frameW * r.frameH)
	offset := 0
	oneMinusWeight := 1.0 - r.blendWeight
	weight := r.blendWeight
	for i := 0; i < pixelCount; i++ {
		r.accumBuffer[offset] = oneMinusWeight*r.frameBuffer[offset] + weight*r.accumBuffer[offset]
		r.accumBuffer[offset+1] = oneMinusWeight*r.frameBuffer[offset+1] + weight*r.accumBuffer[offset+1]
		r.accumBuffer[offset+2] = oneMinusWeight*r.frameBuffer[offset+2] + weight*r.accumBuffer[offset+2]
		offset += 4
	}
}

// Render scene.
func (r *Renderer) RenderFrame() error {
	r.Lock()
	defer r.Unlock()

	if r.scene == nil {
		return ErrSceneNotDefined
	}
	if r.scene.Camera == nil {
		return ErrCameraNotDefined
	}
	if len(r.Tracers) == 0 {
		return ErrNoTracers
	}

	// Update block assignments
	r.assignTracerBlocks()

	// Setup common block request values
	var blockReq tracer.BlockRequest
	blockReq.FrameW = r.frameW
	blockReq.FrameH = r.frameH
	blockReq.DoneChan = r.tracerDoneChan
	blockReq.ErrChan = r.tracerErrChan
	blockReq.SamplesPerPixel = 1
	blockReq.Exposure = r.scene.Camera.Exposure

	// Enqueue work units
	var pendingRows uint = 0
	for idx, tr := range r.Tracers {
		blockReq.BlockY = pendingRows
		blockReq.BlockH = r.blockAssignment[idx]
		tr.Enqueue(blockReq)

		pendingRows += blockReq.BlockH
	}

	// Wait for all rows to be completed
	for {
		select {
		case completedRows := <-r.tracerDoneChan:
			pendingRows -= completedRows
			if pendingRows == 0 {
				r.updateAccumulationBuffer()
				return nil
			}
		case err := <-r.tracerErrChan:
			return err
		}
	}
}
