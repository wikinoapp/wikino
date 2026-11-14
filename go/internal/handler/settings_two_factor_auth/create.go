package settings_two_factor_auth

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
	// ユーザー単位で5回/15分。6桁のコードと現在のパスワードの総当たりを防ぐため、通った試行も含めて数える
	rateLimitEnableLimit  = 5
	rateLimitEnableWindow = 15 * time.Minute

	// ユーザー単位で5回/15分。無効化とリカバリーコードの再生成で、現在のパスワードの総当たりを防ぐ
	rateLimitPasswordLimit  = 5
	rateLimitPasswordWindow = 15 * time.Minute
)

// Createは現在のパスワードと認証アプリのコードを照合して二要素認証を有効にし、
// リカバリーコードの画面へ送ります (POST /settings/two_factor_auth)。
//
// パスワードかコードを誤ったときは、登録の途中の秘密鍵のまま登録の画面を描き直します。
// 認証アプリへ登録した秘密鍵が無駄になり、QRコードを読み取り直すことにならないようにするためです。
// 登録の途中の設定が無いとき (二重送信で既に有効にした・照合のあとに別の画面で秘密鍵が差し替わった) は、
// 設定の画面へ送ります。
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
		Key:    ratelimit.SettingsTwoFactorAuthEnableUserKey(string(user.ID)),
		Limit:  rateLimitEnableLimit,
		Window: rateLimitEnableWindow,
	}
	if err := h.limiter.Allow(ctx, check); err != nil {
		if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
			slog.WarnContext(ctx, "Rate Limit超過 (二要素認証の有効化)", "user_id", user.ID, "key", check.Key)
			ve := model.NewValidationError()
			ve.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
			h.renderError(w, r, user, ve)
			return
		}
		slog.ErrorContext(ctx, "Rate Limitチェックでエラー", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err := h.enableTwoFactorAuthUC.Execute(ctx, usecase.EnableTwoFactorAuthInput{
		UserID:   user.ID,
		Password: r.FormValue("password"),
		TOTPCode: r.FormValue("totp_code"),
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.renderError(w, r, user, ve)
			return
		}
		if ae := model.AsAppError(err); ae != nil && ae.Code == model.AppErrCodeConflict {
			h.flashMgr.SetWarning(w, ae.UserMsg)
			http.Redirect(w, r, string(templates.SettingsTwoFactorAuthPath()), http.StatusSeeOther)
			return
		}
		slog.ErrorContext(ctx, "二要素認証の有効化に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	slog.InfoContext(ctx, "二要素認証を有効にしました", "user_id", user.ID)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_two_factor_auth_enabled"))
	http.Redirect(w, r, string(templates.SettingsTwoFactorAuthRecoveryCodesPath()), http.StatusSeeOther)
}

// renderErrorは、登録の途中の秘密鍵のまま誤りを示して登録の画面を422で描き直します。
// 登録の途中の設定が無くなっていたときは、設定の画面へ送ります。
func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, user *model.User, ve *model.ValidationError) {
	ctx := r.Context()

	pending, err := h.getPendingTwoFactorAuthUC.Execute(ctx, usecase.GetPendingTwoFactorAuthInput{UserID: user.ID, Email: user.Email})
	if err != nil {
		slog.ErrorContext(ctx, "登録の途中の二要素認証の設定の取得に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if pending.Setup == nil {
		http.Redirect(w, r, string(templates.SettingsTwoFactorAuthPath()), http.StatusSeeOther)
		return
	}

	h.renderNew(w, r, user, http.StatusUnprocessableEntity, pending.Setup, middleware.GetCSRFTokenFromContext(ctx), ve)
}
