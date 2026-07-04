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
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ProductByIDs is a raw id-lookup: it does NOT scope by team and does NOT filter on
// the `deleted` flag (by design — callers resolve products by explicit id, e.g. for
// historical orders). These tests pin that behavior.
func TestProductByIDs(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product by ids",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}))
				svc := product.NewProductService(db)

				products := []product_models.Product{
					{TeamID: 7, Name: "Alpha", RefID: "A", Image: datatypes.NewJSONSlice([]string{"a1.jpg", "a2.jpg"})},
					{TeamID: 8, Name: "Beta", RefID: "B", Image: datatypes.NewJSONSlice([]string{"b1.jpg"})},
					{TeamID: 7, Name: "Gamma", RefID: "G", Image: datatypes.NewJSONSlice([]string{"g1.jpg"})},
				}
				assert.NoError(t, db.Create(&products).Error)
				a, b, g := uint64(products[0].ID), uint64(products[1].ID), uint64(products[2].ID)

				t.Run("maps the requested ids to product data", func(t *testing.T) {
					res, err := svc.ProductByIDs(t.Context(), connect.NewRequest(&product_iface.ProductByIDsRequest{
						Ids: []uint64{a, g},
					}))
					assert.NoError(t, err)
					assert.Len(t, res.Msg.Products, 2)
					assert.NotContains(t, res.Msg.Products, b) // only the ids asked for

					alpha := res.Msg.Products[a]
					assert.NotNil(t, alpha)
					assert.Equal(t, a, alpha.Id)
					assert.Equal(t, "Alpha", alpha.Name)
					assert.Equal(t, "A", alpha.RefId)
					assert.Equal(t, "a1.jpg", alpha.Image) // image ->> 0 (first element)
					assert.Equal(t, uint64(7), alpha.TeamId)

					assert.Equal(t, "Gamma", res.Msg.Products[g].Name)
				})

				t.Run("returns rows regardless of team or deleted flag", func(t *testing.T) {
					// Mark Alpha deleted; ByIDs has no soft-delete filter, so it is still returned.
					assert.NoError(t, db.
						Model(&product_models.Product{}).
						Where("id = ?", a).
						Update("deleted", true).
						Error)
					res, err := svc.ProductByIDs(t.Context(), connect.NewRequest(&product_iface.ProductByIDsRequest{
						Ids: []uint64{a},
					}))
					assert.NoError(t, err)
					assert.Contains(t, res.Msg.Products, a)
				})
			})
		},
	)
}
