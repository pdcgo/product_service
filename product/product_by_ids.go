package product

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/gorm"
)

// ProductByIDs implements [product_ifaceconnect.ProductServiceHandler].
//
// Follows the "load Data By IDs" guideline (docs/proto-guideline.md): the caller
// picks the data types (GENERAL, TEAM) and gets a map keyed by product id, each id
// carrying one item per requested type in data_request order. It is an explicit-id
// lookup: soft-deleted products are still returned (by design — e.g. resolving
// products referenced by historical orders); team_id is an optional scope.
func (p *productSrvImpl) ProductByIDs(
	ctx context.Context,
	req *connect.Request[product_iface.ProductByIDsRequest],
) (*connect.Response[product_iface.ProductByIDsResponse], error) {
	payload := req.Msg
	if payload.GetFilter() == nil || len(payload.GetFilter().GetIds()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("filter.ids is required"))
	}
	if len(payload.GetDataRequest()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("data_request is required"))
	}

	db := p.db.WithContext(ctx)
	filter := payload.GetFilter()

	result := &product_iface.ProductByIDsResponse{
		Items: map[uint64]*product_iface.ProductByIdsItemList{},
	}
	appendItem := func(id uint64, item *product_iface.ProductByIdsItem) {
		list := result.Items[id]
		if list == nil {
			list = &product_iface.ProductByIdsItemList{}
			result.Items[id] = list
		}
		list.Items = append(list.Items, item)
	}

	for _, dt := range payload.GetDataRequest() {
		switch dt {
		case product_iface.ProductByIdsDataType_PRODUCT_BY_IDS_DATA_TYPE_GENERAL:
			var rows []struct {
				ID    uint64
				Name  string
				RefID string
				Image string
			}
			err := scopeByIds(db.Table("products"), filter).
				Select("id, name, ref_id, image ->> 0 as image").
				Scan(&rows).
				Error
			if err != nil {
				return nil, err
			}
			for _, r := range rows {
				appendItem(r.ID, &product_iface.ProductByIdsItem{
					D: &product_iface.ProductByIdsItem_General{General: &product_iface.ProductByIdsGeneralItem{
						Id:    r.ID,
						Name:  r.Name,
						RefId: r.RefID,
						Image: r.Image,
					}},
				})
			}

		case product_iface.ProductByIdsDataType_PRODUCT_BY_IDS_DATA_TYPE_TEAM:
			var rows []struct {
				ID       uint64
				TeamID   uint64
				TeamName string
			}
			err := scopeByIds(db.Table("products"), filter).
				Joins("LEFT JOIN teams t ON t.id = products.team_id").
				Select("products.id as id, products.team_id as team_id, coalesce(t.name, '') as team_name").
				Scan(&rows).
				Error
			if err != nil {
				return nil, err
			}
			for _, r := range rows {
				appendItem(r.ID, &product_iface.ProductByIdsItem{
					D: &product_iface.ProductByIdsItem_Team{Team: &product_iface.ProductByIdsTeamItem{
						TeamId:   r.TeamID,
						TeamName: r.TeamName,
					}},
				})
			}

		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid data type"))
		}
	}

	return connect.NewResponse(result), nil
}

// scopeByIds restricts a products query to the filter's ids (+ optional team scope).
// No soft-delete filter: an explicit-id lookup returns deleted products by design.
func scopeByIds(q *gorm.DB, filter *product_iface.ProductByIdsFilter) *gorm.DB {
	q = q.Where("products.id IN ?", filter.GetIds())
	if filter.GetTeamId() > 0 {
		q = q.Where("products.team_id = ?", filter.GetTeamId())
	}
	return q
}
