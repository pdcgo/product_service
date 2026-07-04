package category

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
)

// CategoryList implements [product_ifaceconnect.CategoryServiceHandler].
//
// Cross-team master data: requires a valid authenticated identity but does NOT
// filter by team. Returns the category tree nested (roots, or the parent_id node's
// children). When `search` is set it returns a flat list of matching nodes instead.
func (c *categorySrvImpl) CategoryList(
	ctx context.Context,
	req *connect.Request[product_iface.CategoryListRequest],
) (*connect.Response[product_iface.CategoryListResponse], error) {
	pay := req.Msg
	db := c.db.WithContext(ctx)

	// Search: flat list of matches (convenient for a picker type-ahead).
	if pay.GetSearch() != "" {
		var rows []product_models.Category
		err := db.
			Model(&product_models.Category{}).
			Where("name ILIKE ?", "%"+pay.GetSearch()+"%").
			Order("level asc, name asc, id asc").
			Find(&rows).
			Error
		if err != nil {
			return nil, err
		}
		out := make([]*product_iface.Category, 0, len(rows))
		for i := range rows {
			out = append(out, toProtoCategory(rows[i]))
		}
		return connect.NewResponse(&product_iface.CategoryListResponse{Data: out}), nil
	}

	// Otherwise build the nested tree from all rows.
	var rows []product_models.Category
	err := db.
		Model(&product_models.Category{}).
		Order("level asc, id asc").
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}

	byID := make(map[uint]*product_iface.Category, len(rows))
	for i := range rows {
		byID[rows[i].ID] = toProtoCategory(rows[i])
	}
	roots := []*product_iface.Category{}
	for i := range rows {
		r := rows[i]
		node := byID[r.ID]
		if r.ParentID != nil {
			parent, ok := byID[*r.ParentID]
			if ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	data := roots
	if pay.GetParentId() > 0 {
		parent, ok := byID[uint(pay.GetParentId())]
		if ok {
			data = parent.Children
		} else {
			data = []*product_iface.Category{}
		}
	}

	return connect.NewResponse(&product_iface.CategoryListResponse{Data: data}), nil
}
