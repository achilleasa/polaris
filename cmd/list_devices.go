package cmd

import (
	"bytes"
	"fmt"

	"github.com/achilleasa/go-pathtrace/tracer/opencl"
	"github.com/codegangsta/cli"
)

// List available opencl devices.
func ListDevices(ctx *cli.Context) {
	var storage []byte
	buf := bytes.NewBuffer(storage)

	clPlatforms := opencl.GetPlatformInfo()
	buf.WriteString(fmt.Sprintf("\nSystem provides %d opencl platform(s):\n\n", len(clPlatforms)))
	for pIdx, platformInfo := range clPlatforms {
		buf.WriteString(fmt.Sprintf("[Platform %02d]\n  Name    %s\n  Version %s\n  Profile %s\n  Devices %d\n\n", pIdx, platformInfo.Name, platformInfo.Version, platformInfo.Profile, len(platformInfo.Devices)))
		for dIdx, device := range platformInfo.Devices {
			buf.WriteString(fmt.Sprintf("  [Device %02d]\n    Name  %s\n    Type  %s\n    Speed %3.1f\n\n", dIdx, device.Name, device.Type, device.SpeedEstimate()))
		}
	}

	logger.Print(buf.String())
}
