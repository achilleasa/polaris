package io

import (
	"archive/zip"
	"encoding/gob"
	"fmt"
	"log"
	"os"

	"github.com/achilleasa/go-pathtrace/scene"
)

const (
	bvhFile         = "bvhData.bin"
	primFile        = "primitives.bin"
	matFile         = "materials.bin"
	matNamesFile    = "meterialNames.bin"
	emissiveIndFile = "emissiveIndices.bin"
	cameraFile      = "camera.bin"
)

type zipSceneWriter struct {
	sceneFile string
}

// Create a new zip scene writer
func newZipSceneWriter(sceneFile string) *zipSceneWriter {
	return &zipSceneWriter{
		sceneFile: sceneFile,
	}
}

// Write scene definition to zip file.
func (w *zipSceneWriter) Write(sc *scene.Scene) error {
	var err error

	zipFile, err := os.Create(w.sceneFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// Create zip writer
	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	// Write bvh data
	cw, err := zw.Create(bvhFile)
	encoder := gob.NewEncoder(cw)
	err = encoder.Encode(sc.BvhNodes)
	if err != nil {
		return err
	}

	// Write primitive data
	cw, err = zw.Create(primFile)
	encoder = gob.NewEncoder(cw)
	err = encoder.Encode(sc.Primitives)
	if err != nil {
		return err
	}

	// Write material data
	cw, err = zw.Create(matFile)
	encoder = gob.NewEncoder(cw)
	err = encoder.Encode(sc.Materials)
	if err != nil {
		return err
	}

	// Write material name to index data
	cw, err = zw.Create(matNamesFile)
	encoder = gob.NewEncoder(cw)
	err = encoder.Encode(sc.MatNameToIndex)
	if err != nil {
		return err
	}
	// Write emissive primitive indices
	cw, err = zw.Create(emissiveIndFile)
	encoder = gob.NewEncoder(cw)
	err = encoder.Encode(sc.EmissivePrimitiveIndices)
	if err != nil {
		return err
	}

	// Write camera data
	cw, err = zw.Create(cameraFile)
	encoder = gob.NewEncoder(cw)
	err = encoder.Encode(sc.Camera)
	if err != nil {
		return err
	}

	return nil
}

type zipSceneReader struct {
	logger    *log.Logger
	sceneFile string
}

// Create a new zip scene writer
func newZipSceneReader(sceneFile string) *zipSceneReader {
	return &zipSceneReader{
		logger:    log.New(os.Stderr, "zipSceneReader: ", log.LstdFlags),
		sceneFile: sceneFile,
	}
}

// Read scene definition from zip file.
func (p *zipSceneReader) Read() (*scene.Scene, error) {
	var err error
	zr, err := zip.OpenReader(p.sceneFile)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	sc := &scene.Scene{}
	var target interface{}
	for _, f := range zr.File {
		switch f.Name {
		case bvhFile:
			target = &sc.BvhNodes
		case primFile:
			target = &sc.Primitives
		case matFile:
			target = &sc.Materials
		case matNamesFile:
			target = &sc.MatNameToIndex
		case emissiveIndFile:
			target = &sc.EmissivePrimitiveIndices
		case cameraFile:
			target = &sc.Camera
		default:
			p.logger.Printf("unknown file %s in scene zip file; skipping", f.Name)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		decoder := gob.NewDecoder(rc)
		err = decoder.Decode(target)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("zipSceneReader: failed to load %s: %s", f.Name, err.Error())
		}
	}

	return sc, nil
}
