package texture

type TextureFormat uint32

const (
	Luminance8 TextureFormat = iota
	Luminance32F
	Rgba8
	Rgba32F
)
