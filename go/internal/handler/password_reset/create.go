package password_reset

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	passwordpages "github.com/wikinoapp/wikino/go/internal/templates/pages/password"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create はパスワードリセット申請を処理します (POST /password/reset)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// フォームをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	turnstileToken := r.FormValue("cf-turnstile-response")

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// Turnstile検証
	valid, err := h.turnstileVerifier.Verify(ctx, turnstileToken)
	if err != nil {
		slog.WarnContext(ctx, "Turnstile検証でエラー", "error", err)
	}
	if !valid {
		slog.WarnContext(ctx, "Turnstile検証に失敗", "email", email)
		formErrors := session.NewFormErrors()
		formErrors.AddGlobal(i18n.T(ctx, "validation_bot_detected"))
		h.renderForm(w, r, formErrors, csrfToken, email)
		return
	}

	// フォームバリデーション
	validator := NewCreateValidator()
	result := validator.Validate(ctx, CreateValidatorInput{
		Email: email,
	})
	if result.FormErrors != nil && result.FormErrors.HasErrors() {
		h.renderForm(w, r, result.FormErrors, csrfToken, email)
		return
	}

	// Rate Limiting: IPアドレス単位の制限（5回/時間）
	if h.limiter != nil {
		ip := clientip.GetClientIP(r)
		ipKey := fmt.Sprintf("password_reset:ip:%s", ip)
		checkResult, err := h.limiter.Check(ctx, ratelimit.CheckInput{
			Key:    ipKey,
			Limit:  5,
			Window: 1 * time.Hour,
		})
		if err != nil {
			slog.ErrorContext(ctx, "Rate Limitingチェックが失敗しました", "error", err)
		} else if !checkResult.Allowed {
			slog.WarnContext(ctx, "パスワードリセット申請がRate Limitingにより制限されました（IPアドレス単位）",
				"ip_address", ip,
			)
			formErrors := session.NewFormErrors()
			formErrors.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
			h.renderForm(w, r, formErrors, csrfToken, email)
			return
		}
	}

	// Rate Limiting: メールアドレス単位の制限（3回/時間）
	if h.limiter != nil {
		emailKey := fmt.Sprintf("password_reset:email:%s", email)
		checkResult, err := h.limiter.Check(ctx, ratelimit.CheckInput{
			Key:    emailKey,
			Limit:  3,
			Window: 1 * time.Hour,
		})
		if err != nil {
			slog.ErrorContext(ctx, "Rate Limitingチェックが失敗しました", "error", err)
		} else if !checkResult.Allowed {
			slog.WarnContext(ctx, "パスワードリセット申請がRate Limitingにより制限されました（メールアドレス単位）",
				"email", email,
				"ip_address", clientip.GetClientIP(r),
			)
			formErrors := session.NewFormErrors()
			formErrors.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
			h.renderForm(w, r, formErrors, csrfToken, email)
			return
		}
	}

	// ユーザーを検索（存在しない場合もエラーを返さない - セキュリティ対策）
	user, err := h.userRepo.FindByEmail(ctx, email)
	if err != nil {
		slog.ErrorContext(ctx, "ユーザーの検索エラー", "error", err)
	}

	// ユーザーが存在する場合のみトークンを生成
	if user != nil {
		// ロケールを取得
		locale := i18n.GetLocale(ctx)

		_, err := h.createTokenUsecase.Execute(ctx, usecase.CreatePasswordResetTokenInput{
			UserID: user.ID,
			Email:  user.Email,
			Locale: locale,
		})
		if err != nil {
			slog.ErrorContext(ctx, "パスワードリセットトークンの生成エラー", "error", err, "user_id", user.ID)
			// トークン生成に失敗してもセキュリティ上、成功ページを表示
		} else {
			slog.InfoContext(ctx, "パスワードリセット申請を受け付けました",
				"user_id", user.ID,
				"email", user.Email,
				"ip_address", clientip.GetClientIP(r),
			)
		}
	} else {
		slog.InfoContext(ctx, "パスワードリセット申請（ユーザーが存在しない）",
			"email", email,
			"ip_address", clientip.GetClientIP(r),
		)
	}

	// 常に成功ページを表示（ユーザーの存在を明かさない）
	h.renderSentPage(w, r)
}

// renderForm はパスワードリセット申請フォームをエラー付きでレンダリングします
func (h *Handler) renderForm(w http.ResponseWriter, r *http.Request, formErrors *session.FormErrors, csrfToken string, email string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_reset_title")

	content := passwordpages.Reset(passwordpages.ResetPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       formErrors,
		Email:            email,
	})

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	w.WriteHeader(http.StatusUnprocessableEntity)

	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// renderSentPage はパスワードリセットメール送信完了ページをレンダリングします
func (h *Handler) renderSentPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_reset_sent_title")

	content := passwordpages.ResetSent()

	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
