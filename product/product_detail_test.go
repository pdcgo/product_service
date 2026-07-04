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

func TestProductDetail(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product detail",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				// teamRow is defined in product_list_test.go (same package).
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}, &teamRow{}, &product_models.Category{}, &productCategoryRow{}))
				assert.NoError(t, db.Create(&teamRow{ID: 7, Name: "Team G"}).Error)
				svc := product.NewProductService(db)

				created, err := svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{
					TeamId: 7, Name: "Widget", RefId: "r-w", Images: []string{"1.jpg", "2.jpg"}, Description: "desc", MarkupPercent: 15, CrossLocked: true,
				}))
				assert.NoError(t, err)
				id := created.Msg.Id

				t.Run("returns all editable fields + team name", func(t *testing.T) {
					res, err := svc.ProductDetail(t.Context(), connect.NewRequest(&product_iface.ProductDetailRequest{Id: id, TeamId: 7}))
					assert.NoError(t, err)
					assert.Equal(t, id, res.Msg.Id)
					assert.Equal(t, uint64(7), res.Msg.TeamId)
					assert.Equal(t, "Team G", res.Msg.TeamName)
					assert.Equal(t, "Widget", res.Msg.Name)
					assert.Equal(t, "r-w", res.Msg.RefId)
					assert.Equal(t, []string{"1.jpg", "2.jpg"}, res.Msg.Images)
					assert.Equal(t, "desc", res.Msg.Description)
					assert.Equal(t, float64(15), res.Msg.MarkupPercent)
					assert.True(t, res.Msg.CrossLocked)
				})

				t.Run("returns the assigned category", func(t *testing.T) {
					assert.NoError(t, db.Create(&product_models.Category{ID: 42, Name: "Gadgets", IsLast: true}).Error)
					assert.NoError(t, db.Create(&productCategoryRow{ProductID: uint(id), CategoryID: 42}).Error)
					res, err := svc.ProductDetail(t.Context(), connect.NewRequest(&product_iface.ProductDetailRequest{Id: id, TeamId: 7}))
					assert.NoError(t, err)
					assert.Equal(t, uint64(42), res.Msg.CategoryId)
					assert.Equal(t, "Gadgets", res.Msg.CategoryName)
				})

				t.Run("category still resolves after it is soft-deleted", func(t *testing.T) {
					// Soft-delete category 42 (assigned above). A product already joined to it
					// must still resolve the name — ProductDetail reads it via a raw join that
					// ignores the gorm soft-delete scope (the "better compatibility" behavior).
					assert.NoError(t, db.Delete(&product_models.Category{}, 42).Error)

					var live int64
					assert.NoError(t, db.
						Model(&product_models.Category{}).
						Where("id = ?", 42).
						Count(&live).
						Error)
					assert.Equal(t, int64(0), live) // hidden from normal category queries

					res, err := svc.ProductDetail(t.Context(), connect.NewRequest(&product_iface.ProductDetailRequest{Id: id, TeamId: 7}))
					assert.NoError(t, err)
					assert.Equal(t, uint64(42), res.Msg.CategoryId)
					assert.Equal(t, "Gadgets", res.Msg.CategoryName)
				})

				t.Run("another team cannot read -> not found", func(t *testing.T) {
					_, err := svc.ProductDetail(t.Context(), connect.NewRequest(&product_iface.ProductDetailRequest{Id: id, TeamId: 999}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})

				t.Run("validation", func(t *testing.T) {
					_, err := svc.ProductDetail(t.Context(), connect.NewRequest(&product_iface.ProductDetailRequest{Id: 0, TeamId: 7}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})

				t.Run("soft-deleted product -> not found", func(t *testing.T) {
					gone, err := svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{TeamId: 7, Name: "Gone"}))
					assert.NoError(t, err)
					_, err = svc.ProductDelete(t.Context(), connect.NewRequest(&product_iface.ProductDeleteRequest{Id: gone.Msg.Id, TeamId: 7}))
					assert.NoError(t, err)
					_, err = svc.ProductDetail(t.Context(), connect.NewRequest(&product_iface.ProductDetailRequest{Id: gone.Msg.Id, TeamId: 7}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})
			})
		},
	)
}
