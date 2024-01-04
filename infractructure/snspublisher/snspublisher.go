package snspublisher

import (
	"context"
	"encoding/json"

	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pkg/errors"
)

type Publisher struct {
	snsClient *sns.Client
	QueueURL  string
}

func New(cfg awsCfg.Config, queueURL string) *Publisher {
	return &Publisher{
		snsClient: sns.NewFromConfig(cfg.(aws.Config)),
		QueueURL:  queueURL,
	}
}

func (p *Publisher) PublishFollowUpdate(ctx context.Context, req outbound.FollowUpdate) error {
	msgBytes, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "error marshalling message")
	}

	msgString := string(msgBytes)
	_, err = p.snsClient.Publish(ctx, &sns.PublishInput{
		Message:   &msgString,
		TargetArn: &msgString,
	})
	return err
}
