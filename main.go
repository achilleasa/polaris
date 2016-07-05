package main

import (
	"os"

	"github.com/achilleasa/go-pathtrace/cmd"
	"github.com/urfave/cli"
)

func main() {
	cli.VersionFlag = cli.BoolFlag{
		Name:  "version",
		Usage: "print only the version",
	}

	app := cli.NewApp()
	app.Name = "go-pathtrace"
	app.Usage = "render scenes using path tracing"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "v",
			Usage: "enable verbose logging",
		},
		cli.BoolFlag{
			Name:  "vv",
			Usage: "enable even more verbose logging",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "compile",
			Usage: "compile text scene representation into a binary compressed format",
			Description: `
Parse a scene definition from a wavefront obj file, build a BVH tree to optimize
ray intersection tests and package scene elements in a GPU-friendly format.

The optimized scene data is then written to a zip archive which can be supplied
as an argument to the render command.`,
			ArgsUsage: "scene_file1.obj scene_file2.obj ...",
			Action:    cmd.CompileScene,
		},
		{
			Name:   "list-devices",
			Usage:  "list available opencl devices",
			Action: cmd.ListDevices,
		},
		{
			Name:   "render",
			Usage:  "render scene",
			Action: nil,
			Subcommands: []cli.Command{
				{
					Name:        "frame",
					Usage:       "render single frame",
					Description: `Render a single frame.`,
					Flags: []cli.Flag{
						cli.IntFlag{
							Name:  "width",
							Value: 512,
							Usage: "frame width",
						},
						cli.IntFlag{
							Name:  "height",
							Value: 512,
							Usage: "frame height",
						},
						cli.IntFlag{
							Name:  "spp",
							Value: 16,
							Usage: "samples per pixel",
						},
						cli.Float64Flag{
							Name:  "exposure",
							Value: 1.0,
							Usage: "camera exposure for tone-mapping",
						},
						cli.StringSliceFlag{
							Name:  "blacklist, b",
							Value: &cli.StringSlice{},
							Usage: "blacklist opencl device whose names contain this value",
						},
						cli.StringFlag{
							Name:  "out, o",
							Value: "frame.png",
							Usage: "image filename for the rendered frame",
						},
					},
					//	Action: cmd.RenderFrame,
				},
				{
					Name:        "interactive",
					Usage:       "render interactive view of the scene",
					Description: ``,
					Flags: []cli.Flag{
						cli.IntFlag{
							Name:  "width",
							Value: 512,
							Usage: "frame width",
						},
						cli.IntFlag{
							Name:  "height",
							Value: 512,
							Usage: "frame height",
						},
						cli.Float64Flag{
							Name:  "exposure",
							Value: 1.0,
							Usage: "camera exposure for tone-mapping",
						},
						cli.StringSliceFlag{
							Name:  "blacklist, b",
							Value: &cli.StringSlice{},
							Usage: "blacklist opencl device whose names contain this value",
						},
					},
					//	Action: cmd.RenderInteractive,
				},
			},
		},
		{
			Name:   "debug",
			Action: cmd.Debug,
		},
	}

	app.Run(os.Args)
}
