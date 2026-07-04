package product_service

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/category"
	"github.com/pdcgo/product_service/product"
	"github.com/pdcgo/san_collection/san_caches"
	"github.com/pdcgo/schema/services/product_iface/v1/product_ifaceconnect"
	"github.com/pdcgo/shared/configs"
	"github.com/pdcgo/shared/custom_connect"
	"github.com/pdcgo/user_service/access_interceptors"
	"gorm.io/gorm"
)

type ServiceReflectNames []string
type RegisterHandler func() ServiceReflectNames

func NewRegister(
	mux *http.ServeMux,
	db *gorm.DB,
	cfg *configs.AppConfig,
	cacheMgr san_caches.CacheManager,
	defaultInterceptor custom_connect.DefaultInterceptor,
) RegisterHandler {
	return func() ServiceReflectNames {
		grpcReflects := ServiceReflectNames{}

		// v2 roling: enforce the (role_base.v1.request_policy) declared on each
		// request message (authenticated + team-scoped for the CRUD RPCs).
		roleOpt := connect.WithInterceptors(access_interceptors.NewAccessInterceptor(db, cfg.JwtSecret, cacheMgr))

		path, handler := product_ifaceconnect.NewProductServiceHandler(product.NewProductService(db), defaultInterceptor, roleOpt)
		mux.Handle(path, handler)
		grpcReflects = append(grpcReflects, product_ifaceconnect.ProductServiceName)

		catPath, catHandler := product_ifaceconnect.NewCategoryServiceHandler(category.NewCategoryService(db), defaultInterceptor, roleOpt)
		mux.Handle(catPath, catHandler)
		grpcReflects = append(grpcReflects, product_ifaceconnect.CategoryServiceName)

		return grpcReflects
	}
}
