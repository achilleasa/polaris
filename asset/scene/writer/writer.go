package writer

import "github.com/achilleasa/polaris/asset/scene"

// The Writer interface is implemented by all scene writers.
type Writer interface {
	// Write scene definition
	Write(*scene.Scene) error
}

// Write scene to binary format.
func WriteScene(sc *scene.Scene, filename string) error {
	writer := newZipSceneWriter(filename)
	return writer.Write(sc)
}
