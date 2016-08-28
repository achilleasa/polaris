package material

import "regexp"

var (
	// supported image extensions regex
	supportedImageRegex = regexp.MustCompile(`(?i)\.(?:jpg|jpeg|gif|png|tga|tiff|bmp|pnm|hdr|exr|webp)$`)
)
