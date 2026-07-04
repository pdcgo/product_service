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

// itemNames flattens the search results to their names for order-independent checks.
func itemNames(items []*product_iface.ProductSearchItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

// ProductSearch matches on name OR ref_id (the request oneof), optionally scoped to a
// team, excludes deleted products, and caps the result with `limit`. Note: the handler
// applies LIMIT unconditionally, so every request must set a positive Limit.
func TestProductSearch(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product search",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}))
				svc := product.NewProductService(db)

				assert.NoError(t, db.Create(&[]product_models.Product{
					{TeamID: 7, Name: "Alpha", RefID: "A", Image: datatypes.NewJSONSlice([]string{"a.jpg"})},
					{TeamID: 8, Name: "Beta", RefID: "B", Image: datatypes.NewJSONSlice([]string{"b.jpg"})},
					{TeamID: 7, Name: "Gamma", RefID: "G", Image: datatypes.NewJSONSlice([]string{"g.jpg"})},
					{TeamID: 7, Name: "Zeta", RefID: "Z", Deleted: true, Image: datatypes.NewJSONSlice([]string{"z.jpg"})},
				}).Error)

				t.Run("by name", func(t *testing.T) {
					res, err := svc.ProductSearch(t.Context(), connect.NewRequest(&product_iface.ProductSearchRequest{
						Search: &product_iface.ProductSearchRequest_Name{Name: "alp"},
						Limit:  100,
					}))
					assert.NoError(t, err)
					assert.Equal(t, []string{"Alpha"}, itemNames(res.Msg.Products))
				})

				t.Run("by ref_id", func(t *testing.T) {
					res, err := svc.ProductSearch(t.Context(), connect.NewRequest(&product_iface.ProductSearchRequest{
						Search: &product_iface.ProductSearchRequest_RefId{RefId: "g"},
						Limit:  100,
					}))
					assert.NoError(t, err)
					assert.Equal(t, []string{"Gamma"}, itemNames(res.Msg.Products))
				})

				t.Run("scoped to a team", func(t *testing.T) {
					// "a" matches Alpha, Beta, Gamma (and the deleted Zeta); team 7 + non-deleted
					// leaves Alpha and Gamma.
					res, err := svc.ProductSearch(t.Context(), connect.NewRequest(&product_iface.ProductSearchRequest{
						TeamId: 7,
						Search: &product_iface.ProductSearchRequest_Name{Name: "a"},
						Limit:  100,
					}))
					assert.NoError(t, err)
					assert.ElementsMatch(t, []string{"Alpha", "Gamma"}, itemNames(res.Msg.Products))
				})

				t.Run("excludes deleted products", func(t *testing.T) {
					res, err := svc.ProductSearch(t.Context(), connect.NewRequest(&product_iface.ProductSearchRequest{
						Search: &product_iface.ProductSearchRequest_Name{Name: "zeta"},
						Limit:  100,
					}))
					assert.NoError(t, err)
					assert.Empty(t, res.Msg.Products)
				})

				t.Run("limit caps the result", func(t *testing.T) {
					res, err := svc.ProductSearch(t.Context(), connect.NewRequest(&product_iface.ProductSearchRequest{
						Search: &product_iface.ProductSearchRequest_Name{Name: "a"}, // matches 3 live products
						Limit:  1,
					}))
					assert.NoError(t, err)
					assert.Len(t, res.Msg.Products, 1)
				})
			})
		},
	)
}
