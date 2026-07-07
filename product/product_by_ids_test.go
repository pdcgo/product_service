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

// ProductByIDs follows the by-IDs guideline: filter + data_request in, a map keyed by
// product id out (one item per requested type per id). It is an explicit-id lookup —
// soft-deleted products are still returned by design (e.g. historical orders);
// team_id is an optional scope. These tests pin that behavior.
func TestProductByIDs(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product by ids",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				// teamRow is defined in product_list_test.go (same package).
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}, &teamRow{}))
				assert.NoError(t, db.Create(&teamRow{ID: 7, Name: "Team G"}).Error)
				svc := product.NewProductService(db)

				products := []product_models.Product{
					{TeamID: 7, Name: "Alpha", RefID: "A", Image: datatypes.NewJSONSlice([]string{"a1.jpg", "a2.jpg"})},
					{TeamID: 8, Name: "Beta", RefID: "B", Image: datatypes.NewJSONSlice([]string{"b1.jpg"})},
					{TeamID: 7, Name: "Gamma", RefID: "G", Image: datatypes.NewJSONSlice([]string{"g1.jpg"})},
				}
				assert.NoError(t, db.Create(&products).Error)
				a, b, g := uint64(products[0].ID), uint64(products[1].ID), uint64(products[2].ID)

				byIds := func(filter *product_iface.ProductByIdsFilter, types ...product_iface.ProductByIdsDataType) *product_iface.ProductByIDsResponse {
					res, err := svc.ProductByIDs(t.Context(), connect.NewRequest(&product_iface.ProductByIDsRequest{
						Filter:      filter,
						DataRequest: types,
					}))
					assert.NoError(t, err)
					return res.Msg
				}
				all := []product_iface.ProductByIdsDataType{
					product_iface.ProductByIdsDataType_PRODUCT_BY_IDS_DATA_TYPE_GENERAL,
					product_iface.ProductByIdsDataType_PRODUCT_BY_IDS_DATA_TYPE_TEAM,
				}

				t.Run("maps the requested ids with one item per requested type", func(t *testing.T) {
					res := byIds(&product_iface.ProductByIdsFilter{Ids: []uint64{a, g}}, all...)
					assert.Len(t, res.Items, 2)
					assert.NotContains(t, res.Items, b) // only the ids asked for

					list := res.Items[a]
					assert.NotNil(t, list)
					assert.Len(t, list.Items, 2) // GENERAL + TEAM, in data_request order

					gen := list.Items[0].GetGeneral()
					assert.NotNil(t, gen)
					assert.Equal(t, a, gen.GetId())
					assert.Equal(t, "Alpha", gen.GetName())
					assert.Equal(t, "A", gen.GetRefId())
					assert.Equal(t, "a1.jpg", gen.GetImage()) // image ->> 0 (first element)

					team := list.Items[1].GetTeam()
					assert.NotNil(t, team)
					assert.Equal(t, uint64(7), team.GetTeamId())
					assert.Equal(t, "Team G", team.GetTeamName())

					assert.Equal(t, "Gamma", res.Items[g].Items[0].GetGeneral().GetName())
				})

				t.Run("data_request selects the returned types", func(t *testing.T) {
					res := byIds(&product_iface.ProductByIdsFilter{Ids: []uint64{a}},
						product_iface.ProductByIdsDataType_PRODUCT_BY_IDS_DATA_TYPE_GENERAL)
					list := res.Items[a]
					assert.Len(t, list.Items, 1)
					assert.NotNil(t, list.Items[0].GetGeneral())
					assert.Nil(t, list.Items[0].GetTeam())
				})

				t.Run("team_id scope filters other teams' ids", func(t *testing.T) {
					res := byIds(&product_iface.ProductByIdsFilter{Ids: []uint64{a, b}, TeamId: 7}, all...)
					assert.Contains(t, res.Items, a)
					assert.NotContains(t, res.Items, b) // team 8
				})

				t.Run("soft-deleted products are still returned (by design)", func(t *testing.T) {
					assert.NoError(t, db.
						Model(&product_models.Product{}).
						Where("id = ?", a).
						Update("deleted", true).
						Error)
					res := byIds(&product_iface.ProductByIdsFilter{Ids: []uint64{a}}, all...)
					assert.Contains(t, res.Items, a)
				})

				t.Run("validation", func(t *testing.T) {
					_, err := svc.ProductByIDs(t.Context(), connect.NewRequest(&product_iface.ProductByIDsRequest{
						DataRequest: all,
					}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err)) // no filter/ids

					_, err = svc.ProductByIDs(t.Context(), connect.NewRequest(&product_iface.ProductByIDsRequest{
						Filter: &product_iface.ProductByIdsFilter{Ids: []uint64{a}},
					}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err)) // no data_request
				})
			})
		},
	)
}
