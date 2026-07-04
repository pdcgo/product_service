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

func TestCategoryUpdate(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "category update",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Category{}))
				svc := category.NewCategoryService(db)

				create := func(name string, parent uint64) uint64 {
					res, err := svc.CategoryCreate(t.Context(), connect.NewRequest(&product_iface.CategoryCreateRequest{Name: name, ParentId: parent}))
					assert.NoError(t, err)
					return res.Msg.Id
				}
				// A > B > C ; X > Y
				a := create("A", 0)
				b := create("B", a)
				c := create("C", b)
				x := create("X", 0)
				y := create("Y", x)

				// Reparent B (and its subtree C) under Y: levels shift by +1, is_last flips.
				_, err := svc.CategoryUpdate(t.Context(), connect.NewRequest(&product_iface.CategoryUpdateRequest{Id: b, Name: "B", ParentId: y}))
				assert.NoError(t, err)

				var bRow, cRow, aRow, yRow product_models.Category
				assert.NoError(t, db.First(&bRow, b).Error)
				assert.NoError(t, db.First(&cRow, c).Error)
				assert.NoError(t, db.First(&aRow, a).Error)
				assert.NoError(t, db.First(&yRow, y).Error)
				assert.Equal(t, 2, bRow.Level) // Y(1) + 1
				assert.Equal(t, uint(y), *bRow.ParentID)
				assert.Equal(t, 3, cRow.Level) // shifted with the subtree
				assert.True(t, aRow.IsLast)    // A lost its only child
				assert.False(t, yRow.IsLast)   // Y gained a child

				t.Run("rename only keeps level/parent", func(t *testing.T) {
					_, err := svc.CategoryUpdate(t.Context(), connect.NewRequest(&product_iface.CategoryUpdateRequest{Id: a, Name: "Alpha", ParentId: 0}))
					assert.NoError(t, err)
					var row product_models.Category
					assert.NoError(t, db.First(&row, a).Error)
					assert.Equal(t, "Alpha", row.Name)
					assert.Equal(t, 0, row.Level)
					assert.Nil(t, row.ParentID)
				})

				t.Run("cycle guard rejects moving under a descendant", func(t *testing.T) {
					// X still owns Y (and now Y owns B,C) — moving X under Y is a cycle.
					_, err := svc.CategoryUpdate(t.Context(), connect.NewRequest(&product_iface.CategoryUpdateRequest{Id: x, Name: "X", ParentId: y}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})

				t.Run("unknown id is not found", func(t *testing.T) {
					_, err := svc.CategoryUpdate(t.Context(), connect.NewRequest(&product_iface.CategoryUpdateRequest{Id: 999999, Name: "z"}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})
			})
		},
	)
}
