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

// ProductOrderInfo is the dedicated order-create lookup: name + owner (+ owned check
// against the requesting team) + cross config (markup_percent, cross_locked), one flat
// map keyed by id, missing ids omitted.
func TestProductOrderInfo(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "product order info",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Product{}))
				svc := product.NewProductService(db)

				products := []product_models.Product{
					{TeamID: 7, Name: "Owned", MarkupPercent: 10},
					{TeamID: 8, Name: "Cross", MarkupPercent: 20},
					{TeamID: 8, Name: "Locked", MarkupPercent: 30, CrossLocked: true},
				}
				assert.NoError(t, db.Create(&products).Error)
				owned, cross, locked := uint64(products[0].ID), uint64(products[1].ID), uint64(products[2].ID)

				res, err := svc.ProductOrderInfo(t.Context(), connect.NewRequest(&product_iface.ProductOrderInfoRequest{
					ProductIds: []uint64{owned, cross, locked, 999999},
					TeamId:     7,
				}))
				assert.NoError(t, err)
				assert.Len(t, res.Msg.Items, 3) // unknown id omitted

				o := res.Msg.Items[owned]
				assert.Equal(t, "Owned", o.Name)
				assert.True(t, o.Owned)
				assert.Equal(t, uint64(7), o.OwnerTeamId)
				assert.InDelta(t, 10, o.MarkupPercent, 0.001)
				assert.False(t, o.CrossLocked)

				c := res.Msg.Items[cross]
				assert.False(t, c.Owned)
				assert.Equal(t, uint64(8), c.OwnerTeamId)
				assert.InDelta(t, 20, c.MarkupPercent, 0.001)
				assert.False(t, c.CrossLocked)

				l := res.Msg.Items[locked]
				assert.False(t, l.Owned)
				assert.True(t, l.CrossLocked)

				assert.NotContains(t, res.Msg.Items, uint64(999999))
			})
		})
}
