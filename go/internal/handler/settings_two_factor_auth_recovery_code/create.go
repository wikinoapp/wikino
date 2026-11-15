package settings_two_factor_auth_recovery_code

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

const (
	// ユーザー単位で5回/15分。二要素認証の無効化と同じカウンターで、現在のパスワードの総当たりを防ぐ
	rateLimitPasswordLimit  = 5
	rateLimitPasswordWindow = 15 * time.Minute
)

// Createは現在のパスワードを照合してリカバリーコードを作り直し、
// リカバリーコードの画面へ送ります (POST /settings/two_factor_auth/recovery_codes)。
//
// パスワードを誤ったときは、今のリカバリーコードのまま誤りを示して描き直します。
// 二要素認証が有効でないときは、設定の画面へ送ります。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	check := ratelimit.CheckInput{
		Key:    ratelimit.SettingsTwoFactorAuthPasswordUserKey(string(user.ID)),
		Limit:  rateLimitPasswordLimit,
		Window: rateLimitPasswordWindow,
	}
	if err := h.limiter.Allow(ctx, check); err != nil {
		if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
			slog.WarnContext(ctx, "Rate Limit超過 (リカバリーコードの再生成)", "user_id", user.ID, "key", check.Key)
			ve := model.NewValidationError()
			ve.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
			h.renderCodes(w, r, user, http.StatusUnprocessableEntity, ve)
			return
		}
		slog.ErrorContext(ctx, "Rate Limitチェックでエラー", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err := h.regenerateTwoFactorAuthRecoveryUC.Execute(ctx, usecase.RegenerateTwoFactorAuthRecoveryCodesInput{
		UserID:   user.ID,
		Password: r.FormValue("password"),
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.renderCodes(w, r, user, http.StatusUnprocessableEntity, ve)
			return
		}
		if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeConflict {
			h.flashMgr.SetWarning(w, ae.UserMsg)
			http.Redirect(w, r, string(templates.SettingsTwoFactorAuthPath()), http.StatusSeeOther)
			return
		}
		slog.ErrorContext(ctx, "リカバリーコードの再生成に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(ctx, "二要素認証のリカバリーコードを再生成しました", "user_id", user.ID)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_two_factor_auth_recovery_codes_regenerated"))
	http.Redirect(w, r, string(templates.SettingsTwoFactorAuthRecoveryCodesPath()), http.StatusSeeOther)
}
