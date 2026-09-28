package oauth_authorization

import (
	"net/http"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/redirect"
	oauthauthorizationpages "github.com/wikinoapp/wikino/go/internal/templates/pages/oauth_authorization"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Newは外部アプリからの認可要求を検証し、同意画面を表示します (GET /oauth/authorize)。
//
// 未ログインの利用者はサインインへ送り、サインイン後にこの認可要求へ戻す。ログイン中の
// ユーザーが連携先のスペースでアプリを許可できない場合は、同意の代わりにその理由を示し、
// クライアントへ戻るボタン (拒否として送信する) だけを出す。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	setResponseHeaders(w)

	user := middleware.UserFromContext(ctx)
	if user == nil {
		redirect.ToSignIn(w, r, r.URL.RequestURI())
		return
	}

	values, queryErr := url.ParseQuery(r.URL.RawQuery)
	params := parseOAuthAuthorizationParams(values)
	params.MalformedQuery = queryErr != nil

	output, err := h.getOAuthAuthorizationNewUC.Execute(ctx, usecase.GetOAuthAuthorizationNewInput{
		UserID: user.ID,
		Params: params,
	})
	if err != nil {
		h.handleError(w, r, err, http.StatusFound)
		return
	}

	redirectHost := ""
	if u, err := url.Parse(output.RedirectURI); err == nil {
		redirectHost = u.Host
	}

	content := oauthauthorizationpages.New(oauthauthorizationpages.NewData{
		CSRFToken:       middleware.GetCSRFTokenFromContext(ctx),
		HiddenFields:    hiddenFields(params),
		ApplicationName: output.Application.Name,
		SpaceName:       output.Space.Name,
		UserAtname:      string(user.Atname),
		Scopes:          output.Scopes,
		RedirectHost:    redirectHost,
		NotMember:       output.DeniedReason == usecase.OAuthAuthorizationDeniedNotMember,
		NoPermission:    output.DeniedReason == usecase.OAuthAuthorizationDeniedNoPermission,
	})

	h.render(w, r, "oauth_authorization_new_title", map[string]any{"ApplicationName": output.Application.Name}, content)
}

// hiddenFieldsは、同意の送信に引き継ぐ認可要求のパラメーターを並べる。送信を受けたときも
// 表示と同じ検証をやり直すため、画面に出した値をサーバー側で保持しない
func hiddenFields(params usecase.OAuthAuthorizationParams) []oauthauthorizationpages.HiddenField {
	values := []oauthauthorizationpages.HiddenField{
		{Name: "client_id", Value: params.ClientID},
		{Name: "redirect_uri", Value: params.RedirectURI},
		{Name: "response_type", Value: params.ResponseType},
		{Name: "scope", Value: params.Scope},
		{Name: "state", Value: params.State},
		{Name: "code_challenge", Value: params.CodeChallenge},
		{Name: "code_challenge_method", Value: params.CodeChallengeMethod},
		{Name: "resource", Value: params.Resource},
	}

	fields := make([]oauthauthorizationpages.HiddenField, 0, len(values))
	for _, v := range values {
		if v.Value != "" {
			fields = append(fields, v)
		}
	}
	return fields
}
