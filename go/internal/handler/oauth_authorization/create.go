package oauth_authorization

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Createは同意画面での利用者の決定を受け、クライアントへ303で戻します (POST /oauth/authorize)。
// 許可なら認可コードを、拒否なら `error=access_denied` をリダイレクトURIに付ける。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	setResponseHeaders(w)

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		slog.WarnContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	output, err := h.createOAuthAuthorizationUC.Execute(ctx, usecase.CreateOAuthAuthorizationInput{
		UserID: user.ID,
		Params: parseOAuthAuthorizationParams(r.PostForm),
		// 許可のボタン以外からの送信は、利用者が許可を選んだものとして扱わない
		Approved: r.PostFormValue("decision") == "approve",
	})
	if err != nil {
		h.handleError(w, r, err, http.StatusSeeOther)
		return
	}

	params := url.Values{"code": {output.Code}}
	if output.State != "" {
		params.Set("state", output.State)
	}
	h.redirectToClient(w, r, output.RedirectURI, params, http.StatusSeeOther)
}
