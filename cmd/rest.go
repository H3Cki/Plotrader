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
	addrProp   = "addr"
	dbNameProp = "db-name"
	dbURIProp  = "db-uri"
)

var RESTCommand = &cli.Command{
	Name:   "rest",
	Usage:  "start a http rest api",
	Action: runRESTCommand,
	Flags: []cli.Flag{
		&cli.StringFlag{Name: addrProp, Usage: "http rest server listen addr", EnvVars: []string{"ADDR"}, Value: "0.0.0.0:8080"},
		&cli.StringFlag{Name: dbNameProp, Usage: "name of the database", EnvVars: []string{"DB_NAME"}, Value: "plotrader.db"},
		&cli.StringFlag{Name: dbURIProp, Usage: "uri of the database", EnvVars: []string{"DB_URI"}, Value: "localhost"},
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

	repoCfg := outboundcfg.RepoConfig{
		DBName: ctx.String(dbNameProp),
		URI:    ctx.String(dbURIProp),
	}

	app, err := config.NewApp(appConfig,
		config.WithLogger(logger.Sugar()),
		outboundcfg.WithWebhookPublisher,
		outboundcfg.WithMongo(repoCfg),
		inboundcfg.WithUpdaterService,
		inboundcfg.WithREST(restCfg),
	)
	if err != nil {
		return errors.Wrap(err, "error creating app")
	}

	if err := app.Repository.Connect(ctx.Context); err != nil {
		return err
	}

	return app.HTTPServer.ListenAndServe()
}
