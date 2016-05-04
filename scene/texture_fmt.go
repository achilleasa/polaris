package scene

type TextureFormat uint32

const (
	Luminance8 TextureFormat = iota
	Luminance32
	Rgb8
	Rgba8
	Rgb32
	Rgba32
)
