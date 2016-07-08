package reader

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unsafe"

	"github.com/achilleasa/go-pathtrace/scene"
	"github.com/achilleasa/openimageigo"
)

// The resource class wraps a streamable file or remote resource.
type resource struct {
	io.ReadCloser
	url *url.URL
}

// Returns the path to this resource.
func (r *resource) Path() string {
	return r.url.String()
}

// Returns true if the resource is streamed over http/https.
func (r *resource) IsRemote() bool {
	return r.url.Scheme != ""
}

// Create a new resource data stream. If relTo is specified and pathToResource
// does not define a scheme, then the path to the new resource will be generated
// by concatenating the base path of relTo and pathToResource.
//
// This function can handle http/https URLs by delegating to the net/http package.
// The caller must make sure to close the returned io.ReadCloser to prevent mem leaks.
func newResource(pathToResource string, relTo *resource) (*resource, error) {
	// Replace forward slashes with backslaces and try parsing as a URL
	url, err := url.Parse(strings.Replace(pathToResource, `\`, `/`, -1))
	if err != nil {
		return nil, err
	}

	// If this is a relative url, clone parent url and adjust its path
	if url.Scheme == "" && relTo != nil {
		path := url.Path
		url, _ = url.Parse(relTo.url.String())
		prefix := url.Path
		if url.Scheme == "" {
			prefix, err = filepath.Abs(relTo.url.String())
			if err != nil {
				return nil, fmt.Errorf("resource: could not detect abs path for %s; %s", relTo.url.String(), err.Error())
			}
		}
		url.Path = filepath.Dir(prefix) + "/" + path
	}

	var reader io.ReadCloser
	switch url.Scheme {
	case "":
		reader, err = os.Open(filepath.Clean(url.Path))
		if err != nil {
			return nil, err
		}
	case "http", "https":
		resp, err := http.Get(url.String())
		if err != nil {
			return nil, fmt.Errorf("resource: could not fetch '%s': %s", url.String(), err)
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("resource: could not fetch '%s': status %d", url.String(), resp.StatusCode)
		}
		reader = resp.Body
	default:
		return nil, fmt.Errorf("resource: unsupported scheme '%s'", url.Scheme)
	}

	return &resource{
		ReadCloser: reader,
		url:        url,
	}, nil
}

// Create a new texture from a resource.
func newTexture(res *resource) (*scene.ParsedTexture, error) {
	var pathToFile string

	// If this is a remote resource save it to a temp file so that oiio can load it
	if res.IsRemote() {
		pathToFile = os.TempDir() + "/" + filepath.Base(res.url.Path)
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
	var texFmt scene.TextureFormat
	var convertTo oiio.TypeDesc
	switch spec.Format() {
	case oiio.TypeUint8:
		convertTo = oiio.TypeUint8

		switch spec.NumChannels() {
		case 1:
			texFmt = scene.Luminance8
		default:
			texFmt = scene.Rgba8
		}
	default:
		convertTo = oiio.TypeFloat
		switch spec.NumChannels() {
		case 1:
			texFmt = scene.Luminance32F
		default:
			texFmt = scene.Rgba32F
		}
	}

	// Read data
	imgData, err := input.ReadImageFormat(convertTo, nil)
	if err != nil {
		return nil, fmt.Errorf("texture: could not read data from %s: %s", res.Path(), err.Error())
	}

	// Setup texture
	texture := &scene.ParsedTexture{
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
