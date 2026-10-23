package oauth_authorization

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Createは同意画面での利用者の決定を受け、クライアントへ303で戻します (POST /oauth/authorize)。
// 許可なら利用者が選んだスコープで認可コードを、拒否なら `error=access_denied` をリダイレクト
// URIに付ける。選んだスコープが不正 (1つも無い・要求に無い) なら、同意画面を422で再表示する。
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

	params := parseOAuthAuthorizationParams(r.PostForm)
	granted := r.PostForm["granted_scopes"]
	output, err := h.createOAuthAuthorizationUC.Execute(ctx, usecase.CreateOAuthAuthorizationInput{
		UserID: user.ID,
		Params: params,
		// 許可のボタン以外からの送信は、利用者が許可を選んだものとして扱わない
		Approved:      r.PostFormValue("decision") == "approve",
		GrantedScopes: granted,
	})
	if err != nil {
		// 選んだスコープが受け付けられなければ、選び直せるよう同意画面を再表示する
		if ve := model.AsValidationError(err); ve != nil {
			h.renderNew(w, r, user, params, granted, ve)
			return
		}
		h.handleError(w, r, err, http.StatusSeeOther)
		return
	}

	response := url.Values{"code": {output.Code}}
	if output.State != "" {
		response.Set("state", output.State)
	}
	h.redirectToClient(w, r, output.RedirectURI, response, http.StatusSeeOther)
}
