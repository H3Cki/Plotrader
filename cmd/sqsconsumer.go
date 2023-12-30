package cmd

import (
	"context"

	"github.com/H3Cki/Plotrader/config"
	"github.com/H3Cki/Plotrader/config/inboundcfg"
	"github.com/H3Cki/Plotrader/config/outboundcfg"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	// AWS
	awsRegionProp = "aws_region"

	// SNS
	sqsOrderQueueURL = "sqs_order_queue_url"
)

var SQSCommand = &cli.Command{
	Name:   "rest",
	Usage:  "start a http rest api",
	Action: runRESTCommand,
	Flags: []cli.Flag{
		// AWS Config
		&cli.StringFlag{Name: awsRegionProp, Usage: "region of aws", EnvVars: []string{"AWS_REGION"}},

		// SQS
		&cli.StringFlag{Name: sqsOrderQueueURL, Usage: "URL of a queue", EnvVars: []string{"SQS_ORDER_QUEUE_URL"}},
	},
}

func runSQSCommand(ctx *cli.Context) error {
	logger := ctx.App.Metadata["Logger"].(*zap.Logger)

	appConfig := config.AppConfig{
		AppName:    ctx.App.Name,
		AppVersion: ctx.App.Version,
		Env:        ctx.App.Metadata["Env"].(string),
	}

	awsCfg, err := buildAwsConfig(ctx.Context, ctx.String(awsRegionProp))
	if err != nil {
		return err
	}

	publisherCfg := outboundcfg.SNSPublisherConfig{
		SNSConfig: awsCfg,
		QueueURL:  ctx.String(sqsOrderQueueURL),
	}

	consumerCfg := inboundcfg.SQSConfig{
		SQSConfig: awsCfg,
		QueueURL:  ctx.String(sqsOrderQueueURL),
	}

	app, err := config.NewApp(appConfig,
		config.WithLogger(logger.Sugar()),
		outboundcfg.WithSNSPublisher(publisherCfg),
		inboundcfg.WithUpdaterService,
		inboundcfg.WithSQS(consumerCfg),
	)
	if err != nil {
		return errors.Wrap(err, "error creating app")
	}

	return app.SQSConsumer.Run(ctx.Context)
}

func endpointResolver(endpoint string) aws.EndpointResolverWithOptionsFunc {
	return func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if endpoint == "" {
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		}

		return aws.Endpoint{
			URL:           endpoint,
			PartitionID:   "aws",
			SigningRegion: region,
		}, nil
	}
}

func buildAwsConfig(ctx context.Context, region string) (awsConfig.Config, error) {
	return awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region))
}
