package reader

import (
	"archive/zip"
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/achilleasa/go-pathtrace/scene"
)

const (
	dataFile = "scene.bin"
)

type zipSceneReader struct {
	logger *log.Logger
}

// Create a new zip scene writer
func newZipSceneReader() *zipSceneReader {
	return &zipSceneReader{
		logger: log.New(os.Stdout, "zipSceneReader: ", log.LstdFlags),
	}
}

// Read scene definition from zip file.
func (p *zipSceneReader) Read(sceneRes *resource) (*scene.Scene, error) {
	p.logger.Printf("parsing compiled scene from %s", sceneRes.Path())
	start := time.Now()

	// zip package requires a reader implementing ReaderAt. To work around
	// this requirement we read the entire zip file into memory and create
	// a reader from the bytes package that implements ReaderAt
	data, err := ioutil.ReadAll(sceneRes)
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	sc := &scene.Scene{}
	for _, f := range zr.File {
		switch f.Name {
		case dataFile:
		default:
			p.logger.Printf("unknown file %s in scene zip file; skipping", f.Name)
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		decoder := gob.NewDecoder(rc)
		err = decoder.Decode(&sc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("zipSceneReader: failed to load %s: %s", f.Name, err.Error())
		}
	}

	p.logger.Printf("loaded scene in %d ms", time.Since(start).Nanoseconds()/1000000)
	return sc, nil
}
