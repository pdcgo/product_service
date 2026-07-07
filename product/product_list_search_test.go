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

// searchNames flattens the results to names for order-independent checks.
func searchNames(items []*product_iface.ProductListSearchItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

// ProductListSearch is the fastest-ops picker search: one q matches product name OR
// product code (ref_id), team-scoped, excludes deleted, newest first, capped limit.
func TestProductListSearch(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product list search",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}))
				svc := product.NewProductService(db)

				assert.NoError(t, db.Create(&[]product_models.Product{
					{TeamID: 7, Name: "Cotton Tee", RefID: "H2O-GH-P-X-1", Image: datatypes.NewJSONSlice([]string{"a.jpg"})},
					{TeamID: 7, Name: "Denim Jacket", RefID: "H2O-GH-P-X-2", Image: datatypes.NewJSONSlice([]string{"b.jpg"})},
					{TeamID: 7, Name: "Canvas Tote", RefID: "OLD-9", Deleted: true},
					{TeamID: 8, Name: "Cotton Scarf", RefID: "ZZ-1"}, // other team
				}).Error)

				search := func(q string, limit int64) *product_iface.ProductListSearchResponse {
					res, err := svc.ProductListSearch(t.Context(), connect.NewRequest(&product_iface.ProductListSearchRequest{
						TeamId: 7, Q: q, Limit: limit,
					}))
					assert.NoError(t, err)
					return res.Msg
				}

				t.Run("one q matches name or code", func(t *testing.T) {
					byName := search("cotton", 0)
					assert.Equal(t, []string{"Cotton Tee"}, searchNames(byName.Products)) // team 8's scarf excluded

					byCode := search("p-x-2", 0)
					assert.Equal(t, []string{"Denim Jacket"}, searchNames(byCode.Products))
					assert.Equal(t, "H2O-GH-P-X-2", byCode.Products[0].RefId)
					assert.Equal(t, "b.jpg", byCode.Products[0].Image) // image ->> 0
				})

				t.Run("empty q lists newest first with default limit", func(t *testing.T) {
					res := search("", 0)
					assert.Equal(t, []string{"Denim Jacket", "Cotton Tee"}, searchNames(res.Products)) // id DESC
				})

				t.Run("excludes deleted products", func(t *testing.T) {
					res := search("canvas", 0)
					assert.Empty(t, res.Products)
				})

				t.Run("limit caps the result", func(t *testing.T) {
					res := search("", 1)
					assert.Len(t, res.Products, 1)
					capped := search("", 500) // > 100 cap; only 2 live rows anyway
					assert.Len(t, capped.Products, 2)
				})

				t.Run("team_id is required", func(t *testing.T) {
					_, err := svc.ProductListSearch(t.Context(), connect.NewRequest(&product_iface.ProductListSearchRequest{Q: "x"}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})
			})
		},
	)
}
