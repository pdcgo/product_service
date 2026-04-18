package product

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/gorm"
)

type productSrvImpl struct {
	db *gorm.DB
}

// ProductByIDs implements product_ifaceconnect.ProductServiceHandler.
func (p *productSrvImpl) ProductByIDs(context.Context, *connect.Request[product_iface.ProductByIDsRequest]) (*connect.Response[product_iface.ProductByIDsResponse], error) {
	panic("unimplemented")
}

// ProductMapConnect implements product_ifaceconnect.ProductServiceHandler.
func (p *productSrvImpl) ProductMapConnect(context.Context, *connect.Request[product_iface.ProductMapConnectRequest]) (*connect.Response[product_iface.ProductMapConnectResponse], error) {
	panic("unimplemented")
}

// ProductMapGet implements product_ifaceconnect.ProductServiceHandler.
func (p *productSrvImpl) ProductMapGet(context.Context, *connect.Request[product_iface.ProductMapGetRequest]) (*connect.Response[product_iface.ProductMapGetResponse], error) {
	panic("unimplemented")
}

func NewProductService(db *gorm.DB) *productSrvImpl {
	return &productSrvImpl{
		db: db,
	}
}
