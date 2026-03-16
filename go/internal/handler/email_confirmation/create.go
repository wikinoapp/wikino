package email_confirmation

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	signuppages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_up"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

const (
	// Rate Limiting: IP単位で5回/時間
	rateLimitIPLimit  = 5
	rateLimitIPWindow = time.Hour

	// Rate Limiting: メールアドレス単位で3回/時間
	rateLimitEmailLimit  = 3
	rateLimitEmailWindow = time.Hour
)

// Create はメール確認コードを送信します (POST /email_confirmation)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// フォームをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	event := r.FormValue("event")
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
		h.renderSignUpForm(w, r, formErrors, csrfToken, email)
		return
	}

	// Rate Limiting: IP単位
	clientIP := getClientIP(r)
	if err := h.limiter.Allow(ctx, ratelimit.CheckInput{
		Key:    ratelimit.IPKey(clientIP),
		Limit:  rateLimitIPLimit,
		Window: rateLimitIPWindow,
	}); err != nil {
		if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
			slog.WarnContext(ctx, "Rate Limit超過（IP）", "ip", clientIP)
			formErrors := session.NewFormErrors()
			formErrors.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
			h.renderSignUpForm(w, r, formErrors, csrfToken, email)
			return
		}
		slog.ErrorContext(ctx, "Rate Limitチェックでエラー", "error", err, "ip", clientIP)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Rate Limiting: メールアドレス単位
	if email != "" {
		if err := h.limiter.Allow(ctx, ratelimit.CheckInput{
			Key:    ratelimit.EmailKey(email),
			Limit:  rateLimitEmailLimit,
			Window: rateLimitEmailWindow,
		}); err != nil {
			if errors.Is(err, ratelimit.ErrRateLimitExceeded) {
				slog.WarnContext(ctx, "Rate Limit超過（メールアドレス）", "email", email)
				formErrors := session.NewFormErrors()
				formErrors.AddGlobal(i18n.T(ctx, "validation_rate_limit_exceeded"))
				h.renderSignUpForm(w, r, formErrors, csrfToken, email)
				return
			}
			slog.ErrorContext(ctx, "Rate Limitチェックでエラー", "error", err, "email", email)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	// イベントタイプを変換
	eventType := parseEmailConfirmationEvent(event)

	// バリデーション（形式チェック + 状態チェック）
	result := h.createValidator.Validate(ctx, validator.EmailConfirmationCreateValidatorInput{
		Email: email,
		Event: eventType,
	})
	if result.FormErrors != nil {
		h.renderSignUpForm(w, r, result.FormErrors, csrfToken, email)
		return
	}
	if result.Err != nil {
		slog.ErrorContext(ctx, "バリデーションでエラー", "error", result.Err, "email", email)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// メール確認コードを送信
	locale := i18n.GetLocale(ctx)
	output, err := h.sendEmailConfirmationUC.Execute(ctx, usecase.SendEmailConfirmationInput{
		Email:  email,
		Event:  eventType,
		Locale: locale,
	})
	if err != nil {
		slog.ErrorContext(ctx, "メール確認コード送信に失敗", "error", err, "email", email)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションに email_confirmation_id を保存
	h.sessionMgr.SetEmailConfirmationCookie(w, output.EmailConfirmationID)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_email_confirmation_sent"))

	// 確認コード入力ページにリダイレクト
	http.Redirect(w, r, "/email_confirmation/edit", http.StatusFound)
}

// renderSignUpForm はサインアップフォームをエラー付きでレンダリングします
func (h *Handler) renderSignUpForm(w http.ResponseWriter, r *http.Request, formErrors *session.FormErrors, csrfToken string, email string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_up_title")

	content := signuppages.New(signuppages.NewPageData{
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

// parseEmailConfirmationEvent は文字列からイベントタイプを変換します
func parseEmailConfirmationEvent(event string) model.EmailConfirmationEvent {
	switch event {
	case "signup":
		return model.EmailConfirmationEventSignUp
	case "email_update":
		return model.EmailConfirmationEventEmailUpdate
	case "password_reset":
		return model.EmailConfirmationEventPasswordReset
	default:
		return model.EmailConfirmationEventSignUp
	}
}

// getClientIP はリクエストからクライアントのIPアドレスを取得します
func getClientIP(r *http.Request) string {
	// X-Forwarded-For ヘッダーを確認（プロキシ経由の場合）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// X-Real-IP ヘッダーを確認
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// RemoteAddr から取得
	return r.RemoteAddr
}
