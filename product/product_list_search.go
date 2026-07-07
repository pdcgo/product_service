package product

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/product_iface/v1"
)

// ProductListSearch implements [product_ifaceconnect.ProductServiceHandler].
//
// The "Product List for Fastest Ops" RPC backing picker components: a lean,
// team-scoped search where one q matches the product name OR the product code
// (ref_id). Excludes soft-deleted products; newest first. Limit defaults to 25
// and is capped at 100 (a zero limit never returns zero rows).
func (p *productSrvImpl) ProductListSearch(
	ctx context.Context,
	req *connect.Request[product_iface.ProductListSearchRequest],
) (*connect.Response[product_iface.ProductListSearchResponse], error) {
	payload := req.Msg
	if payload.GetTeamId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("team_id is required"))
	}

	limit := payload.GetLimit()
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	db := p.db.WithContext(ctx)

	query := db.
		Table("products").
		Where("team_id = ? AND deleted = false", payload.GetTeamId())
	if q := payload.GetQ(); q != "" {
		like := "%" + q + "%"
		query = query.Where("(name ILIKE ? OR ref_id ILIKE ?)", like, like)
	}

	res := &product_iface.ProductListSearchResponse{
		Products: []*product_iface.ProductListSearchItem{},
	}
	err := query.
		Select("id, name, ref_id, image ->> 0 as image").
		Order("id DESC").
		Limit(int(limit)).
		Find(&res.Products).
		Error
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(res), nil
}
