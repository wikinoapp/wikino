package usecase

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// errTwoFactorAuthNotEnabledは、操作しようとした時点で二要素認証が有効でないことを表す内部エラー
var errTwoFactorAuthNotEnabled = errors.New("二要素認証が有効ではありません")

// notEnabledTwoFactorAuthErrorは、二要素認証が有効でないことを表すAppErrCodeConflictの*model.AppErrorを返す。
// ハンドラーはこのエラーで二要素認証の設定の画面へ送り、今の状態を見せる。
func notEnabledTwoFactorAuthError(ctx context.Context, userID model.UserID) *model.AppError {
	return &model.AppError{
		Code:     model.AppErrCodeConflict,
		UserMsg:  i18n.T(ctx, "flash_two_factor_auth_not_enabled"),
		Internal: errTwoFactorAuthNotEnabled,
		Metadata: map[string]string{"user_id": userID.String()},
	}
}
