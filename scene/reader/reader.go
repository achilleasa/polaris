package reader

import (
	"fmt"
	"strings"

	"github.com/achilleasa/go-pathtrace/scene"
)

// The Reader interface is implemented by all scene readers.
type Reader interface {
	// Read scene definition from a resource.
	Read(*resource) (*scene.Scene, error)
}

// Read scene from file.
func ReadScene(filename string) (*scene.Scene, error) {
	res, err := newResource(filename, nil)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	// Select reader based on file extension
	var reader Reader
	if strings.HasSuffix(filename, ".obj") {
		reader = newWavefrontReader()
	} else if strings.HasSuffix(filename, ".zip") {
		reader = newZipSceneReader()
	} else {
		return nil, fmt.Errorf("readScene: unsupported file format")
	}
	return reader.Read(res)
}
