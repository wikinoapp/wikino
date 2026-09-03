package sign_in_two_factor_recovery

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/redirect"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	twofactorpages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in_two_factor"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New はリカバリーコード入力フォームを表示します (GET /sign_in/two_factor/recovery/new)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read the destination handed over by the two-factor screen. It is read before the pending user
	// check so that an expired pending user cookie still sends the visitor back to sign-in with the
	// page they asked for.
	//
	// [Ja] 二要素認証画面から引き継がれた遷移先を読む。pending user cookie が期限切れのときも
	// 訪問者が求めたページを付けてサインインへ戻せるよう、pending user の確認より前に読む。
	backURL := r.URL.Query().Get("back")

	// ペンディングユーザーIDを確認
	pendingUserID := h.sessionMgr.GetPendingUserID(r)
	if pendingUserID == "" {
		// ペンディングユーザーIDがない場合はログインページにリダイレクト
		redirect.ToSignIn(w, r, backURL)
		return
	}

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_two_factor_recovery_new_title")

	// テンプレートをレンダリング
	pageData := twofactorpages.RecoveryNewPageData{
		CSRFToken:  csrfToken,
		FormErrors: nil,
		BackURL:    backURL,
	}
	content := twofactorpages.RecoveryNew(pageData)
	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
