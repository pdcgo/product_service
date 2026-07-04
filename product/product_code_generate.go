package product

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
)

// ProductCodeGenerate implements [product_ifaceconnect.ProductServiceHandler].
//
// Previews the product code (legacy RefID) that the next created product would get,
// from the team code + the acting user's team alias + the next product id. Requires an
// authenticated user (the alias is per-user). The final code is (re)assigned by ProductCreate.
func (p *productSrvImpl) ProductCodeGenerate(
	ctx context.Context,
	req *connect.Request[product_iface.ProductCodeGenerateRequest],
) (*connect.Response[product_iface.ProductCodeGenerateResponse], error) {
	pay := req.Msg
	if pay.GetTeamId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("team_id is required"))
	}

	userID := identityUserID(ctx)
	if userID == 0 {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authenticated user required"))
	}

	db := p.db.WithContext(ctx)

	var nextID int64
	err := db.
		Model(&product_models.Product{}).
		Select("COALESCE(MAX(id),0)+1").
		Scan(&nextID).
		Error
	if err != nil {
		return nil, err
	}

	code, err := generateProductCode(db, uint(pay.GetTeamId()), userID, uint(nextID))
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&product_iface.ProductCodeGenerateResponse{ProductCode: code}), nil
}
