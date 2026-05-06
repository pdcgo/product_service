package product

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/product_iface/v1"
)

// ProductByIDs implements product_ifaceconnect.ProductServiceHandler.
func (p *productSrvImpl) ProductByIDs(
	ctx context.Context,
	req *connect.Request[product_iface.ProductByIDsRequest],
) (*connect.Response[product_iface.ProductByIDsResponse], error) {
	var err error

	db := p.db.WithContext(ctx)
	payload := req.Msg

	result := &product_iface.ProductByIDsResponse{
		Products: map[uint64]*product_iface.ProductIDsData{},
	}

	products := []*product_iface.ProductIDsData{}

	query := db.
		Table("products p").
		Where("p.id in ?", payload.Ids)

	err = query.
		Select([]string{
			"p.id",
			"p.name",
			"p.ref_id",
			"p.image ->>0 as image",
		}).
		Find(&products).
		Error

	if err != nil {
		return nil, err
	}

	for _, product := range products {
		result.Products[product.Id] = product
	}

	return connect.NewResponse(result), nil
}
