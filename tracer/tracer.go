package tracer

type ChangeType uint8

const (
	SetBvhNodes ChangeType = iota
	SetPrimitivies
	SetMaterials
	SetEmissiveLightIndices
	UpdateCamera
)

// A unit of work that is processed by a tracer.
type BlockRequest struct {
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
	DoneChan chan<- uint32

	// A channel to signal if an error occurs.
	ErrChan chan<- error
}

// Tracer statistics.
type Stats struct {
	// The rendered block height
	BlockH uint32

	// The time for rendering this block (in nanoseconds)
	BlockTime int64
}

type Tracer interface {
	// Get tracer id.
	Id() string

	// Shutdown and cleanup tracer.
	Close()

	// Get the tracers computation speed estimate compared to a
	// baseline (cpu) impelemntation.
	SpeedEstimate() float32

	// Setup the tracer.
	Setup(frameW, frameH uint32, accumBuffer []float32, frameBuffer []uint8) error

	// Enqueue block request.
	Enqueue(BlockRequest)

	// Append a change to the tracer's update buffer.
	AppendChange(ChangeType, interface{})

	// Apply all pending changes from the update buffer.
	ApplyPendingChanges() error

	// Retrieve last frame statistics.
	Stats() *Stats
}
