package settings_email

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
	settingspages "github.com/wikinoapp/wikino/go/internal/templates/pages/settings"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

const (
	// ユーザー単位で5回/時間。現在のパスワードの照合を試せる回数も、この制限で抑える
	rateLimitUserLimit  = 5
	rateLimitUserWindow = time.Hour

	// 宛先のメールアドレス単位で3回/時間 (新規登録の確認コードの送信と同じカウンターを使う)
	rateLimitEmailLimit  = 3
	rateLimitEmailWindow = time.Hour
)

// Updateは現在のパスワードと新しいメールアドレスを確かめ、新しいメールアドレスへ確認コードを
// 送ります (PATCH /settings/email)。
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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
	newEmail := r.FormValue("new_email")

	checks := []ratelimit.CheckInput{{
		Key:    ratelimit.SettingsEmailUserKey(string(user.ID)),
		Limit:  rateLimitUserLimit,
		Window: rateLimitUserWindow,
	}}
	if newEmail != "" {
		checks = append(checks, ratelimit.CheckInput{
			Key:    ratelimit.EmailKey(newEmail),
			Limit:  rateLimitEmailLimit,
			Window: rateLimitEmailWindow,
		})
	}
	for _, check := range checks {
		if err := h.limiter.Allow(ctx, check); err != nil {
			if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
				slog.WarnContext(ctx, "Rate Limit超過 (メールアドレスの変更)", "user_id", user.ID, "key", check.Key)
				ve := model.NewValidationError()
				ve.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
				h.renderError(w, r, user, ve, newEmail)
				return
			}
			slog.ErrorContext(ctx, "Rate Limitチェックでエラー", "error", err, "user_id", user.ID)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	output, err := h.createEmailUpdateConfirmationUC.Execute(ctx, usecase.CreateEmailUpdateConfirmationInput{
		UserID:   user.ID,
		NewEmail: newEmail,
		Password: r.FormValue("password"),
		Locale:   i18n.GetLocale(ctx),
	})
	if err != nil {
		if ve := model.AsValidationError(err); ve != nil {
			h.renderError(w, r, user, ve, newEmail)
			return
		}
		slog.ErrorContext(ctx, "メールアドレスの変更の確認コードの送信に失敗", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.sessionMgr.SetSettingsEmailConfirmationCookie(w, output.EmailConfirmationID)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_email_confirmation_sent"))
	http.Redirect(w, r, string(templates.EditSettingsEmailConfirmationPath()), http.StatusSeeOther)
}

// renderErrorは誤りを示してメールアドレスの変更の画面を描き直します
func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, user *model.User, ve *model.ValidationError, newEmail string) {
	h.render(w, r, user, http.StatusUnprocessableEntity, settingspages.EmailData{
		CSRFToken:    middleware.GetCSRFTokenFromContext(r.Context()),
		FormErrors:   ve,
		CurrentEmail: user.Email,
		NewEmail:     newEmail,
	})
}
