package category

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/gorm"
)

// CategoryCreate implements [product_ifaceconnect.CategoryServiceHandler].
//
// Creates a category. When parent_id is set, the level is derived from the parent
// (parent.level+1) and the parent stops being a leaf (is_last=false).
func (c *categorySrvImpl) CategoryCreate(
	ctx context.Context,
	req *connect.Request[product_iface.CategoryCreateRequest],
) (*connect.Response[product_iface.CategoryCreateResponse], error) {
	pay := req.Msg
	if pay.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	var newID uint
	err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		level := 0
		var parentPtr *uint

		if pay.GetParentId() > 0 {
			var parent product_models.Category
			perr := tx.
				First(&parent, uint(pay.GetParentId())).
				Error
			if perr != nil {
				if errors.Is(perr, gorm.ErrRecordNotFound) {
					return connect.NewError(connect.CodeNotFound, errors.New("parent not found"))
				}
				return perr
			}

			level = parent.Level + 1
			pid := parent.ID
			parentPtr = &pid

			perr = tx.
				Model(&product_models.Category{}).
				Where("id = ?", parent.ID).
				Update("is_last", false).
				Error
			if perr != nil {
				return perr
			}
		}

		cat := product_models.Category{
			Name:     pay.GetName(),
			ParentID: parentPtr,
			Level:    level,
			IsLast:   true,
		}
		cerr := tx.Create(&cat).Error
		if cerr != nil {
			return cerr
		}
		newID = cat.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&product_iface.CategoryCreateResponse{Id: uint64(newID)}), nil
}
