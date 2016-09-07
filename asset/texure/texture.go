package texture

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"unsafe"

	"github.com/achilleasa/go-pathtrace/asset"
	"github.com/achilleasa/openimageigo"
)

// A texture image and its metadata.
type Texture struct {
	Format Format

	Width  uint32
	Height uint32

	Data []byte
}

// Create a new texture from a Resource.
func New(res *asset.Resource) (*Texture, error) {
	var pathToFile string

	// If this is a remote Resource save it to a temp file so that oiio can load it
	if res.IsRemote() {
		pathToFile = os.TempDir() + "/" + res.RemotePath()
		f, err := os.Create(pathToFile)
		if err != nil {
			return nil, err
		}
		defer os.Remove(pathToFile)
		_, err = io.Copy(f, res)
		f.Close()
		if err != nil {
			return nil, err
		}
	} else {
		pathToFile = res.Path()
	}

	input, err := oiio.OpenImageInput(pathToFile)
	if err != nil {
		return nil, err
	}
	defer input.Close()

	// Get image spec and check whether we support this format
	spec := input.Spec()

	// Validate channel count and depth
	if spec.NumChannels() != 1 && spec.NumChannels() != 3 && spec.NumChannels() != 4 {
		return nil, fmt.Errorf("texture: unsupported channel count %d while loading %s", spec.NumChannels(), res.Path())
	}
	if spec.Depth() != 1 {
		return nil, fmt.Errorf("texture: unsupported depth %d while loading %s", spec.Depth(), res.Path())
	}

	// Select tex format
	var texFmt Format
	var convertTo oiio.TypeDesc
	switch spec.Format() {
	case oiio.TypeUint8:
		convertTo = oiio.TypeUint8

		switch spec.NumChannels() {
		case 1:
			texFmt = Luminance8
		default:
			texFmt = Rgba8
		}
	default:
		convertTo = oiio.TypeFloat
		switch spec.NumChannels() {
		case 1:
			texFmt = Luminance32F
		default:
			texFmt = Rgba32F
		}
	}

	// Read data
	imgData, err := input.ReadImageFormat(convertTo, nil)
	if err != nil {
		return nil, fmt.Errorf("texture: could not read data from %s: %s", res.Path(), err.Error())
	}

	// Setup texture
	texture := &Texture{
		Format: texFmt,
		Width:  uint32(spec.Width()),
		Height: uint32(spec.Height()),
	}

	// Cast data to []byte
	switch t := imgData.(type) {
	case []uint8:
		// convert to rgba as this makes addressing in opencl much easier
		if spec.NumChannels() == 3 {
			tData := make([]byte, texture.Width*texture.Height*4)
			wOffset := 0
			for rOffset := 0; rOffset < len(t); {
				tData[wOffset] = t[rOffset]
				tData[wOffset+1] = t[rOffset+1]
				tData[wOffset+2] = t[rOffset+2]
				tData[wOffset+3] = 255

				rOffset += 3
				wOffset += 4
			}

			t = tData
		}

		texture.Data = t
	case []float32:
		// convert to rgba as this makes addressing in opencl much easier
		if spec.NumChannels() == 3 {
			tData := make([]float32, texture.Width*texture.Height*4)
			wOffset := 0
			for rOffset := 0; rOffset < len(t); {
				tData[wOffset] = t[rOffset]
				tData[wOffset+1] = t[rOffset+1]
				tData[wOffset+2] = t[rOffset+2]
				tData[wOffset+3] = 1.0

				rOffset += 3
				wOffset += 4
			}

			t = tData
		}

		// Fetch slice header and adjust len/capacity (1 float32 = 4 bytes)
		header := *(*reflect.SliceHeader)(unsafe.Pointer(&t))
		header.Len <<= 2
		header.Cap <<= 2

		// Convert to a []byte
		texture.Data = *(*[]byte)(unsafe.Pointer(&header))
	}

	return texture, nil
}
