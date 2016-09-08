package renderer

type Options struct {
	// Frame dims.
	FrameW uint32
	FrameH uint32

	// Number of indirect bounces.
	NumBounces uint32

	// Min bounces before applying russian roulette for path elimination.
	MinBouncesForRR uint32

	// Number of samples.
	SamplesPerPixel uint32

	// Exposure for tonemapping.
	Exposure float32

	// Device selection.
	BlackListedDevices []string
	ForcePrimaryDevice string
}
