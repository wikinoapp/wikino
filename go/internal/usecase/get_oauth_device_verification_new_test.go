package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetOAuthDeviceVerificationNewUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	uc := NewGetOAuthDeviceVerificationNewUsecase(repository.NewFeatureFlagRepository(testutil.QueriesWithTx(tx)))
	setup := func(key string, flagEnabled bool) model.UserID {
		return setupPATMember(t, tx, key, model.SpaceRoleViewer, flagEnabled).userID
	}
	enabled := setup("devnew-enabled", true)
	disabled := setup("devnew-disabled", false)

	if err := uc.Execute(context.Background(), GetOAuthDeviceVerificationNewInput{UserID: enabled}); err != nil {
		t.Errorf("フラグが有効なユーザーのエラー = %v、期待値 = nil", err)
	}
	err := uc.Execute(context.Background(), GetOAuthDeviceVerificationNewInput{UserID: disabled})
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("フラグが無効なユーザーのエラー = %v、見つからないことを表すAppErrorを期待", err)
	}
}
