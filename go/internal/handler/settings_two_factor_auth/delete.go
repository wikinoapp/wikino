package settings_two_factor_auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Deleteは現在のパスワードを照合して二要素認証を無効にし、設定の画面へ送ります (DELETE /settings/two_factor_auth)。
//
// パスワードを誤ったときは、設定の画面に誤りを示して描き直します。
// 二要素認証が有効でないとき (二重送信で既に無効にした・別の画面で無効にした) は、設定の画面へ送ります。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
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
			slog.WarnContext(ctx, "Rate Limit超過 (二要素認証の無効化)", "user_id", user.ID, "key", check.Key)
			ve := model.NewValidationError()
			ve.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
			h.renderDeleteError(w, r, user, ve)
			return
		}
		slog.ErrorContext(ctx, "Rate Limitチェックでエラー", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err := h.disableTwoFactorAuthUC.Execute(ctx, usecase.DisableTwoFactorAuthInput{
		UserID:   user.ID,
		Password: r.FormValue("password"),
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.renderDeleteError(w, r, user, ve)
			return
		}
		if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeConflict {
			h.flashMgr.SetWarning(w, ae.UserMsg)
			http.Redirect(w, r, string(templates.SettingsTwoFactorAuthPath()), http.StatusSeeOther)
			return
		}
		slog.ErrorContext(ctx, "二要素認証の無効化に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(ctx, "二要素認証を無効にしました", "user_id", user.ID)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_two_factor_auth_disabled"))
	http.Redirect(w, r, string(templates.SettingsTwoFactorAuthPath()), http.StatusSeeOther)
}

// renderDeleteErrorは、誤りを示して設定の画面を422で描き直します。
// 描き直す時点で二要素認証が有効でなくなっていれば、無効にするフォームは出ません。
func (h *Handler) renderDeleteError(w http.ResponseWriter, r *http.Request, user *model.User, ve *model.ValidationError) {
	ctx := r.Context()

	settings, err := h.getTwoFactorAuthSettingsUC.Execute(ctx, usecase.GetTwoFactorAuthSettingsInput{UserID: user.ID})
	if err != nil {
		slog.ErrorContext(ctx, "二要素認証の設定の取得に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.renderShow(w, r, user, http.StatusUnprocessableEntity, settings, ve)
}
