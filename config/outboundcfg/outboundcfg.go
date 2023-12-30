package outboundcfg

import (
	"github.com/H3Cki/Plotrader/config"
	"github.com/H3Cki/Plotrader/infractructure/snspublisher"
	"github.com/H3Cki/Plotrader/infractructure/webhookpublisher"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/go-playground/validator/v10"
)

type SNSPublisherConfig struct {
	SNSConfig awsCfg.Config `validate:"required"`
	QueueURL  string        `validate:"required"`
}

func WithSNSPublisher(cfg SNSPublisherConfig) func(app *config.App) error {
	return func(app *config.App) error {
		if err := validator.New().Struct(cfg); err != nil {
			return err
		}

		app.Publisher = snspublisher.New(cfg.SNSConfig, cfg.QueueURL)
		return nil
	}
}

type WebhookPublisherConfig struct {
	Addr string `validate:"required"`
}

func WithWebhookPublisher(cfg WebhookPublisherConfig) func(app *config.App) error {
	return func(app *config.App) error {
		if err := validator.New().Struct(cfg); err != nil {
			return err
		}

		app.Publisher = webhookpublisher.New(cfg.Addr)
		return nil
	}
}
