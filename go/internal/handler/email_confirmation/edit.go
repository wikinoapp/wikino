package email_confirmation

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	emailconfirmationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Edit は確認コード入力フォームを表示します (GET /email_confirmation/edit)
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// セッションから email_confirmation_id を取得
	emailConfirmationID := h.sessionMgr.GetEmailConfirmationID(r)
	if emailConfirmationID == "" {
		// email_confirmation_id がない場合は /sign_up にリダイレクト
		http.Redirect(w, r, "/sign_up", http.StatusFound)
		return
	}

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// フラッシュメッセージを取得
	flash := h.flashMgr.GetFlash(w, r)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "email_confirmation_edit_title")

	// テンプレートをレンダリング
	content := emailconfirmationpages.Edit(emailconfirmationpages.EditPageData{
		CSRFToken:  csrfToken,
		FormErrors: nil,
	})
	err := layouts.Simple(meta, flash, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
