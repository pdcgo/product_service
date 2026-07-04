package category

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/gorm"
)

// CategoryUpdate implements [product_ifaceconnect.CategoryServiceHandler].
//
// Renames a category and, when parent_id differs, reparents it: guards against
// cycles, recomputes the level of the node and its whole subtree, and maintains
// is_last on the old and new parents.
func (c *categorySrvImpl) CategoryUpdate(
	ctx context.Context,
	req *connect.Request[product_iface.CategoryUpdateRequest],
) (*connect.Response[product_iface.CategoryUpdateResponse], error) {
	pay := req.Msg
	if pay.GetId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if pay.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
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
		newParentID := uint(pay.GetParentId())

		// Rename only.
		if newParentID == oldParentID {
			return tx.
				Model(&product_models.Category{}).
				Where("id = ?", node.ID).
				Update("name", pay.GetName()).
				Error
		}

		// Reparent: load the full tree to compute the subtree + new levels.
		var all []product_models.Category
		aerr := tx.
			Select("id, parent_id, level").
			Find(&all).
			Error
		if aerr != nil {
			return aerr
		}

		byID := make(map[uint]product_models.Category, len(all))
		childrenOf := make(map[uint][]uint, len(all))
		for _, r := range all {
			byID[r.ID] = r
			var pid uint
			if r.ParentID != nil {
				pid = *r.ParentID
			}
			childrenOf[pid] = append(childrenOf[pid], r.ID)
		}

		// Gather node's subtree (including node).
		subtree := map[uint]bool{}
		var gather func(id uint)
		gather = func(id uint) {
			subtree[id] = true
			for _, ch := range childrenOf[id] {
				gather(ch)
			}
		}
		gather(node.ID)

		newLevel := 0
		if newParentID != 0 {
			if subtree[newParentID] {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("cannot move a category under itself or its descendant"))
			}
			parent, ok := byID[newParentID]
			if !ok {
				return connect.NewError(connect.CodeNotFound, errors.New("parent not found"))
			}
			newLevel = parent.Level + 1
		}
		delta := newLevel - node.Level

		var newParentVal any
		if newParentID != 0 {
			newParentVal = newParentID
		}
		uerr := tx.
			Model(&product_models.Category{}).
			Where("id = ?", node.ID).
			Updates(map[string]any{
				"name":      pay.GetName(),
				"parent_id": newParentVal,
				"level":     newLevel,
			}).
			Error
		if uerr != nil {
			return uerr
		}

		// Shift descendant levels by the same delta (subtree depth is preserved).
		if delta != 0 {
			descendants := make([]uint, 0, len(subtree))
			for id := range subtree {
				if id != node.ID {
					descendants = append(descendants, id)
				}
			}
			if len(descendants) > 0 {
				derr := tx.
					Model(&product_models.Category{}).
					Where("id IN ?", descendants).
					Update("level", gorm.Expr("level + ?", delta)).
					Error
				if derr != nil {
					return derr
				}
			}
		}

		// New parent is no longer a leaf.
		if newParentID != 0 {
			perr := tx.
				Model(&product_models.Category{}).
				Where("id = ?", newParentID).
				Update("is_last", false).
				Error
			if perr != nil {
				return perr
			}
		}

		// Old parent may become a leaf again.
		if oldParentID != 0 {
			oerr := recomputeIsLast(tx, oldParentID)
			if oerr != nil {
				return oerr
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&product_iface.CategoryUpdateResponse{}), nil
}
