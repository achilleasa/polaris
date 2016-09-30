package cmd

import (
	"github.com/achilleasa/polaris/log"
	"github.com/urfave/cli"
)

var logger = log.New("polaris")

func setupLogging(ctx *cli.Context) {
	if ctx.GlobalBool("v") {
		log.SetLevel(log.Info)
	}

	if ctx.GlobalBool("vv") {
		log.SetLevel(log.Debug)
	}
}
