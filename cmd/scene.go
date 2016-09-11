package cmd

import (
	"errors"
	"strings"

	"github.com/achilleasa/go-pathtrace/asset/scene/reader"
	"github.com/achilleasa/go-pathtrace/asset/scene/writer"
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

		// Display compiled scene info
		logger.Noticef("scene information:\n%s", sc.Stats())

		zipFile := strings.Replace(sceneFile, ".obj", ".zip", -1)
		err = writer.WriteScene(sc, zipFile)
		if err != nil {
			return err
		}
	}

	return nil
}

// Display compiled scene info.
func ShowSceneInfo(ctx *cli.Context) error {
	setupLogging(ctx)

	if ctx.NArg() != 1 {
		return errors.New("missing compiled scene zip file")
	}

	sceneFile := ctx.Args().First()
	if !strings.HasSuffix(sceneFile, ".zip") {
		return errors.New("only compiled scene files with a .zip extension are supported")
	}

	sc, err := reader.ReadScene(sceneFile)
	if err != nil {
		return err
	}

	// Display compiled scene info
	logger.Noticef("scene information:\n%s", sc.Stats())

	return nil
}
