package sqsconsumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/H3Cki/Plotrader/core/inbound"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"go.uber.org/zap"
)

type Action string

const (
	ActionCreateOrder Action = "create"
	ActionCancelOrder Action = "cancel"
)

type orderRequestMessage struct {
	Action  Action `json:"action"`
	Request any    `json:"request"`
}

type Config struct {
	Logger     *zap.SugaredLogger
	UpdaterSvc inbound.UpdaterService
	SQSConfig  config.Config
	QueueURL   string
}

type Consumer struct {
	logger     *zap.SugaredLogger
	updaterSvc inbound.UpdaterService
	client     *sqs.Client
	queueURL   string
}

func New(cfg Config) *Consumer {
	return &Consumer{
		logger:     cfg.Logger,
		updaterSvc: cfg.UpdaterSvc,
		client:     sqs.NewFromConfig(cfg.SQSConfig.(aws.Config)),
		queueURL:   cfg.QueueURL,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:          &c.queueURL,
			VisibilityTimeout: 15,
		})
		if err != nil {
			return err
		}

		if len(msg.Messages) == 0 {
			continue
		}

		for _, message := range msg.Messages {
			go func(m types.Message) {
				if err := c.processMessage(ctx, m); err != nil {
					c.logger.Error(err.Error())
				}
			}(message)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, sqsMsg types.Message) error {
	reqMsg := orderRequestMessage{}

	if err := json.Unmarshal([]byte(*sqsMsg.Body), &reqMsg); err != nil {
		return err
	}

	switch reqMsg.Action {
	case ActionCreateOrder:
		req := inbound.CreateOrderRequest{}
		msgReqBytes, err := json.Marshal(reqMsg.Request)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(msgReqBytes, &req); err != nil {
			return err
		}
		if err := c.updaterSvc.CreateOrder(ctx, req); err != nil {
			return err
		}
		return nil
	case ActionCancelOrder:
		req := inbound.CancelOrderRequest{}
		msgReqBytes, err := json.Marshal(reqMsg.Request)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(msgReqBytes, &req); err != nil {
			return err
		}
		if err := c.updaterSvc.CancelOrder(ctx, req); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("unknown action: %s", reqMsg.Action)
}
