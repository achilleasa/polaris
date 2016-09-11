package renderer

import "time"

type TracerStat struct {
	// The tracer id.
	Id string

	// True if this is the primary tracer
	IsPrimary bool

	// The block height and the percentage of total frame area it represents.
	BlockH       uint32
	FramePercent float32

	// Render time for assigned block
	RenderTime time.Duration
}

type FrameStats struct {
	// Individual tracer stats.
	Tracers []TracerStat

	// Total render time for entire frame.
	RenderTime time.Duration
}
