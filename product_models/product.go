package product_models

import (
	"time"

	"gorm.io/datatypes"
)

// Product is product_service's local view of the legacy `products` table (this
// service is taking over the legacy product system — see docs/database-schema.md).
//
// It intentionally OMITS the legacy variation/stock-count/category/tag fields and
// the Team/User relation structs: those belong to other domains now. team_id /
// user_id are kept as plain columns (the migration keeps them as FKs).
type Product struct {
	ID      uint   `json:"id" gorm:"primarykey"`
	TeamID  uint   `json:"team_id" gorm:"index"`
	Deleted bool   `json:"deleted" gorm:"index"`
	UserID  uint   `json:"user_id"`
	RefID   string `json:"ref_id"`

	Name  string                      `json:"name"`
	Image datatypes.JSONSlice[string] `json:"image"`
	Desc  string                      `json:"desc"`

	MarkupPercent float64 `json:"markup_percent"`

	Priority      bool      `json:"priority"`
	CrossLocked   bool      `json:"cross_locked"`
	StockReserved int       `json:"stock_reserved"`
	Created       time.Time `json:"created"`
}

func (Product) TableName() string {
	return "products"
}
