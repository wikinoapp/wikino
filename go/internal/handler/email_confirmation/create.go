package email_confirmation

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	signuppages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_up"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
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

	// リクエストのバリデーション
	req := NewCreateRequest(email, event)
	if errors := req.Validate(ctx); errors != nil {
		h.renderSignUpForm(w, r, errors, csrfToken, email)
		return
	}

	// signup イベントの場合のみ、メールアドレス重複チェックを行う
	if req.GetEvent() == model.EmailConfirmationEventSignUp {
		user, err := h.userRepo.FindByEmail(ctx, email)
		if err != nil {
			slog.ErrorContext(ctx, "ユーザー検索でエラー", "error", err, "email", email)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if user != nil {
			formErrors := session.NewFormErrors()
			formErrors.AddField("email", i18n.T(ctx, "validation_email_already_registered"))
			h.renderSignUpForm(w, r, formErrors, csrfToken, email)
			return
		}
	}

	// メール確認コードを送信
	locale := i18n.GetLocale(ctx)
	output, err := h.sendEmailConfirmationUC.Execute(ctx, usecase.SendEmailConfirmationInput{
		Email:  email,
		Event:  req.GetEvent(),
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

	err := layouts.Simple(meta, nil, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
