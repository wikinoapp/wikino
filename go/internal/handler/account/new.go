package account

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	accountpages "github.com/wikinoapp/wikino/go/internal/templates/pages/account"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New はアカウント作成フォームを表示します (GET /accounts/new)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// セッションから email_confirmation_id を取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		// email_confirmation_id がない場合は /sign_up にリダイレクト
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// メール確認情報を取得
	emailConfirmation, err := h.emailConfirmationRepo.FindByID(ctx, emailConfirmationID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if emailConfirmation == nil {
		// 確認情報が見つからない場合は /sign_up にリダイレクト
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// メール確認が完了していない場合は /email_confirmation/edit にリダイレクト
	if !emailConfirmation.IsSucceeded() {
		http.Redirect(w, r, "/email_confirmation/edit", http.StatusFound)
		return
	}

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// フラッシュメッセージを取得
	flash := h.flashMgr.GetFlash(w, r)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "account_new_title")

	// テンプレートをレンダリング
	content := accountpages.New(accountpages.NewPageData{
		CSRFToken:  csrfToken,
		FormErrors: nil,
		Email:      emailConfirmation.Email,
		Atname:     "",
	})
	err = layouts.Simple(meta, flash, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
