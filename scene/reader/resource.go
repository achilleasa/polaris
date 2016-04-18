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
func newTexture(res *resource) (*texture, error) {
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
	var texFmt TextureFormat
	var convertTo oiio.TypeDesc
	switch spec.Format() {
	case oiio.TypeUint8:
		convertTo = oiio.TypeUint8

		switch spec.NumChannels() {
		case 1:
			texFmt = Luminance8
		case 3:
			texFmt = Rgb8
		case 4:
			texFmt = Rgba8
		}
	default:
		convertTo = oiio.TypeUint
		switch spec.NumChannels() {
		case 1:
			texFmt = Luminance32
		case 3:
			texFmt = Rgb32
		case 4:
			texFmt = Rgba32
		}
	}

	// Read data
	imgData, err := input.ReadImageFormat(convertTo, nil)
	if err != nil {
		return nil, fmt.Errorf("texture: could not read data from %s: %s", res.Path(), err.Error())
	}

	// Setup texture
	texture := &texture{
		format: texFmt,
		width:  uint32(spec.Width()),
		height: uint32(spec.Height()),
	}

	// Cast data to []byte
	switch t := imgData.(type) {
	case []uint8:
		texture.data = t
	case []uint:
		// Fetch slice header and adjust len/capacity (1 uint32 = 4 bytes)
		header := *(*reflect.SliceHeader)(unsafe.Pointer(&t))
		header.Len <<= 2
		header.Cap <<= 2

		// Convert to a []byte
		texture.data = *(*[]byte)(unsafe.Pointer(&header))
	}

	return texture, nil
}
