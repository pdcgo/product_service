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

func TestProductCreate(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product create",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}))
				svc := product.NewProductService(db)

				res, err := svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{
					TeamId:        7,
					Name:          "Cotton Tee",
					RefId:         "REF-1",
					Images:        []string{"a.jpg", "b.jpg"},
					Description:   "soft",
					MarkupPercent: 12.5,
					CrossLocked:   true,
				}))
				assert.NoError(t, err)
				assert.NotZero(t, res.Msg.Id)

				var prod product_models.Product
				assert.NoError(t, db.First(&prod, res.Msg.Id).Error)
				assert.Equal(t, uint(7), prod.TeamID)
				assert.Equal(t, "Cotton Tee", prod.Name)
				assert.Equal(t, "REF-1", prod.RefID)
				assert.Equal(t, []string{"a.jpg", "b.jpg"}, []string(prod.Image))
				assert.Equal(t, "soft", prod.Desc)
				assert.Equal(t, 12.5, prod.MarkupPercent)
				assert.True(t, prod.CrossLocked)
				assert.False(t, prod.Deleted)

				t.Run("validation", func(t *testing.T) {
					_, err := svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{TeamId: 0, Name: "x"}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
					_, err = svc.ProductCreate(t.Context(), connect.NewRequest(&product_iface.ProductCreateRequest{TeamId: 7, Name: ""}))
					assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				})
			})
		},
	)
}
