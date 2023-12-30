package webhookpublisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/pkg/errors"
)

type Publisher struct {
	addr string
}

func New(addr string) *Publisher {
	return &Publisher{
		addr: addr,
	}
}

func (p *Publisher) PublishOrderUpdate(ctx context.Context, req outbound.OrderUpdate) error {
	msgBytes, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "error marshalling message")
	}

	res, err := http.Post(p.addr, "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}
