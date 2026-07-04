package product_models

import "gorm.io/gorm"

// Category is product_service's local view of the legacy `categories` table — a
// GLOBAL (no team ownership) hierarchy. The tree is built in the handlers from the
// flat rows, so the recursive `Children` relation from the legacy struct is omitted
// here (see docs/database-schema.md).
//
// DeletedAt makes CategoryDelete a soft-delete: model-based queries auto-exclude
// deleted rows (so they vanish from listings/pickers), but the `product_category`
// joins are preserved so already-inserted products still resolve their category
// (see docs/test.md and CategoryDelete).
type Category struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	ParentID  *uint          `json:"parent_id" gorm:"index"`
	Level     int            `json:"level"`
	Name      string         `json:"name"`
	IsLast    bool           `json:"is_last"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (Category) TableName() string {
	return "categories"
}
