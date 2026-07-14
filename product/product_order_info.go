package product

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/product_iface/v1"
)

// ProductOrderInfo implements [product_ifaceconnect.ProductServiceHandler]. The
// dedicated order-create lookup for the selling v3 OrderService: one flat batched
// read of everything an order line needs — name snapshot, owning team (+ the `owned`
// check against the requesting team), and the cross-selling config
// (markup_percent, cross_locked). Missing ids are omitted from the map.
func (p *productSrvImpl) ProductOrderInfo(
	ctx context.Context,
	req *connect.Request[product_iface.ProductOrderInfoRequest],
) (*connect.Response[product_iface.ProductOrderInfoResponse], error) {
	pay := req.Msg
	if len(pay.GetProductIds()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("product_ids is required"))
	}
	if pay.GetTeamId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("team_id is required"))
	}

	db := p.db.WithContext(ctx)

	var rows []struct {
		ID            uint64
		Name          string
		TeamID        uint64
		MarkupPercent float64
		CrossLocked   bool
	}
	err := db.
		Table("products").
		Select("id, name, team_id, markup_percent, cross_locked").
		Where("id IN ?", pay.GetProductIds()).
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	result := &product_iface.ProductOrderInfoResponse{
		Items: map[uint64]*product_iface.ProductOrderInfoItem{},
	}
	for _, r := range rows {
		result.Items[r.ID] = &product_iface.ProductOrderInfoItem{
			Name:          r.Name,
			OwnerTeamId:   r.TeamID,
			Owned:         r.TeamID == pay.GetTeamId(),
			MarkupPercent: r.MarkupPercent,
			CrossLocked:   r.CrossLocked,
		}
	}

	return connect.NewResponse(result), nil
}
