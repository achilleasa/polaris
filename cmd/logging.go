package cmd

import (
	"github.com/achilleasa/go-pathtrace/log"
	"github.com/urfave/cli"
)

var logger = log.New("go-pathtrace")

func setupLogging(ctx *cli.Context) {
	if ctx.GlobalBool("v") {
		log.SetLevel(log.Info)
	}

	if ctx.GlobalBool("vv") {
		log.SetLevel(log.Debug)
	}
}
