package config

import (
	"context"
	"net/http"

	"github.com/H3Cki/Plotrader/core/inbound"
	"github.com/H3Cki/Plotrader/core/outbound"
	"go.uber.org/zap"
)

type Option func(*App) error

type App struct {
	Config AppConfig
	Logger *zap.SugaredLogger

	// application
	FollowService inbound.FollowService

	// infrastructure
	Publisher  outbound.Publisher
	Repository outbound.Repository

	// presentation
	HTTPServer http.Server
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

func (a *App) Run(ctx context.Context) error {
	if err := a.Repository.Connect(ctx); err != nil {
		return err
	}
	return nil
}

func (a *App) Defer(ctx context.Context) error {

	if err := a.Repository.Connect(ctx); err != nil {
		return err
	}
	return nil
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
