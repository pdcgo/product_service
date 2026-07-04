package product

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/gorm"
)

// ProductDetail implements [product_ifaceconnect.ProductServiceHandler].
//
// Returns one product (scoped to its owning team) with all editable fields —
// backs the edit form. NotFound when no such live (id, team) row exists.
func (p *productSrvImpl) ProductDetail(
	ctx context.Context,
	req *connect.Request[product_iface.ProductDetailRequest],
) (*connect.Response[product_iface.ProductDetailResponse], error) {
	pay := req.Msg
	if pay.GetId() == 0 || pay.GetTeamId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id and team_id are required"))
	}

	db := p.db.WithContext(ctx)

	var prod product_models.Product
	err := db.
		Where("id = ? AND team_id = ? AND deleted = false", pay.GetId(), pay.GetTeamId()).
		First(&prod).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("product not found"))
		}
		return nil, err
	}

	var team struct{ Name string }
	err = db.
		Table("teams").
		Select("name").
		Where("id = ?", pay.GetTeamId()).
		Limit(1).
		Scan(&team).
		Error
	if err != nil {
		return nil, err
	}

	// The product's (single) category, via the product_category join.
	var cat struct {
		CategoryID uint64
		Name       string
	}
	err = db.
		Table("product_category pc").
		Joins("JOIN categories c ON c.id = pc.category_id").
		Select("pc.category_id as category_id, c.name as name").
		Where("pc.product_id = ?", pay.GetId()).
		Limit(1).
		Scan(&cat).
		Error
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&product_iface.ProductDetailResponse{
		Id:            uint64(prod.ID),
		TeamId:        uint64(prod.TeamID),
		TeamName:      team.Name,
		Name:          prod.Name,
		RefId:         prod.RefID,
		Images:        []string(prod.Image),
		Description:   prod.Desc,
		MarkupPercent: prod.MarkupPercent,
		CrossLocked:   prod.CrossLocked,
		CategoryId:    cat.CategoryID,
		CategoryName:  cat.Name,
	}), nil
}
