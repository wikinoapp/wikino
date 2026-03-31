package account

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	accountpages "github.com/wikinoapp/wikino/go/internal/templates/pages/account"
	"github.com/wikinoapp/wikino/go/internal/timezone"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Create はアカウントを作成します (POST /accounts)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// フォームをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	atname := r.FormValue("atname")
	password := r.FormValue("password")

	// セッションから email_confirmation_id を取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// ロケールを取得
	locale := i18n.GetLocale(ctx)
	var modelLocale model.Locale
	if locale == "ja" {
		modelLocale = model.LocaleJa
	} else {
		modelLocale = model.LocaleEn
	}

	// アカウントを作成
	output, err := h.createAccountUC.Execute(ctx, usecase.CreateAccountInput{
		EmailConfirmationID: emailConfirmationID,
		Atname:              atname,
		Password:            password,
		Locale:              modelLocale,
		TimeZone:            timezone.FromContext(ctx),
	})
	if err != nil {
		h.handleCreateError(w, r, err, atname)
		return
	}

	// セッションを作成
	sessionOutput, err := h.createUserSessionUC.Execute(ctx, usecase.CreateUserSessionInput{
		UserID:    output.UserID,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		slog.ErrorContext(ctx, "セッション作成に失敗", "error", err, "user_id", output.UserID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// セッションCookieを設定
	h.sessionMgr.SetSessionCookie(w, sessionOutput.Token)

	// メール確認用のCookieを削除
	h.sessionMgr.DeleteEmailConfirmationCookie(w)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_account_created"))

	// ホームにリダイレクト
	http.Redirect(w, r, "/home", http.StatusFound)
}

func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, atname string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		h.renderAccountForm(w, r, ve, atname)
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			http.Redirect(w, r, "/sign_up", http.StatusFound)
		case model.AppErrCodeConflict:
			http.Redirect(w, r, "/email_confirmation/edit", http.StatusFound)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "アカウント作成に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// renderAccountForm はアカウント作成フォームをエラー付きでレンダリングします
func (h *Handler) renderAccountForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, atname string) {
	ctx := r.Context()

	// メール確認情報を再取得（フォーム表示に必要）
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	output, err := h.getAccountNewDataUC.Execute(ctx, usecase.GetAccountNewDataInput{
		EmailConfirmationID: emailConfirmationID,
	})
	if err != nil || output == nil {
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "account_new_title")

	content := accountpages.New(accountpages.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: ve,
		Email:      output.EmailConfirmation.Email,
		Atname:     atname,
	})

	w.WriteHeader(http.StatusUnprocessableEntity)

	if err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
