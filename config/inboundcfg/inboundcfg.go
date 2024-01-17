package inboundcfg

import (
	"github.com/H3Cki/Plotrader/config"
	"github.com/H3Cki/Plotrader/core/application/followsvc"
	"github.com/H3Cki/Plotrader/presentation/rest"
)

func WithUpdaterService(app *config.App) error {
	app.FollowService = followsvc.New(followsvc.Config{
		Logger:     app.Logger,
		Publisher:  app.Publisher,
		Repository: app.Repository,
	})
	return nil
}

type RESTConfig struct {
	Addr string
}

func WithREST(cfg RESTConfig) config.Option {
	return func(app *config.App) error {
		app.HTTPServer = rest.New(app.FollowService, cfg.Addr)
		return nil
	}
}
