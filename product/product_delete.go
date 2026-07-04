package product

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
)

// ProductDelete implements [product_ifaceconnect.ProductServiceHandler].
//
// Soft-deletes a product (sets `deleted = true`), scoped to its owning team.
// Returns NotFound when no such live (id, team) row exists.
func (p *productSrvImpl) ProductDelete(
	ctx context.Context,
	req *connect.Request[product_iface.ProductDeleteRequest],
) (*connect.Response[product_iface.ProductDeleteResponse], error) {
	pay := req.Msg
	if pay.GetId() == 0 || pay.GetTeamId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id and team_id are required"))
	}

	res := p.db.
		WithContext(ctx).
		Model(&product_models.Product{}).
		Where("id = ? AND team_id = ? AND deleted = false", pay.GetId(), pay.GetTeamId()).
		Update("deleted", true)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("product not found"))
	}

	return connect.NewResponse(&product_iface.ProductDeleteResponse{}), nil
}
