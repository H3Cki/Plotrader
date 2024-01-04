package config

import (
	"net/http"

	"github.com/H3Cki/Plotrader/core/inbound"
	"github.com/H3Cki/Plotrader/core/outbound"
	"github.com/H3Cki/Plotrader/presentation/sqsconsumer"
	"go.uber.org/zap"
)

type App struct {
	Config AppConfig
	Logger *zap.SugaredLogger

	// application
	UpdaterService inbound.FollowerService

	// infrastructure
	Publisher outbound.Publisher

	// presentation
	SQSConsumer *sqsconsumer.Consumer
	HTTPServer  http.Server
}

func NewApp(cfg AppConfig, opts ...func(*App) error) (*App, error) {
	app := &App{Config: cfg}

	for _, opt := range opts {
		if err := opt(app); err != nil {
			return nil, err
		}
	}

	return app, nil
}

func WithLogger(logger *zap.SugaredLogger) func(*App) error {
	return func(a *App) error {
		a.Logger = logger
		return nil
	}
}

type AppConfig struct {
	AppName    string
	AppVersion string
	Env        string
}
