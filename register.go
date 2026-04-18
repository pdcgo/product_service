package product_service

import (
	"net/http"

	"github.com/pdcgo/product_service/product"
	"github.com/pdcgo/schema/services/product_iface/v1/product_ifaceconnect"
	"github.com/pdcgo/shared/custom_connect"
	"gorm.io/gorm"
)

type ServiceReflectNames []string
type RegisterHandler func() ServiceReflectNames

func NewRegister(
	mux *http.ServeMux,
	db *gorm.DB,
	defaultInterceptor custom_connect.DefaultInterceptor,
) RegisterHandler {
	return func() ServiceReflectNames {
		grpcReflects := ServiceReflectNames{}
		path, handler := product_ifaceconnect.NewProductServiceHandler(product.NewProductService(db), defaultInterceptor)
		mux.Handle(path, handler)
		grpcReflects = append(grpcReflects, product_ifaceconnect.ProductServiceName)

		return grpcReflects
	}
}
