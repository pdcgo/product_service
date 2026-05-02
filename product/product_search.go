package product

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/product_iface/v1"
)

// ProductSearch implements [product_ifaceconnect.ProductServiceHandler].
func (p *productSrvImpl) ProductSearch(
	ctx context.Context,
	req *connect.Request[product_iface.ProductSearchRequest],
) (*connect.Response[product_iface.ProductSearchResponse], error) {
	var err error

	db := p.db.WithContext(ctx)

	payload := req.Msg
	res := product_iface.ProductSearchResponse{
		Products: []*product_iface.ProductSearchItem{},
	}

	query := db.
		Table("products p").
		Where("p.team_id = ?", payload.TeamId).
		Where("p.deleted != true")

	switch search := payload.Search.(type) {
	case *product_iface.ProductSearchRequest_Name:
		q := strings.ToLower(search.Name)
		query = query.Where("p.name ilike ?", "%"+q+"%")
	case *product_iface.ProductSearchRequest_RefId:
		q := strings.ToLower(search.RefId)
		query = query.Where("p.ref_id ilike ?", "%"+q+"%")
	}

	query = query.
		Limit(int(payload.Limit)).
		Select([]string{
			"p.id",
			"p.name",
			"p.ref_id",
			"p.image ->> 0 as image",
		})

	err = query.
		Find(&res.Products).
		Error
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&res), nil
}
