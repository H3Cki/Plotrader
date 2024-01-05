package inboundcfg

import (
	"github.com/H3Cki/Plotrader/config"
	"github.com/H3Cki/Plotrader/core/application/followsvc"
	"github.com/H3Cki/Plotrader/presentation/rest"
)

func WithUpdaterService(app *config.App) error {
	app.UpdaterService = followsvc.New(followsvc.Config{
		Logger:     app.Logger,
		Pubblisher: app.Publisher,
	})
	return nil
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
