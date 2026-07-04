package main

import (
	"context"
	"os"

	"github.com/pdcgo/san_collection/san_caches"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/db_connect"
	"github.com/pdcgo/shared/pkg/cloud_logging"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
)

func NewDatabase(cfg *configs.AppConfig) (*gorm.DB, error) {
	return db_connect.NewProductionDatabase("product_service", &cfg.Database)
}

// NewCacheManager backs the v2 access interceptor's role cache. Skip-cache (no Redis)
// mirrors the development omnibus; every role lookup hits the DB.
func NewCacheManager() san_caches.CacheManager {
	return san_caches.NewSkipCacheManager()
}

func NewApp(

	serviceApiFunc ServiceApiFunc,
	// reindexFunc ReindexFunc,
	// cache ware_cache.Cache
	// auth authorization_iface.Authorization,
) *cli.Command {

	return &cli.Command{
		Name:     "product-service",
		Action:   cli.ActionFunc(serviceApiFunc),
		Commands: []*cli.Command{
			// {
			// 	Name:   "reindex",
			// 	Action: cli.ActionFunc(reindexFunc),
			// },
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

	err = app.Run(context.Background(), os.Args)
	if err != nil {
		panic(err)
	}
}
