package sign_in

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/redirect"
	"github.com/wikinoapp/wikino/go/internal/templates"
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

	// Turnstile検証
	valid, err := h.turnstileVerifier.Verify(ctx, turnstileToken)
	if err != nil {
		slog.WarnContext(ctx, "Turnstile検証でエラー", "error", err)
	}
	if !valid {
		slog.WarnContext(ctx, "Turnstile検証に失敗", "email", email)
		ve := model.NewValidationError()
		ve.AddGlobal(i18n.T(ctx, "validation_email_or_password_invalid"))
		h.renderSignInForm(w, r, ve, email, backURL)
		return
	}

	// UseCase を実行
	output, err := h.signInUC.Execute(ctx, usecase.CreateSignInInput{
		Email:     email,
		Password:  password,
		IPAddress: clientip.GetClientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.handleCreateError(w, r, err, email, backURL)
		return
	}

	// 二要素認証が必要な場合
	if output.TwoFactorRequired {
		h.sessionMgr.SetPendingUserCookie(w, output.UserID)
		http.Redirect(w, r, "/sign_in/two_factor/new", http.StatusFound)
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

func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, email string, backURL string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		h.renderSignInForm(w, r, ve, email, backURL)
		return
	}

	slog.ErrorContext(ctx, "ログイン処理に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderSignInForm はログインフォームをエラー付きでレンダリングします
func (h *Handler) renderSignInForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, email string, backURL string) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_new_title")
	meta.OGURL = h.cfg.AppURL() + string(templates.SignInPath())

	content := signinpages.New(signinpages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       ve,
		BackURL:          backURL,
	})

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	w.WriteHeader(http.StatusUnprocessableEntity)

	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
