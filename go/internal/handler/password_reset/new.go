package password_reset

import (
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/templates/layouts"
	passwordpages "github.com/wikinoapp/wikino/go/internal/templates/pages/password"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New はパスワードリセット申請フォームを表示します (GET /password/reset)
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// CSRFトークンを取得
	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	// ページメタ情報を設定
	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_reset_title")

	// テンプレートをレンダリング
	content := passwordpages.Reset(passwordpages.ResetPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       nil,
		Email:            "",
	})
	err := layouts.Simple(meta, nil, content).Render(ctx, w)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
