package oauth_authorization

import (
	"net/http"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/redirect"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
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

	h.renderNew(w, r, user, params, nil, nil)
}

// renderNewは認可要求を検証し、同意画面を描画します。formErrorsがnilなら初めての表示として
// 要求されたスコープをすべてチェックし、拒否された送信の再表示 (422) では利用者が選んだ
// スコープ (granted) をチェックしたまま戻す。再表示は同意の送信から来るため、検証に失敗して
// クライアントへ戻すときは送信への応答として303を使う
func (h *Handler) renderNew(w http.ResponseWriter, r *http.Request, user *model.User, params usecase.OAuthAuthorizationParams, granted []string, formErrors *model.ValidationError) {
	ctx := r.Context()

	status, redirectStatus := http.StatusOK, http.StatusFound
	if formErrors != nil {
		status, redirectStatus = http.StatusUnprocessableEntity, http.StatusSeeOther
	}

	output, err := h.getOAuthAuthorizationNewUC.Execute(ctx, usecase.GetOAuthAuthorizationNewInput{
		UserID: user.ID,
		Params: params,
	})
	if err != nil {
		h.handleError(w, r, err, redirectStatus)
		return
	}

	redirectHost := ""
	if u, err := url.Parse(output.RedirectURI); err == nil {
		redirectHost = u.Host
	}

	checked := output.Scopes
	if formErrors != nil {
		checked = model.StringsToScopes(granted)
	}

	content := oauthauthorizationpages.New(oauthauthorizationpages.NewData{
		CSRFToken:       middleware.GetCSRFTokenFromContext(ctx),
		HiddenFields:    hiddenFields(params),
		ApplicationName: output.Application.Name,
		SpaceName:       output.Space.Name,
		UserAtname:      string(user.Atname),
		ScopeOptions:    components.NewAPITokenScopeOptions(output.Scopes, checked, output.IneffectiveScopes),
		RedirectHost:    redirectHost,
		NotMember:       output.DeniedReason == usecase.OAuthAuthorizationDeniedNotMember,
		NoPermission:    output.DeniedReason == usecase.OAuthAuthorizationDeniedNoPermission,
		FormErrors:      formErrors,
	})

	w.WriteHeader(status)
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
