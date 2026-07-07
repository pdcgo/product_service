# Database Schema & Model

Schema and Model that have :
1. Product
2. Category

## Category Schema
1. Legacy compatibility
    Because Category Schema is exist in legacy system before. we must aware about the migration. in new migration this project, create category **If Only** that table is not exist.
2. This is legacy golang struct that reflected the schema. for now, the field and schema already accomodate this system. No need to change.
    ```
    type Category struct {
        ID       uint        `json:"id" gorm:"primarykey"`
        ParentID *uint       `json:"parent_id" gorm:"index"`
        Level    int         `json:"level"`
        Name     string      `json:"name"`
        IsLast   bool        `json:"is_last"`
        Children []*Category `json:"children" gorm:"foreignKey:ParentID"`
    }
    ```
3. if on `./product_models` doesn't have golang model for definition, duplicate legacy and place at `./product_models/category.go`
4. **Soft delete** — the model adds a `DeletedAt gorm.DeletedAt` field (column `deleted_at`, added by
   migration `db_migrations/00003_add_deleted_at_to_categories.sql`, additive/legacy-safe with
   `ADD COLUMN IF NOT EXISTS`). `CategoryDelete` soft-deletes the category + its subtree so
   listings/pickers hide them while already-inserted products keep their `product_category` link
   (see [readme](readme.md) Category rules and [Testing](test.md)). The legacy `Children` relation is
   intentionally omitted from the local model — the tree is rebuilt in the handlers from flat rows.


## Product Schema
1. Legacy compatibility
    Because Product Schema is exist in legacy system before. we must aware about the migration. in new migration this project, create category **If Only** that table is not exist.
2. This is legacy golang struct that reflected the schema. for now, the field and schema already accomodate this system. No need to change.
    ```
    type Product struct {
        ID      uint  `json:"id" gorm:"primarykey"`
        TeamID  uint  `json:"team_id"`
        Deleted bool  `json:"deleted" gorm:"index"`
        UserID  uint  `json:"user_id"`
        RefID   RefID `json:"ref_id"`

        Name            string                      `json:"name"`
        Image           datatypes.JSONSlice[string] `json:"image"`
        Desc            string                      `json:"desc"`
        VariationNames  []*VariationName            `json:"variation_names"`  <---- dont include this in new model and migration
        VariationValues []*VariationValue           `json:"variation_values"` <---- dont include this in new model and migration

        MarkupPercent MarkupValue `json:"markup_percent"`

        StockReady    int       `json:"stock_ready"`      <---- dont include this in new model and migration
        StockPending  int       `json:"stock_pending"`    <---- dont include this in new model and migration
        BundleCount   int       `json:"bundle_count"`     <---- dont include this in new model and migration
        Priority      bool      `json:"priority"`
        CrossLocked   bool      `json:"cross_locked"`
        StockReserved int       `json:"stock_reserved"`
        Created       time.Time `json:"created"`

        Team     *Team       `json:"team,omitempty"`     <---- define this as foreign key in migration but dont include this in new model
        User     *User       `json:"user,omitempty"`     <---- define this as foreign key in migration but dont include this in new model
        Category []*Category `json:"category,omitempty" gorm:"many2many:product_category;"`         <---- dont include this in new model and migration
        Tags     []*Tag      `json:"tags,omitempty" gorm:"many2many:product_tags;"`                 <---- dont include this in new model and migration
    }
    ```
3. if on `./product_models` doesn't have golang model for that definition, duplicate legacy and place at `./product_models/product.go`

