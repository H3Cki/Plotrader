package outboundcfg

import (
	"github.com/H3Cki/Plotrader/config"
	mongorepo "github.com/H3Cki/Plotrader/infractructure/repository/mongo"
	"github.com/H3Cki/Plotrader/infractructure/webhookpublisher"
)

type RepoConfig struct {
	DBName string
	URI    string
}

// func WithGORM(cfg RepoConfig) config.Option {
// 	return func(app *config.App) error {
// 		app.Repository = gormrepo.New(app.Logger, "plotrader.db")
// 		return nil
// 	}
// }

func WithMongo(cfg RepoConfig) config.Option {
	return func(app *config.App) error {
		app.Repository = mongorepo.New(mongorepo.Config{
			DBName: cfg.DBName,
			URI:    cfg.URI,
		})
		return nil
	}
}

func WithWebhookPublisher(app *config.App) error {
	app.Publisher = webhookpublisher.New(app.Logger)
	return nil
}
