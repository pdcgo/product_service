package category_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/category"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// productCategoryRow stands in for the legacy `product_category` m2m join. A
// soft-delete PRESERVES these rows so already-inserted products keep resolving their
// category — the assertions below check they survive the cascade.
type productCategoryRow struct {
	ProductID  uint
	CategoryID uint
}

func (productCategoryRow) TableName() string { return "product_category" }

func TestCategoryDelete(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "category delete",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Category{}, &productCategoryRow{}))
				svc := category.NewCategoryService(db)

				create := func(name string, parent uint64) uint64 {
					res, err := svc.CategoryCreate(t.Context(), connect.NewRequest(&product_iface.CategoryCreateRequest{Name: name, ParentId: parent}))
					assert.NoError(t, err)
					return res.Msg.Id
				}
				// A > B > C ; D
				a := create("A", 0)
				b := create("B", a)
				c := create("C", b)
				d := create("D", 0)

				assert.NoError(t, db.Create(&[]productCategoryRow{
					{ProductID: 1, CategoryID: uint(c)},
					{ProductID: 1, CategoryID: uint(b)},
					{ProductID: 2, CategoryID: uint(d)},
				}).Error)

				// Delete B → soft-deletes B + C (cascade), preserves their join rows,
				// A becomes a leaf again.
				_, err := svc.CategoryDelete(t.Context(), connect.NewRequest(&product_iface.CategoryDeleteRequest{Id: b}))
				assert.NoError(t, err)

				// Soft-deleted: hidden from normal (model-scoped) queries...
				var liveCount int64
				assert.NoError(t, db.Model(&product_models.Category{}).Where("id IN ?", []uint{uint(b), uint(c)}).Count(&liveCount).Error)
				assert.Equal(t, int64(0), liveCount) // B and C no longer listed

				// ...but still physically present with deleted_at set (Unscoped sees them).
				var deadCount int64
				assert.NoError(t, db.
					Unscoped().
					Model(&product_models.Category{}).
					Where("id IN ?", []uint{uint(b), uint(c)}).
					Count(&deadCount).
					Error)
				assert.Equal(t, int64(2), deadCount)

				var bRow product_models.Category
				assert.NoError(t, db.Unscoped().First(&bRow, b).Error)
				assert.True(t, bRow.DeletedAt.Valid) // deleted_at populated

				var aRow product_models.Category
				assert.NoError(t, db.First(&aRow, a).Error)
				assert.True(t, aRow.IsLast) // A lost its only child

				// The product_category joins are PRESERVED (all three) — including the ones
				// for the soft-deleted B and C. That is the whole point of soft-delete.
				var joinCount int64
				assert.NoError(t, db.Model(&productCategoryRow{}).Count(&joinCount).Error)
				assert.Equal(t, int64(3), joinCount)

				t.Run("unknown id is not found", func(t *testing.T) {
					_, err := svc.CategoryDelete(t.Context(), connect.NewRequest(&product_iface.CategoryDeleteRequest{Id: 999999}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})
			})
		},
	)
}
