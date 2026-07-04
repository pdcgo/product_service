package product_test

import (
	"context"
	"testing"

	role_base "github.com/pdcgo/schema/services/role_base/v1"
	"github.com/pdcgo/user_service/access_interceptors"
)

// authCtx returns a context carrying the given v2 identity (user id), as the access
// interceptor would set it — the handlers read the acting user via GetIdentityFromCtx.
func authCtx(t *testing.T, userID uint32) context.Context {
	return access_interceptors.SetIdentityToCtx(t.Context(), &role_base.Identity{IdentityId: userID})
}

// productCategoryRow stands in for the legacy `product_category` m2m join table.
type productCategoryRow struct {
	ProductID  uint
	CategoryID uint
}

func (productCategoryRow) TableName() string { return "product_category" }
