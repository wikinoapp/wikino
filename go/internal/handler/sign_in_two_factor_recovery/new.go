package sign_in_two_factor_recovery

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	twofactorpages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New はリカバリーコード入力フォームを表示します (GET /sign_in/two_factor/recovery/new)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ペンディングユーザーIDを確認
	pendingUserID := h.sessionMgr.GetPendingUserID(r)
	if pendingUserID == "" {
		// ペンディングユーザーIDがない場合はログインページにリダイレクト
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_two_factor_recovery_title")

	// テンプレートをレンダリング
	pageData := twofactorpages.RecoveryNewPageData{
		CSRFToken:  csrfToken,
		FormErrors: nil,
	}
	content := twofactorpages.RecoveryNew(pageData)
	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
