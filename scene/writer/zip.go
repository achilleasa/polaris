package writer

import (
	"archive/zip"
	"encoding/gob"
	"log"
	"os"
	"time"

	"github.com/achilleasa/go-pathtrace/scene"
)

const (
	dataFile = "scene.bin"
)

type zipSceneWriter struct {
	logger    *log.Logger
	sceneFile string
}

// Create a new zip scene writer
func newZipSceneWriter(sceneFile string) *zipSceneWriter {
	return &zipSceneWriter{
		logger:    log.New(os.Stdout, "zipSceneWriter: ", log.LstdFlags),
		sceneFile: sceneFile,
	}
}

// Write scene definition to zip file.
func (w *zipSceneWriter) Write(sc *scene.Scene) error {
	w.logger.Printf("writing compressed scene to %s", w.sceneFile)
	start := time.Now()

	var err error
	zipFile, err := os.Create(w.sceneFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// Create zip writer
	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	// Write scene data
	cw, err := zw.Create(dataFile)
	encoder := gob.NewEncoder(cw)
	err = encoder.Encode(sc)
	if err != nil {
		return err
	}

	w.logger.Printf("compressed scene in %d ms", time.Since(start).Nanoseconds()/1000000)
	return nil
}
