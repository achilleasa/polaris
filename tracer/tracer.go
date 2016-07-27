package tracer

import (
	"time"

	"github.com/achilleasa/go-pathtrace/scene"
)

// A unit of work that is processed by a tracer.
type BlockRequest struct {
	// Frame dimensions
	FrameW uint32
	FrameH uint32

	// Block start row and height.
	BlockY uint32
	BlockH uint32

	// The number of emitted rays per traced pixel.
	SamplesPerPixel uint32

	// The exposure value controls HDR -> LDR mapping.
	Exposure float32

	// A random seed value for the tracer's random number generator.
	Seed uint32

	// Number of sequential rendered frames from current camera position.
	FrameCount uint32

	// A channel to signal on block completion with the number of completed rows.
	DoneChan chan uint32

	// A channel to signal if an error occurs.
	ErrChan chan error
}

// Tracer statistics.
type Stats struct {
	// The rendered block dimensions.
	BlockW uint32
	BlockH uint32

	// The time for applying queued scene changes.
	UpdateTime time.Duration

	// The time for rendering this block
	RenderTime time.Duration
}

type Flag uint8

// Tracer or-able flag list.
const (
	// Locally attached device
	Local Flag = 1 << iota

	// Remote device.
	Remote = 1 << iota

	// Supports GL interop.
	GLInterop = 1 << iota
)

type UpdateType uint8

// Supported update types.
const (
	UpdateScene UpdateType = iota
	UpdateCamera
)

// Alias for a function that can be used to attach a pipeline stage to the tracer.
type Stage func(tracer Tracer) error

type Tracer interface {
	// Get tracer id.
	Id() string

	// Get tracer flags.
	Flags() Flag

	// Get the computation speed estimate (in GFlops).
	Speed() uint32

	// Initialize tracer.
	Init(frameW, frameH uint32, stages ...Stage) error

	// Shutdown and cleanup tracer.
	Close()

	// Enqueue block request.
	Enqueue(BlockRequest)

	// Queue an update.
	Update(UpdateType, interface{})

	// Retrieve last frame statistics.
	Stats() *Stats

	// Upload scene data.
	UploadSceneData(sc *scene.Scene) error

	// Upload camera data.
	UploadCameraData(sc *scene.Camera) error
}
