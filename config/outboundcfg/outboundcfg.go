package outboundcfg

import (
	"github.com/H3Cki/Plotrader/config"
	"github.com/H3Cki/Plotrader/infractructure/webhookpublisher"
)

func WithWebhookPublisher(app *config.App) error {
	app.Publisher = webhookpublisher.New(app.Logger)
	return nil

}
