package cmd

import (
	"log"
	"os"
)

var logger = log.New(os.Stdout, "go-pathtrace: ", log.LstdFlags)
