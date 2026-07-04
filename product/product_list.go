package product

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	common "github.com/pdcgo/schema/services/common/v1"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"github.com/pdcgo/shared/db_connect"
	"gorm.io/gorm"
)

type productListRow struct {
	Id            uint64
	Name          string
	RefId         string
	Image         string
	MarkupPercent float64
	CrossLocked   bool
	TeamId        uint64
	TeamName      string
}

// ProductList implements [product_ifaceconnect.ProductServiceHandler].
//
// Paginated list of a team's (non-deleted) products from the legacy `products`
// table, joined to `teams` for the owner name. `image` is the first array element.
func (p *productSrvImpl) ProductList(
	ctx context.Context,
	req *connect.Request[product_iface.ProductListRequest],
) (*connect.Response[product_iface.ProductListResponse], error) {
	pay := req.Msg
	if pay.GetTeamId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("team_id is required"))
	}
	if pay.GetPage() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("page is required"))
	}

	db := p.db.WithContext(ctx)

	result := &product_iface.ProductListResponse{
		Products: []*product_iface.ProductListItem{},
		PageInfo: &common.PageInfo{},
	}

	var rows []*productListRow
	paginated, pageInfo, err := db_connect.SetPaginationQuery(db, func() (*gorm.DB, error) {
		query := db.
			Table("products p").
			Joins("LEFT JOIN teams t ON t.id = p.team_id").
			Scopes(func(d *gorm.DB) *gorm.DB {
				d = d.Where("p.deleted = false")
				d = d.Where("p.team_id = ?", pay.GetTeamId())
				if pay.GetSearch() != "" {
					q := "%" + strings.ToLower(pay.GetSearch()) + "%"
					d = d.Where("lower(p.name) LIKE ?", q)
				}
				return d
			}).
			Select("p.id as id, p.name as name, p.ref_id as ref_id, p.image ->> 0 as image, p.markup_percent as markup_percent, p.cross_locked as cross_locked, p.team_id as team_id, t.name as team_name")
		return query, nil
	}, pay.GetPage())
	if err != nil {
		return nil, err
	}

	err = paginated.
		Order("p.id DESC").
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	result.PageInfo = pageInfo
	for _, row := range rows {
		if row == nil {
			continue
		}
		result.Products = append(result.Products, &product_iface.ProductListItem{
			Id:            row.Id,
			Name:          row.Name,
			RefId:         row.RefId,
			Image:         row.Image,
			MarkupPercent: row.MarkupPercent,
			CrossLocked:   row.CrossLocked,
			TeamId:        row.TeamId,
			TeamName:      row.TeamName,
		})
	}

	return connect.NewResponse(result), nil
}
