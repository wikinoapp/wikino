package sign_in

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	signinpages "github.com/wikinoapp/wikino/go/internal/templates/pages/sign_in"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New はログインフォームを表示します (GET /sign_in)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// backパラメータを取得（ログイン後のリダイレクト先）
	backURL := r.URL.Query().Get("back")

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_title")
	// The back parameter only decides where to go after signing in, so it stays out of the canonical
	// URL and every /sign_in?back=... collapses to the same address.
	//
	// [Ja] back パラメータはログイン後の遷移先を決めるだけなので正規 URL には入れず、
	// /sign_in?back=... はすべて同じアドレスに集約する。
	meta.OGURL = h.cfg.AppURL() + string(templates.SignInPath())

	// テンプレートをレンダリング
	content := signinpages.New(signinpages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       nil,
		BackURL:          backURL,
	})
	err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
