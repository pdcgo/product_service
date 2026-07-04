package product_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product"
	"github.com/pdcgo/product_service/product_models"
	common "github.com/pdcgo/schema/services/common/v1"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// teamRow is a stand-in for the legacy `teams` table (used for the owner-name join
// and by product_detail_test.go — same test package).
type teamRow struct {
	ID   uint `gorm:"primarykey"`
	Name string
}

func (teamRow) TableName() string { return "teams" }

func TestProductList(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product list",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}, &teamRow{}))
				assert.NoError(t, db.Create(&[]teamRow{{ID: 1, Name: "Team A"}, {ID: 2, Name: "Team B"}}).Error)
				assert.NoError(t, db.Create(&[]product_models.Product{
					{ID: 100, TeamID: 1, Name: "Apple", RefID: "r100", Image: datatypes.NewJSONSlice([]string{"a.jpg"}), MarkupPercent: 10, CrossLocked: false},
					{ID: 200, TeamID: 1, Name: "Banana", RefID: "r200", Image: datatypes.NewJSONSlice([]string{})},
					{ID: 300, TeamID: 1, Name: "Cherry", Image: datatypes.NewJSONSlice([]string{}), Deleted: true}, // excluded: deleted
					{ID: 400, TeamID: 2, Name: "Durian", Image: datatypes.NewJSONSlice([]string{})},                // excluded: other team
				}).Error)

				svc := product.NewProductService(db)
				page := &common.PageFilter{Page: 1, Limit: 20}

				t.Run("lists the team's live products, newest first", func(t *testing.T) {
					res, err := svc.ProductList(t.Context(), connect.NewRequest(&product_iface.ProductListRequest{TeamId: 1, Page: page}))
					assert.NoError(t, err)
					assert.Len(t, res.Msg.Products, 2) // 300 (deleted) + 400 (other team) excluded
					assert.Equal(t, uint64(200), res.Msg.Products[0].Id)
					assert.Equal(t, uint64(100), res.Msg.Products[1].Id)
					assert.Equal(t, "Apple", res.Msg.Products[1].Name)
					assert.Equal(t, "a.jpg", res.Msg.Products[1].Image)
					assert.Equal(t, "Team A", res.Msg.Products[1].TeamName)
					assert.Equal(t, float64(10), res.Msg.Products[1].MarkupPercent)
					assert.Equal(t, int64(2), res.Msg.PageInfo.TotalItems)
				})

				t.Run("search filters by name", func(t *testing.T) {
					res, err := svc.ProductList(t.Context(), connect.NewRequest(&product_iface.ProductListRequest{TeamId: 1, Search: "ban", Page: page}))
					assert.NoError(t, err)
					assert.Len(t, res.Msg.Products, 1)
					assert.Equal(t, uint64(200), res.Msg.Products[0].Id)
				})

				t.Run("validation", func(t *testing.T) {
					_, err := svc.ProductList(t.Context(), connect.NewRequest(&product_iface.ProductListRequest{TeamId: 0, Page: page}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
					_, err = svc.ProductList(t.Context(), connect.NewRequest(&product_iface.ProductListRequest{TeamId: 1}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})
			})
		},
	)
}
