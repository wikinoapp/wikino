package account

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	accountpages "github.com/wikinoapp/wikino/go/internal/templates/pages/account"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
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

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// セッションから email_confirmation_id を取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// バリデーション（形式チェック + 状態チェック）
	result := h.createValidator.Validate(ctx, validator.AccountCreateValidatorInput{
		EmailConfirmationID: emailConfirmationID,
		Atname:              atname,
		Password:            password,
	})

	// メール確認情報が見つからない場合
	if result.Err != nil && errors.Is(result.Err, validator.ErrEmailConfirmationNotFound) {
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// メール確認が未完了の場合
	if result.Err != nil && errors.Is(result.Err, validator.ErrEmailNotConfirmed) {
		http.Redirect(w, r, "/email_confirmation/edit", http.StatusFound)
		return
	}

	// フォームエラーの場合（形式バリデーションエラー、アットネーム重複）
	if result.FormErrors != nil {
		h.renderAccountForm(w, r, result.FormErrors, csrfToken, result.EmailConfirmation.Email, atname)
		return
	}

	// システムエラーの場合
	if result.Err != nil {
		slog.ErrorContext(ctx, "バリデーションに失敗", "error", result.Err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
		Email:               result.EmailConfirmation.Email,
		Atname:              atname,
		Password:            password,
		Locale:              modelLocale,
		TimeZone:            "Asia/Tokyo",
	})
	if err != nil {
		slog.ErrorContext(ctx, "アカウント作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

// renderAccountForm はアカウント作成フォームをエラー付きでレンダリングします
func (h *Handler) renderAccountForm(w http.ResponseWriter, r *http.Request, formErrors *session.FormErrors, csrfToken string, email string, atname string) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "account_new_title")

	content := accountpages.New(accountpages.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: formErrors,
		Email:      email,
		Atname:     atname,
	})

	// バリデーションエラー時は 422 Unprocessable Entity を返す
	w.WriteHeader(http.StatusUnprocessableEntity)

	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
