package category

import (
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/gorm"
)

// categorySrvImpl serves the CategoryService — GLOBAL (cross-team) hierarchical
// master data on the legacy `categories` table. Authentication is enforced by the
// v2 access interceptor (allow_only_authenticated on each request message); the
// handlers do no per-team/role scoping.
type categorySrvImpl struct {
	db *gorm.DB
}

func NewCategoryService(db *gorm.DB) *categorySrvImpl {
	return &categorySrvImpl{db: db}
}

// recomputeIsLast sets is_last on a category from whether it currently has any children.
func recomputeIsLast(tx *gorm.DB, id uint) error {
	var cnt int64
	err := tx.
		Model(&product_models.Category{}).
		Where("parent_id = ?", id).
		Count(&cnt).
		Error
	if err != nil {
		return err
	}
	return tx.
		Model(&product_models.Category{}).
		Where("id = ?", id).
		Update("is_last", cnt == 0).
		Error
}

func toProtoCategory(r product_models.Category) *product_iface.Category {
	var pid uint64
	if r.ParentID != nil {
		pid = uint64(*r.ParentID)
	}
	return &product_iface.Category{
		Id:       uint64(r.ID),
		Name:     r.Name,
		ParentId: pid,
		Level:    int32(r.Level),
		IsLast:   r.IsLast,
	}
}
