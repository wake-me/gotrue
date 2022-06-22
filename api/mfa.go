package api

import (
	"github.com/netlify/gotrue/crypto"
	"github.com/netlify/gotrue/models"
	"github.com/netlify/gotrue/storage"
	"net/http"
	"time"
)

// RecoveryCodesResponse repreesnts a successful recovery code generation response
type RecoveryCodesResponse struct {
	RecoveryCodes []string
}

func (a *API) EnableMFA(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user := getUser(ctx)
	instanceID := getInstanceID(ctx)
	err := a.db.Transaction(func(tx *storage.Connection) error {
		if terr := user.EnableMFA(tx); terr != nil {
			return terr
		}
		if terr := models.NewAuditLogEntry(tx, instanceID, user, models.UserModifiedAction, r.RemoteAddr, map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
			"user_phone": user.Phone,
		}); terr != nil {
			return terr
		}
		return nil
	})
	if err != nil {
		return err
	}
	return sendJSON(w, http.StatusOK, user)
}

func (a *API) DisableMFA(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	user := getUser(ctx)
	instanceID := getInstanceID(ctx)
	err := a.db.Transaction(func(tx *storage.Connection) error {
		if terr := user.DisableMFA(tx); terr != nil {
			return terr
		}
		if terr := models.NewAuditLogEntry(tx, instanceID, user, models.UserModifiedAction, r.RemoteAddr, map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
			"user_phone": user.Phone,
		}); terr != nil {
			return terr
		}
		return nil
	})
	if err != nil {
		return err
	}
	return sendJSON(w, http.StatusOK, user)
}

func (a *API) GenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) error {
	const NUM_RECOVERY_CODES = 8
	const RECOVERY_CODE_LENGTH = 8

	ctx := r.Context()
	user := getUser(ctx)
	instanceID := getInstanceID(ctx)
	if !user.MFAEnabled {
		return MFANotEnabledError
	}
	now := time.Now()
	recoveryCodeModels := []*models.RecoveryCode{}
	var terr error
	var recoveryCode string
	var recoveryCodes []string
	var recoveryCodeModel *models.RecoveryCode

	for i := 0; i < NUM_RECOVERY_CODES; i++ {
		recoveryCode = crypto.SecureToken(RECOVERY_CODE_LENGTH)
		recoveryCodeModel, terr = models.NewRecoveryCode(user, recoveryCode, &now)
		if terr != nil {
			return internalServerError("Error creating recovery code").WithInternalError(terr)
		}
		recoveryCodes = append(recoveryCodes, recoveryCode)
		recoveryCodeModels = append(recoveryCodeModels, recoveryCodeModel)
	}
	terr = a.db.Transaction(func(tx *storage.Connection) error {
		if terr = tx.Create(recoveryCodeModels); terr != nil {
			return terr
		}

		if terr := models.NewAuditLogEntry(tx, instanceID, user, models.GenerateRecoveryCodesAction, r.RemoteAddr, nil); terr != nil {
			return terr
		}
		return nil
	})

	return sendJSON(w, http.StatusOK, &RecoveryCodesResponse{
		RecoveryCodes: recoveryCodes,
	})
}
