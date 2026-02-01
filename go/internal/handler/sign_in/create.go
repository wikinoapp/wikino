package sign_in

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/redirect"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	signinpages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create はログイン処理を行います (POST /sign_in)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// フォームをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	turnstileToken := r.FormValue("cf-turnstile-response")
	backURL := r.FormValue("back")

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
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		h.renderSignInForm(w, r, formErrors, csrfToken, email, backURL)
		return
	}

	// リクエストのバリデーション
	req := NewCreateRequest(email, password)
	if errors := req.Validate(ctx); errors != nil {
		h.renderSignInForm(w, r, errors, csrfToken, email, backURL)
		return
	}

	// メールアドレスでユーザーを検索
	user, err := h.userRepo.FindByEmail(ctx, email)
	if err != nil {
		slog.ErrorContext(ctx, "ユーザー検索でエラー", "error", err, "email", email)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		// セキュリティ対策: 存在しないメールアドレスでも同じエラーメッセージを表示
		formErrors := session.NewFormErrors()
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		h.renderSignInForm(w, r, formErrors, csrfToken, email, backURL)
		return
	}

	// パスワードを取得
	userPassword, err := h.userPasswordRepo.FindByUserID(ctx, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "パスワード取得でエラー", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if userPassword == nil {
		// パスワードが設定されていない場合
		formErrors := session.NewFormErrors()
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		h.renderSignInForm(w, r, formErrors, csrfToken, email, backURL)
		return
	}

	// パスワードを検証
	if !auth.VerifyPassword(userPassword.PasswordDigest, password) {
		formErrors := session.NewFormErrors()
		formErrors.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		h.renderSignInForm(w, r, formErrors, csrfToken, email, backURL)
		return
	}

	// セッションを作成
	output, err := h.createUserSessionUC.Execute(ctx, usecase.CreateUserSessionInput{
		UserID:    user.ID,
		IPAddress: clientip.GetClientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "セッション作成でエラー", "error", err, "user_id", user.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションCookieを設定
	h.sessionMgr.SetSessionCookie(w, output.Token)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_sign_in_success"))

	// リダイレクト先を決定（backパラメータが有効な場合はそのURLへ、それ以外はホームへ）
	redirectURL := redirect.GetSafeRedirectURL(backURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// renderSignInForm はログインフォームをエラー付きでレンダリングします
func (h *Handler) renderSignInForm(w http.ResponseWriter, r *http.Request, formErrors *session.FormErrors, csrfToken string, email string, backURL string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_title")

	content := signinpages.New(signinpages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       formErrors,
		BackURL:          backURL,
	})

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	w.WriteHeader(http.StatusUnprocessableEntity)

	err := layouts.Simple(meta, nil, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
