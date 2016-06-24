package cmd

import (
	"strings"

	"github.com/achilleasa/go-pathtrace/scene/reader"
	"github.com/achilleasa/go-pathtrace/scene/writer"
	"github.com/urfave/cli"
)

// Compile scene to binary format.
func CompileScene(ctx *cli.Context) error {
	setupLogging(ctx)

	for idx := 0; idx < ctx.NArg(); idx++ {
		sceneFile := ctx.Args().Get(idx)
		if !strings.HasSuffix(sceneFile, ".obj") {
			logger.Warning("skipping unsupported file %s", sceneFile)
			continue
		}

		logger.Noticef("parsing and compiling scene: %s", sceneFile)
		sc, err := reader.ReadScene(sceneFile)
		if err != nil {
			return err
		}

		zipFile := strings.Replace(sceneFile, ".obj", ".zip", -1)
		err = writer.WriteScene(sc, zipFile)
		if err != nil {
			return err
		}
	}

	return nil
}
