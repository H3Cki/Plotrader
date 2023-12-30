package main

import (
	"log"
	"os"

	"github.com/H3Cki/Plotrader/cmd"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	AppName    = "plotrader"
	AppVersion = "0.0.0"
)

func main() {
	app := &cli.App{
		Name:           AppName,
		Version:        AppVersion,
		Description:    "plotrader",
		Before:         before,
		DefaultCommand: cmd.RESTCommand.Name,
		Commands: []*cli.Command{
			cmd.RESTCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func before(ctx *cli.Context) error {
	var logger *zap.Logger

	envVar, ok := os.LookupEnv("ENV")
	if !ok {
		envVar = "DEV"
	}
	ctx.App.Metadata["Env"] = envVar

	if envVar == "PRD" {
		l, err := zap.NewProduction()
		if err != nil {
			return err
		}
		logger = l
	} else {
		l, err := zap.NewDevelopment()
		if err != nil {
			return err
		}
		logger = l
	}

	ctx.App.Metadata["Logger"] = logger
	return nil
}
