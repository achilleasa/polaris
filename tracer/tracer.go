package tracer

import (
	"github.com/achilleasa/go-pathtrace/scene"
)

// A unit of work that is processed by a tracer.
type BlockRequest struct {
	// Block start row and height.
	BlockY uint
	BlockH uint

	// The number of emitted rays per traced pixel.
	SamplesPerPixel uint

	// The exposure value controls HDR -> LDR mapping.
	Exposure float32

	// A framebuffer where the rendered block is to be copied.
	RenderTarget []float32

	// A channel to signal on block completion with the number of completed rows.
	DoneChan chan<- uint

	// A channel to signal if an error occurs.
	ErrChan chan<- error
}

type Tracer interface {
	// Get tracer id.
	Id() string

	// Shutdown and cleanup tracer.
	Close()

	// Get the tracers computation speed estimate compared to a
	// baseline (cpu) impelemntation.
	SpeedEstimate() float32

	// Attach tracer to scene and start processing incoming block requests.
	Setup(sc *scene.Scene, frameW, frameH uint) error

	// Enqueue block request.
	Enqueue(blockReq BlockRequest)
}
