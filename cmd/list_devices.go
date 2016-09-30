package cmd

import (
	"bytes"
	"fmt"

	"github.com/achilleasa/polaris/tracer/opencl/device"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

// List available opencl devices.
func ListDevices(ctx *cli.Context) error {
	setupLogging(ctx)

	var buf bytes.Buffer

	clPlatforms, err := device.GetPlatformInfo()
	if err != nil {
		return fmt.Errorf("could not list devices: %s", err.Error())
	}

	table := tablewriter.NewWriter(&buf)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"Device", "Type", "Estimated speed", "Vendor", "Version"})

	for _, platformInfo := range clPlatforms {
		for _, dev := range platformInfo.Devices {
			table.Append([]string{dev.Name, dev.Type.String(), fmt.Sprintf("%d GFlops", dev.Speed), platformInfo.Name, platformInfo.Version})
		}
	}
	table.Render()

	logger.Noticef("system provides %d opencl platform(s)\n%s", len(clPlatforms), buf.String())
	return nil
}
