package category_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/category"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"github.com/pdcgo/shared/pkg/moretest"
	"github.com/pdcgo/shared/pkg/moretest/moretest_mock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// findChild returns the child with the given name, or nil.
func findChild(node *product_iface.Category, name string) *product_iface.Category {
	for _, ch := range node.Children {
		if ch.Name == name {
			return ch
		}
	}
	return nil
}

func TestCategoryList(t *testing.T) {
	var scenario moretest_mock.DbScenario
	moretest.Suite(t, "category list",
		moretest.SetupListFunc{moretest_mock.MockPostgresDatabase(&scenario)},
		func(t *testing.T) {
			scenario(t, func(db *gorm.DB) {
				assert.NoError(t, db.AutoMigrate(&product_models.Category{}))
				svc := category.NewCategoryService(db)

				create := func(name string, parent uint64) uint64 {
					res, err := svc.CategoryCreate(t.Context(), connect.NewRequest(&product_iface.CategoryCreateRequest{Name: name, ParentId: parent}))
					assert.NoError(t, err)
					return res.Msg.Id
				}
				// Electronics > {Phones > iPhone, Laptops}; Food
				e := create("Electronics", 0)
				p := create("Phones", e)
				create("iPhone", p)
				create("Laptops", e)
				create("Food", 0)

				t.Run("nested tree from roots", func(t *testing.T) {
					res, err := svc.CategoryList(t.Context(), connect.NewRequest(&product_iface.CategoryListRequest{}))
					assert.NoError(t, err)
					assert.Len(t, res.Msg.Data, 2) // Electronics, Food
					assert.Equal(t, "Electronics", res.Msg.Data[0].Name)
					electronics := res.Msg.Data[0]
					assert.Len(t, electronics.Children, 2) // Phones, Laptops
					phones := findChild(electronics, "Phones")
					assert.NotNil(t, phones)
					assert.Len(t, phones.Children, 1) // iPhone
					assert.Equal(t, "iPhone", phones.Children[0].Name)
					assert.Equal(t, int32(2), phones.Children[0].Level)
				})

				t.Run("subtree via parent_id", func(t *testing.T) {
					res, err := svc.CategoryList(t.Context(), connect.NewRequest(&product_iface.CategoryListRequest{ParentId: e}))
					assert.NoError(t, err)
					assert.Len(t, res.Msg.Data, 2) // Phones, Laptops
				})

				t.Run("search returns flat matches", func(t *testing.T) {
					res, err := svc.CategoryList(t.Context(), connect.NewRequest(&product_iface.CategoryListRequest{Search: "phone"}))
					assert.NoError(t, err)
					assert.Len(t, res.Msg.Data, 2) // Phones, iPhone
					for _, c := range res.Msg.Data {
						assert.Empty(t, c.Children)
					}
				})
			})
		},
	)
}
