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

func TestCategoryCreate(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "category create",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Category{}))
				svc := category.NewCategoryService(db)

				root, err := svc.CategoryCreate(t.Context(), connect.NewRequest(&product_iface.CategoryCreateRequest{Name: "Electronics"}))
				assert.NoError(t, err)
				assert.NotZero(t, root.Msg.Id)

				var r product_models.Category
				assert.NoError(t, db.First(&r, root.Msg.Id).Error)
				assert.Equal(t, "Electronics", r.Name)
				assert.Equal(t, 0, r.Level)
				assert.Nil(t, r.ParentID)
				assert.True(t, r.IsLast)

				child, err := svc.CategoryCreate(t.Context(), connect.NewRequest(&product_iface.CategoryCreateRequest{Name: "Phones", ParentId: root.Msg.Id}))
				assert.NoError(t, err)

				var c product_models.Category
				assert.NoError(t, db.First(&c, child.Msg.Id).Error)
				assert.Equal(t, 1, c.Level)
				assert.NotNil(t, c.ParentID)
				assert.Equal(t, uint(root.Msg.Id), *c.ParentID)
				assert.True(t, c.IsLast)

				// The parent is no longer a leaf.
				assert.NoError(t, db.First(&r, root.Msg.Id).Error)
				assert.False(t, r.IsLast)

				t.Run("validation", func(t *testing.T) {
					_, err := svc.CategoryCreate(t.Context(), connect.NewRequest(&product_iface.CategoryCreateRequest{Name: ""}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})

				t.Run("missing parent is not found", func(t *testing.T) {
					_, err := svc.CategoryCreate(t.Context(), connect.NewRequest(&product_iface.CategoryCreateRequest{Name: "x", ParentId: 999999}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})
			})
		},
	)
}
