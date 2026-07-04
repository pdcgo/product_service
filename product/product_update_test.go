package product_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestProductUpdate(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product update",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}, &productCategoryRow{}))
				svc := product.NewProductService(db)

				created, err := svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{
					TeamId: 7, Name: "Old", RefId: "r", Images: []string{"x"}, Description: "d", MarkupPercent: 5, CrossLocked: false,
				}))
				assert.NoError(t, err)
				id := created.Msg.Id

				_, err = svc.ProductUpdate(t.Context(), connect.NewRequest(&product_iface.ProductUpdateRequest{
					Id: id, TeamId: 7, Name: "New", RefId: "r2", Images: []string{"y", "z"}, Description: "d2", MarkupPercent: 9, CrossLocked: true,
				}))
				assert.NoError(t, err)

				var prod product_models.Product
				assert.NoError(t, db.First(&prod, id).Error)
				assert.Equal(t, "New", prod.Name)
				assert.Equal(t, "r2", prod.RefID)
				assert.Equal(t, []string{"y", "z"}, []string(prod.Image))
				assert.Equal(t, "d2", prod.Desc)
				assert.Equal(t, 9.0, prod.MarkupPercent)
				assert.True(t, prod.CrossLocked)

				t.Run("another team cannot update -> not found", func(t *testing.T) {
					_, err := svc.ProductUpdate(t.Context(), connect.NewRequest(&product_iface.ProductUpdateRequest{
						Id: id, TeamId: 999, Name: "Hijack",
					}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})

				t.Run("validation", func(t *testing.T) {
					_, err := svc.ProductUpdate(t.Context(), connect.NewRequest(&product_iface.ProductUpdateRequest{Id: 0, TeamId: 7, Name: "x"}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
					_, err = svc.ProductUpdate(t.Context(), connect.NewRequest(&product_iface.ProductUpdateRequest{Id: id, TeamId: 7, Name: ""}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})

				t.Run("replaces the category association", func(t *testing.T) {
					// Assign category 9.
					_, err := svc.ProductUpdate(t.Context(), connect.NewRequest(&product_iface.ProductUpdateRequest{
						Id: id, TeamId: 7, Name: "New", CategoryId: 9,
					}))
					assert.NoError(t, err)
					var has9 int64
					assert.NoError(t, db.
						Model(&productCategoryRow{}).
						Where("product_id = ? AND category_id = ?", id, 9).
						Count(&has9).
						Error)
					assert.Equal(t, int64(1), has9)

					// Re-assign category 10 → old join removed, new one inserted (exactly one).
					_, err = svc.ProductUpdate(t.Context(), connect.NewRequest(&product_iface.ProductUpdateRequest{
						Id: id, TeamId: 7, Name: "New", CategoryId: 10,
					}))
					assert.NoError(t, err)
					var total, has10 int64
					assert.NoError(t, db.
						Model(&productCategoryRow{}).
						Where("product_id = ?", id).
						Count(&total).
						Error)
					assert.Equal(t, int64(1), total) // exactly one association survives
					assert.NoError(t, db.
						Model(&productCategoryRow{}).
						Where("product_id = ? AND category_id = ?", id, 10).
						Count(&has10).
						Error)
					assert.Equal(t, int64(1), has10)
				})

				t.Run("soft-deleted product -> not found", func(t *testing.T) {
					gone, err := svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{TeamId: 7, Name: "Gone"}))
					assert.NoError(t, err)
					_, err = svc.ProductDelete(t.Context(), connect.NewRequest(&product_iface.ProductDeleteRequest{Id: gone.Msg.Id, TeamId: 7}))
					assert.NoError(t, err)
					_, err = svc.ProductUpdate(t.Context(), connect.NewRequest(&product_iface.ProductUpdateRequest{Id: gone.Msg.Id, TeamId: 7, Name: "Zombie"}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})
			})
		},
	)
}
