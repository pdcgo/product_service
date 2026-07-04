package product_test

import (
	"fmt"
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

// When a user is authenticated (team_code + alias present), ProductCreate generates the
// authoritative product code from the real inserted id and writes the category join.
func TestProductCreateGeneratesCodeAndCategory(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product create generated",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(
					&product_models.Product{},
					&teamCodeRow{},
					&userTeamRow{},
					&product_models.Category{},
					&productCategoryRow{},
				))
				assert.NoError(t, db.Create(&teamCodeRow{ID: 7, TeamCode: "H2O"}).Error)
				assert.NoError(t, db.Create(&userTeamRow{ID: 1, UserID: 5, TeamID: 7, Alias: "GH"}).Error)
				assert.NoError(t, db.Create(&product_models.Category{ID: 9, Name: "Gadgets", IsLast: true}).Error)

				svc := product.NewProductService(db)

				res, err := svc.ProductCreate(authCtx(t, 5), connect.NewRequest(&product_iface.ProductCreateRequest{
					TeamId: 7, Name: "Widget", RefId: "ignored", CategoryId: 9,
				}))
				assert.NoError(t, err)

				var prod product_models.Product
				assert.NoError(t, db.First(&prod, res.Msg.Id).Error)
				expected := fmt.Sprintf("H2O-GH-P-X-%X", prod.ID)
				assert.Equal(t, expected, prod.RefID)          // authoritative, from the real id
				assert.Equal(t, expected, res.Msg.ProductCode) // returned to the caller
				assert.Equal(t, uint(5), prod.UserID)

				var joinCount int64
				assert.NoError(t, db.Model(&productCategoryRow{}).
					Where("product_id = ? AND category_id = ?", prod.ID, 9).
					Count(&joinCount).Error)
				assert.Equal(t, int64(1), joinCount)
			})
		},
	)
}
