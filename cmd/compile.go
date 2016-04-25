package cmd

import (
	"strings"

	"github.com/achilleasa/go-pathtrace/scene/reader"
	"github.com/achilleasa/go-pathtrace/scene/writer"
	"github.com/codegangsta/cli"
)

// Compile scene to binary format.
func CompileScene(ctx *cli.Context) {
	for idx := 0; idx < ctx.NArg(); idx++ {
		sceneFile := ctx.Args().Get(idx)
		if !strings.HasSuffix(sceneFile, ".obj") {
			logger.Printf("skipping unsupported file %s", sceneFile)
			continue
		}

		sc, err := reader.ReadScene(sceneFile)
		if err != nil {
			logger.Printf("error: %s", err.Error())
			continue
		}

		zipFile := strings.Replace(sceneFile, ".obj", ".zip", -1)
		err = writer.WriteScene(sc, zipFile)
		if err != nil {
			logger.Printf("error: %s", err.Error())
			continue
		}
	}
}
