package io

import (
	"fmt"
	"strings"

	"github.com/achilleasa/go-pathtrace/scene"
)

// The Reader interface is implemented by all scene readers.
type Reader interface {
	// Read scene definition.
	Read() (*scene.Scene, error)
}

// Read scene from file.
func ReadScene(filename string) (*scene.Scene, error) {
	var reader Reader
	if strings.HasSuffix(filename, ".obj") {
		reader = newTextSceneReader(filename)
	} else if strings.HasSuffix(filename, ".zip") {
		reader = newZipSceneReader(filename)
	} else {
		return nil, fmt.Errorf("readScene: unsupported file format")
	}
	return reader.Read()
}
