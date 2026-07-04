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

func TestProductDelete(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product delete",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}))
				svc := product.NewProductService(db)

				p1, err := svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{TeamId: 7, Name: "P1"}))
				assert.NoError(t, err)
				p2, err := svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{TeamId: 7, Name: "P2"}))
				assert.NoError(t, err)

				_, err = svc.ProductDelete(t.Context(), connect.NewRequest(&product_iface.ProductDeleteRequest{Id: p1.Msg.Id, TeamId: 7}))
				assert.NoError(t, err)

				// soft delete: the row remains but is flagged.
				var prod product_models.Product
				assert.NoError(t, db.First(&prod, p1.Msg.Id).Error)
				assert.True(t, prod.Deleted)

				t.Run("deleting again -> not found", func(t *testing.T) {
					_, err := svc.ProductDelete(t.Context(), connect.NewRequest(&product_iface.ProductDeleteRequest{Id: p1.Msg.Id, TeamId: 7}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})

				t.Run("another team cannot delete -> not found", func(t *testing.T) {
					_, err := svc.ProductDelete(t.Context(), connect.NewRequest(&product_iface.ProductDeleteRequest{Id: p2.Msg.Id, TeamId: 999}))
					assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				})

				t.Run("validation", func(t *testing.T) {
					_, err := svc.ProductDelete(t.Context(), connect.NewRequest(&product_iface.ProductDeleteRequest{Id: 0, TeamId: 7}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})
			})
		},
	)
}
