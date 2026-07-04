package product

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/pdcgo/product_service/product_models"
	product_iface "github.com/pdcgo/schema/services/product_iface/v1"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ProductCreate implements [product_ifaceconnect.ProductServiceHandler].
//
// Creates a product owned by team_id in the legacy `products` table. The product code
// (ref_id) is generated authoritatively from the real inserted id + team code + the
// acting user's team alias when a user is authenticated; otherwise the passed ref_id is
// kept. When category_id is set, a product_category association row is written.
func (p *productSrvImpl) ProductCreate(
	ctx context.Context,
	req *connect.Request[product_iface.ProductCreateRequest],
) (*connect.Response[product_iface.ProductCreateResponse], error) {
	pay := req.Msg
	if pay.GetTeamId() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("team_id is required"))
	}
	if pay.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	userID := identityUserID(ctx)

	prod := product_models.Product{
		TeamID:        uint(pay.GetTeamId()),
		UserID:        userID,
		Name:          pay.GetName(),
		RefID:         pay.GetRefId(),
		Image:         datatypes.NewJSONSlice(pay.GetImages()),
		Desc:          pay.GetDescription(),
		MarkupPercent: pay.GetMarkupPercent(),
		CrossLocked:   pay.GetCrossLocked(),
		Deleted:       false,
		Created:       time.Now(),
	}

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		cerr := tx.Create(&prod).Error
		if cerr != nil {
			return cerr
		}

		// Authoritative product code from the real id (best-effort: needs a team_code +
		// the user's alias; on failure the passed ref_id is kept).
		if userID > 0 {
			code, gerr := generateProductCode(tx, uint(pay.GetTeamId()), userID, prod.ID)
			if gerr == nil && code != "" {
				uerr := tx.
					Model(&prod).
					Update("ref_id", code).
					Error
				if uerr != nil {
					return uerr
				}
				prod.RefID = code
			}
		}

		if pay.GetCategoryId() > 0 {
			jerr := tx.
				Exec("INSERT INTO product_category (product_id, category_id) VALUES (?, ?)", prod.ID, pay.GetCategoryId()).
				Error
			if jerr != nil {
				return jerr
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&product_iface.ProductCreateResponse{
		Id:          uint64(prod.ID),
		ProductCode: prod.RefID,
	}), nil
}
