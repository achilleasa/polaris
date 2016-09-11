package tracer

import "time"

// A unit of work that is processed by a tracer.
type BlockRequest struct {
	// Frame dimensions.
	FrameW uint32
	FrameH uint32

	// Block dimensions.
	BlockX uint32
	BlockY uint32
	BlockW uint32
	BlockH uint32

	// The number of emitted rays per traced pixel.
	SamplesPerPixel uint32

	// The number of bounces to trace.
	NumBounces uint32

	// Number of bounces before applying russian roulette to terminate paths.
	MinBouncesForRR uint32

	// The exposure value controls HDR -> LDR mapping.
	Exposure float32

	// A random seed value for the tracer's random number generator.
	Seed uint32

	// Number of sequential rendered frames from current camera position.
	AccumulatedSamples uint32
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
	Remote

	// CPU-based
	CpuDevice
)

type UpdateMode uint8

// Supported update type.
const (
	Synchronous UpdateMode = iota
	Asynchronous
)

type ChangeType uint8

// Supported update data.
const (
	FrameDimensions ChangeType = iota
	SceneData
	CameraData
)

type Tracer interface {
	// Get tracer id.
	Id() string

	// Get tracer flags.
	Flags() Flag

	// Get the computation speed estimate (in GFlops).
	Speed() uint32

	// Initialize tracer.
	Init() error

	// Shutdown and cleanup tracer.
	Close()

	// Retrieve last frame statistics.
	Stats() *Stats

	// Update tracer state.
	UpdateState(UpdateMode, ChangeType, interface{}) (time.Duration, error)

	// Process block request.
	Trace(*BlockRequest) (time.Duration, error)

	// Merge accumulator output from another tracer into this tracer's buffer.
	MergeOutput(Tracer, *BlockRequest) (time.Duration, error)

	// Run post-process filters to the accumulated trace data and
	// update the output frame buffer.
	SyncFramebuffer(*BlockRequest) (time.Duration, error)
}
