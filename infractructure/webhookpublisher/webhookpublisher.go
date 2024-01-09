package webhookpublisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Publisher struct {
	logger *zap.SugaredLogger
}

func New(logger *zap.SugaredLogger) *Publisher {
	return &Publisher{logger: logger}
}

func (p *Publisher) PublishFollowUpdate(ctx context.Context, update outbound.FollowUpdate) error {
	if update.WebhookURL == "" {
		//p.logger.Debug("webhook URL not specified for update %+v", update)
		return nil
	}

	msgBytes, err := json.Marshal(update)
	if err != nil {
		return errors.Wrap(err, "error marshalling message")
	}

	res, err := http.Post(update.WebhookURL, "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}
