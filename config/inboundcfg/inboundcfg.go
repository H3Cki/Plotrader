package inboundcfg

import (
	"github.com/H3Cki/Plotrader/config"
	"github.com/H3Cki/Plotrader/core/application/updatersvc"
	"github.com/H3Cki/Plotrader/presentation/rest"
	"github.com/H3Cki/Plotrader/presentation/sqsconsumer"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
)

func WithUpdaterService(app *config.App) error {
	app.UpdaterService = updatersvc.New(updatersvc.Config{
		Logger:     app.Logger,
		Pubblisher: app.Publisher,
	})
	return nil
}

type SQSConfig struct {
	SQSConfig awsConfig.Config
	QueueURL  string
}

func WithSQS(cfg SQSConfig) func(app *config.App) error {
	return func(app *config.App) error {
		app.SQSConsumer = sqsconsumer.New(sqsconsumer.Config{
			Logger:     app.Logger,
			UpdaterSvc: app.UpdaterService,
			SQSConfig:  cfg.SQSConfig,
			QueueURL:   cfg.QueueURL,
		})
		return nil
	}
}

type RESTConfig struct {
	Addr string
}

func WithREST(cfg RESTConfig) func(app *config.App) error {
	return func(app *config.App) error {
		app.HTTPServer = rest.New(app.UpdaterService, cfg.Addr)
		return nil
	}
}
