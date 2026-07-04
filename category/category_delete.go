package category

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/gorm"
)

// CategoryDelete implements [product_ifaceconnect.CategoryServiceHandler].
//
// Soft-deletes a category and its whole subtree (cascade) via gorm.DeletedAt, then
// recomputes the old parent's is_last. The product_category join rows are deliberately
// left intact: soft-deleted categories vanish from listings/pickers, but products that
// already reference them still resolve their category (ProductDetail reads the name via
// a raw join that ignores the soft-delete scope). See docs/test.md.
func (c *categorySrvImpl) CategoryDelete(
	ctx context.Context,
	req *connect.Request[product_iface.CategoryDeleteRequest],
) (*connect.Response[product_iface.CategoryDeleteResponse], error) {
	pay := req.Msg
	if pay.GetId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var node product_models.Category
		nerr := tx.
			First(&node, uint(pay.GetId())).
			Error
		if nerr != nil {
			if errors.Is(nerr, gorm.ErrRecordNotFound) {
				return connect.NewError(connect.CodeNotFound, errors.New("category not found"))
			}
			return nerr
		}

		var oldParentID uint
		if node.ParentID != nil {
			oldParentID = *node.ParentID
		}

		// Gather the subtree ids (node + all descendants).
		var all []product_models.Category
		aerr := tx.
			Select("id, parent_id").
			Find(&all).
			Error
		if aerr != nil {
			return aerr
		}
		childrenOf := make(map[uint][]uint, len(all))
		for _, r := range all {
			var pid uint
			if r.ParentID != nil {
				pid = *r.ParentID
			}
			childrenOf[pid] = append(childrenOf[pid], r.ID)
		}
		ids := []uint{}
		var gather func(id uint)
		gather = func(id uint) {
			ids = append(ids, id)
			for _, ch := range childrenOf[id] {
				gather(ch)
			}
		}
		gather(node.ID)

		// Soft-delete the subtree (sets deleted_at). The product_category joins are
		// left in place so already-inserted products keep resolving their category.
		derr := tx.
			Where("id IN ?", ids).
			Delete(&product_models.Category{}).
			Error
		if derr != nil {
			return derr
		}

		if oldParentID != 0 {
			return recomputeIsLast(tx, oldParentID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&product_iface.CategoryDeleteResponse{}), nil
}
