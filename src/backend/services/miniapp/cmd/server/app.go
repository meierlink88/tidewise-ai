package main

import (
	"context"

	kratos "github.com/go-kratos/kratos/v3"

	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/biz"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/conf"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/data"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/server"
	"github.com/meierlink88/tidewise-ai/backend/services/miniapp/internal/service"
)

func buildApp(config conf.RuntimeConfig) (*kratos.App, error) {
	repository, err := data.NewHTTPClient(data.HTTPConfig{
		BaseURL:      config.DataService.BaseURL,
		ServiceToken: config.DataService.IdentityToken,
		Timeout:      config.DataService.Timeout,
	})
	if err != nil {
		return nil, err
	}
	useCase := biz.NewResearchService(repository)
	applicationService := service.NewResearchService(useCase)
	httpServer := server.NewHTTPServer(config, applicationService)

	return kratos.New(
		kratos.Name(conf.ServiceName),
		kratos.Server(httpServer),
		kratos.AfterStop(func(context.Context) error {
			return repository.Close()
		}),
	), nil
}
