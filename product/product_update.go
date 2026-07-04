package product

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ProductUpdate implements [product_ifaceconnect.ProductServiceHandler].
//
// Updates the editable fields of a product, scoped to its owning team so a team
// cannot edit another team's product. Returns NotFound when no such (id, team) row.
func (p *productSrvImpl) ProductUpdate(
	ctx context.Context,
	req *connect.Request[product_iface.ProductUpdateRequest],
) (*connect.Response[product_iface.ProductUpdateResponse], error) {
	pay := req.Msg
	if pay.GetId() == 0 || pay.GetTeamId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id and team_id are required"))
	}
	if pay.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Select the editable columns explicitly so zero values (cross_locked=false,
		// markup_percent=0) are written too; gorm quotes the reserved `desc` column.
		res := tx.
			Model(&product_models.Product{}).
			Where("id = ? AND team_id = ? AND deleted = false", pay.GetId(), pay.GetTeamId()).
			Select("name", "ref_id", "image", "desc", "markup_percent", "cross_locked").
			Updates(product_models.Product{
				Name:          pay.GetName(),
				RefID:         pay.GetRefId(),
				Image:         datatypes.NewJSONSlice(pay.GetImages()),
				Desc:          pay.GetDescription(),
				MarkupPercent: pay.GetMarkupPercent(),
				CrossLocked:   pay.GetCrossLocked(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return connect.NewError(connect.CodeNotFound, errors.New("product not found"))
		}

		// Replace the product's category association when a new one is given.
		if pay.GetCategoryId() > 0 {
			derr := tx.
				Exec("DELETE FROM product_category WHERE product_id = ?", pay.GetId()).
				Error
			if derr != nil {
				return derr
			}
			ierr := tx.
				Exec("INSERT INTO product_category (product_id, category_id) VALUES (?, ?)", pay.GetId(), pay.GetCategoryId()).
				Error
			if ierr != nil {
				return ierr
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&product_iface.ProductUpdateResponse{}), nil
}
