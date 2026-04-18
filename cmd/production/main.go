package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pdcgo/product_service"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/pkg/cloud_logging"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gorm.io/gorm"
)

func NewDatabase(cfg *configs.AppConfig) (*gorm.DB, error) {
	return db_connect.NewProductionDatabase("product_service", &cfg.Database)
}

type App struct {
	Run func() error
}

func NewApp(
	mux *http.ServeMux,
	productRegister product_service.RegisterHandler,
	reflectorRegister custom_connect.RegisterReflectFunc,
	// cache ware_cache.Cache
	// auth authorization_iface.Authorization,
) *App {
	return &App{
		Run: func() error {
			cancel, err := custom_connect.InitTracer("product-service")
			if err != nil {
				return err
			}

			defer cancel(context.Background())

			productReflect := productRegister()
			reflectorRegister(productReflect)

			port := os.Getenv("PORT")
			if port == "" {
				port = "8087"
			}

			host := os.Getenv("HOST")
			listen := fmt.Sprintf("%s:%s", host, port)
			log.Println("listening on", listen)

			http.ListenAndServe(
				listen,
				// Use h2c so we can serve HTTP/2 without TLS.
				h2c.NewHandler(
					custom_connect.WithCORS(mux),
					&http2.Server{}),
			)

			return nil
		},
	}
}

func main() {
	if os.Getenv("DISABLE_CLOUD_LOGGING") == "" {
		cloud_logging.SetCloudLoggingDefault()
	}

	app, err := InitializeApp()
	if err != nil {
		panic(err)
	}

	err = app.Run()
	if err != nil {
		panic(err)
	}
}
