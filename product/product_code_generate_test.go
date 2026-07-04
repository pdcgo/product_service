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

// teamCodeRow / userTeamRow stand in for the `teams` (team_code) and `user_teams`
// (alias) tables the product-code generator reads.
type teamCodeRow struct {
	ID       uint `gorm:"primarykey"`
	TeamCode string
}

func (teamCodeRow) TableName() string { return "teams" }

type userTeamRow struct {
	ID     uint `gorm:"primarykey"`
	UserID uint
	TeamID uint
	Alias  string
}

func (userTeamRow) TableName() string { return "user_teams" }

func TestProductCodeGenerate(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product code generate",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}, &teamCodeRow{}, &userTeamRow{}))
				assert.NoError(t, db.Create(&teamCodeRow{ID: 7, TeamCode: "H2O"}).Error)
				assert.NoError(t, db.Create(&userTeamRow{ID: 1, UserID: 5, TeamID: 7, Alias: "GH"}).Error)

				svc := product.NewProductService(db)

				t.Run("previews the next product's code", func(t *testing.T) {
					// no products yet -> next id 1 -> hex "1"
					res, err := svc.ProductCodeGenerate(authCtx(t, 5), connect.NewRequest(&product_iface.ProductCodeGenerateRequest{TeamId: 7}))
					assert.NoError(t, err)
					assert.Equal(t, "H2O-GH-P-X-1", res.Msg.ProductCode)
				})

				t.Run("unauthenticated is rejected", func(t *testing.T) {
					// plain context (no identity injected) -> no user id
					_, err := svc.ProductCodeGenerate(t.Context(), connect.NewRequest(&product_iface.ProductCodeGenerateRequest{TeamId: 7}))
					assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
				})

				t.Run("validation", func(t *testing.T) {
					_, err := svc.ProductCodeGenerate(authCtx(t, 5), connect.NewRequest(&product_iface.ProductCodeGenerateRequest{TeamId: 0}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})
			})
		},
	)
}
