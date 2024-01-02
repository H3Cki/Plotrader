package cmd

import (
	"github.com/H3Cki/Plotrader/config"
	"github.com/H3Cki/Plotrader/config/inboundcfg"
	"github.com/H3Cki/Plotrader/config/outboundcfg"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	addrProp = "addr"
)

var RESTCommand = &cli.Command{
	Name:   "rest",
	Usage:  "start a http rest api",
	Action: runRESTCommand,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: addrProp, Usage: "http rest server listen addr", EnvVars: []string{"ADDR"}, Value: "0.0.0.0:8080"},
	},
}

func runRESTCommand(ctx *cli.Context) error {
	logger := ctx.App.Metadata["Logger"].(*zap.Logger)

	logger.Info("Starting REST server")

	appConfig := config.AppConfig{
		AppName:    ctx.App.Name,
		AppVersion: ctx.App.Version,
		Env:        ctx.App.Metadata["Env"].(string),
	}

	restCfg := inboundcfg.RESTConfig{
		Addr: ctx.String(addrProp),
	}

	app, err := config.NewApp(appConfig,
		config.WithLogger(logger.Sugar()),
		outboundcfg.WithWebhookPublisher,
		inboundcfg.WithUpdaterService,
		inboundcfg.WithREST(restCfg),
	)
	if err != nil {
		return errors.Wrap(err, "error creating app")
	}

	return app.HTTPServer.ListenAndServe()
}
