package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pdcgo/product_service"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/urfave/cli/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type ServiceApiFunc cli.ActionFunc

func NewServiceApiFunc(
	mux *http.ServeMux,
	productRegister product_service.RegisterHandler,
	reflectorRegister custom_connect.RegisterReflectFunc,
) ServiceApiFunc {
	return func(ctx context.Context, c *cli.Command) error {

		cancel, err := custom_connect.InitTracer("product-service")
		if err != nil {
			return err
		}

		defer cancel(ctx)

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

	}
}
