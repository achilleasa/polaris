package main

import (
	"fmt"
	"os"

	"github.com/achilleasa/polaris/cmd"
	"github.com/urfave/cli"
)

var (
	sceneCompileHelp = `
Parse a scene definition from a wavefront obj file, build a BVH tree to optimize
ray intersection tests and package scene assets in a GPU-friendly format.

The optimized scene data is then written to a zip archive which can be supplied
as an argument to the render commands.
`
)

func main() {
	cli.VersionFlag = cli.BoolFlag{
		Name:  "version",
		Usage: "print only the version",
	}

	app := cli.NewApp()
	app.Name = "polaris"
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
			Name: "scene",
			Subcommands: []cli.Command{
				{
					Name:        "compile",
					Usage:       "compile text scene representation into a binary compressed format",
					Description: sceneCompileHelp,
					ArgsUsage:   "scene_file1.obj scene_file2.obj ...",
					Action:      cmd.CompileScene,
				},
				{
					Name:      "info",
					Usage:     "print the size of the various compiled scene assets",
					ArgsUsage: "scene_file.zip",
					Action:    cmd.ShowSceneInfo,
				},
			},
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
					ArgsUsage:   "scene_file.zip or scene_file.obj",
					Flags: []cli.Flag{
						cli.IntFlag{
							Name:  "width",
							Value: 1024,
							Usage: "frame width",
						},
						cli.IntFlag{
							Name:  "height",
							Value: 1024,
							Usage: "frame height",
						},
						cli.IntFlag{
							Name:  "spp",
							Value: 16,
							Usage: "samples per pixel",
						},
						cli.IntFlag{
							Name:  "num-bounces, nb",
							Value: 5,
							Usage: "number of indirect ray bounces",
						},
						cli.IntFlag{
							Name:  "rr-bounces, nr",
							Value: 3,
							Usage: "number of indirect ray bounces before applying RR (disabled if 0 or >= than num-bounces)",
						},
						cli.Float64Flag{
							Name:  "exposure",
							Value: 1.2,
							Usage: "camera exposure for tone-mapping",
						},
						cli.StringSliceFlag{
							Name:  "blacklist, b",
							Value: &cli.StringSlice{},
							Usage: "blacklist opencl device whose names contain this value",
						},
						cli.StringFlag{
							Name:  "force-primary",
							Value: "",
							Usage: "force a particular device name as the primary device",
						},
						cli.StringFlag{
							Name:  "out, o",
							Value: "frame.png",
							Usage: "image filename for the rendered frame",
						},
					},
					Action: cmd.RenderFrame,
				},
				{
					Name:        "interactive",
					Usage:       "render interactive view of the scene",
					Description: ``,
					Flags: []cli.Flag{
						cli.IntFlag{
							Name:  "width",
							Value: 1024,
							Usage: "frame width",
						},
						cli.IntFlag{
							Name:  "height",
							Value: 1024,
							Usage: "frame height",
						},
						cli.IntFlag{
							Name:  "spp",
							Value: 0,
							Usage: "samples per pixel; setting to 0 enables progressive rendering",
						},
						cli.IntFlag{
							Name:  "num-bounces, nb",
							Value: 5,
							Usage: "number of indirect ray bounces",
						},
						cli.IntFlag{
							Name:  "rr-bounces, nr",
							Value: 3,
							Usage: "number of indirect ray bounces before applying RR (disabled if 0 or >= than num-bounces)",
						},
						cli.Float64Flag{
							Name:  "exposure",
							Value: 1.2,
							Usage: "camera exposure for tone-mapping",
						},
						cli.StringSliceFlag{
							Name:  "blacklist, b",
							Value: &cli.StringSlice{},
							Usage: "blacklist opencl device whose names contain this value",
						},
						cli.StringFlag{
							Name:  "force-primary",
							Value: "",
							Usage: "force a particular device name as the primary device",
						},
					},
					Action: cmd.RenderInteractive,
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
