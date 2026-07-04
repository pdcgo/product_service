package product

import (
	"context"

	"github.com/pdcgo/shared/db_models"
	"github.com/pdcgo/user_service/access_interceptors"
	"gorm.io/gorm"
)

// identityUserID resolves the acting user's id from the v2 identity the access
// interceptor stored in the context, or 0 when there is no authenticated identity.
func identityUserID(ctx context.Context) uint {
	id, err := access_interceptors.GetIdentityFromCtx(ctx)
	if err != nil || id == nil {
		return 0
	}
	return uint(id.IdentityId)
}

// generateProductCode builds the legacy product code (RefID: TEAMCODE-ALIAS-P-X-HEXID)
// for a product owned by teamID and created by userID. Errors if the team has no
// team_code or the user has no alias in the team.
func generateProductCode(db *gorm.DB, teamID, userID, productID uint) (string, error) {
	teamCode, err := db_models.GetTeamCode(db, teamID)
	if err != nil {
		return "", err
	}

	var ut db_models.UserTeam
	err = db.
		Where("user_id = ? AND team_id = ?", userID, teamID).
		First(&ut).
		Error
	if err != nil {
		return "", err
	}

	ref, err := db_models.NewRefID(&db_models.RefData{
		TeamCode: teamCode,
		UserCode: ut.Alias,
		RefType:  db_models.ProductRef,
		RefIDs:   []uint{productID},
	})
	if err != nil {
		return "", err
	}
	return string(ref), nil
}
