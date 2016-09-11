package renderer

type Renderer interface {
	// Render frame.
	Render() error

	// Shutdown renderer and any attached tracer.
	Close()

	// Get render statistics.
	Stats() FrameStats
}
