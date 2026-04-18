//go:build wireinject
// +build wireinject

package main

import (
	"net/http"

	"github.com/google/wire"
	"github.com/pdcgo/product_service"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
)

func InitializeApp() (*App, error) {
	wire.Build(
		http.NewServeMux,
		configs.NewProductionConfig,
		NewDatabase,
		custom_connect.NewDefaultInterceptor,
		custom_connect.NewRegisterReflect,
		product_service.NewRegister,
		NewApp,
	)

	return &App{}, nil
}
