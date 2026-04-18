package product

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/product_iface/v1"
)

// ProductDuplicate implements product_ifaceconnect.ProductServiceHandler.
func (p *productSrvImpl) ProductDuplicate(
	ctx context.Context,
	req *connect.Request[product_iface.ProductDuplicateRequest]) (*connect.Response[product_iface.ProductDuplicateResponse], error) {
	// var err error

	// pay := req.Msg
	// db := p.db.WithContext(ctx)
	// err = db.Transaction(func(tx *gorm.DB) error {
	// 	// var old db_models.VariationValue
	// 	// tx.First(&old, pay.)

	// 	return nil
	// })
	return nil, nil
}
