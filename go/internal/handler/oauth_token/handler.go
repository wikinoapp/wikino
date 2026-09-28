// Package oauth_tokenは、OAuthのトークンエンドポイント (`POST /oauth/token`) と、トークンの失効の
// エンドポイント (`POST /oauth/revoke`) のHTTPハンドラーを提供します。
package oauth_token

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// basicAuthRealmは、クライアントの認証に失敗したときの `WWW-Authenticate: Basic` のrealm
const basicAuthRealm = "wikino"

// HandlerはOAuthのトークンエンドポイントと、トークンの失効のエンドポイントを提供します。
type Handler struct {
	createOAuthTokenUC *usecase.CreateOAuthTokenUsecase
	revokeOAuthTokenUC *usecase.RevokeOAuthTokenUsecase
}

// NewHandlerはOAuthのトークンのハンドラーを生成します。
func NewHandler(createOAuthTokenUC *usecase.CreateOAuthTokenUsecase, revokeOAuthTokenUC *usecase.RevokeOAuthTokenUsecase) *Handler {
	return &Handler{
		createOAuthTokenUC: createOAuthTokenUC,
		revokeOAuthTokenUC: revokeOAuthTokenUC,
	}
}

// oauthTokenParamNamesは、トークン要求のパラメーターとして受け取る名前
var oauthTokenParamNames = []string{
	"grant_type",
	"code",
	"redirect_uri",
	"code_verifier",
	"refresh_token",
	"scope",
	"resource",
	"client_id",
	"client_secret",
}

// oauthTokenRevocationParamNamesは、トークンの失効の要求のパラメーターとして受け取る名前。
// `token_type_hint` は値を使わないが、ほかのパラメーターと同じく重複を拒否する
var oauthTokenRevocationParamNames = []string{
	"token",
	"token_type_hint",
	"client_id",
	"client_secret",
}

// tokenResponseはトークンレスポンス (RFC 6749 §5.1)
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// errorResponseはトークン要求・トークンの失効のエラーレスポンス (RFC 6749 §5.2、RFC 7009 §2.2.1)
type errorResponse struct {
	Error model.OAuthTokenErrorCode `json:"error"`
}

// parseOAuthTokenParamsは、本文のパラメーターと `Authorization: Basic` からトークン要求の
// パラメーターを取り出す。クエリは読まない (パラメーターは本文で送る。RFC 6749 §3.2)。
//
// クライアントの資格情報はparseClientCredentialsで取り出す。
func parseOAuthTokenParams(r *http.Request) (params usecase.OAuthTokenParams, errCode model.OAuthTokenErrorCode) {
	form := r.PostForm

	var duplicated []string
	for _, name := range oauthTokenParamNames {
		if len(form[name]) > 1 {
			duplicated = append(duplicated, name)
		}
	}

	params = usecase.OAuthTokenParams{
		GrantType:        form.Get("grant_type"),
		Code:             form.Get("code"),
		RedirectURI:      form.Get("redirect_uri"),
		CodeVerifier:     form.Get("code_verifier"),
		RefreshToken:     form.Get("refresh_token"),
		Scope:            form.Get("scope"),
		Resource:         form.Get("resource"),
		DuplicatedParams: duplicated,
	}

	clientID, clientSecret, secretProvided, errCode := parseClientCredentials(r)
	if errCode != "" {
		return params, errCode
	}
	params.ClientID = clientID
	params.ClientSecret = clientSecret
	params.ClientSecretProvided = secretProvided
	return params, ""
}

// parseOAuthTokenRevocationParamsは、本文のパラメーターと `Authorization: Basic` からトークンの
// 失効の要求 (RFC 7009 §2.1) のパラメーターを取り出す。クライアントの資格情報の扱いはトークン要求と同じ
func parseOAuthTokenRevocationParams(r *http.Request) (params usecase.OAuthTokenRevocationParams, errCode model.OAuthTokenErrorCode) {
	form := r.PostForm

	var duplicated []string
	for _, name := range oauthTokenRevocationParamNames {
		if len(form[name]) > 1 {
			duplicated = append(duplicated, name)
		}
	}

	clientID, clientSecret, secretProvided, errCode := parseClientCredentials(r)
	if errCode != "" {
		return params, errCode
	}

	return usecase.OAuthTokenRevocationParams{
		Token:                form.Get("token"),
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		ClientSecretProvided: secretProvided,
		DuplicatedParams:     duplicated,
	}, ""
}

// parseClientCredentialsは、`Authorization: Basic` (client_secret_basic) か本文の `client_id`・
// `client_secret` (client_secret_post) のどちらか一方からクライアントの資格情報を取り出す。
// 2つの方式を同時に使った要求は `invalid_request`、Basic以外の認証スキームや読めない値は
// `invalid_client` として、エラーコードを返す。secretProvidedは値が空でもシークレットを送った場合にtrue
func parseClientCredentials(r *http.Request) (clientID, clientSecret string, secretProvided bool, errCode model.OAuthTokenErrorCode) {
	form := r.PostForm

	if r.Header.Get("Authorization") == "" {
		return form.Get("client_id"), form.Get("client_secret"), form.Has("client_secret"), ""
	}

	clientID, clientSecret, ok := basicClientCredentials(r)
	if !ok {
		return "", "", false, model.OAuthTokenErrorInvalidClient
	}
	// クライアントの認証の方式は1つだけ使える (RFC 6749 §2.3)。本文のclient_idは
	// Basicと同じ値であれば受け付ける
	if form.Has("client_secret") || (form.Has("client_id") && form.Get("client_id") != clientID) {
		return "", "", false, model.OAuthTokenErrorInvalidRequest
	}
	return clientID, clientSecret, true, ""
}

// basicClientCredentialsは `Authorization: Basic` からクライアントIDとシークレットを取り出す。
// 値はbase64の前にapplication/x-www-form-urlencodedでエンコードされている (RFC 6749 §2.3.1)
func basicClientCredentials(r *http.Request) (clientID, clientSecret string, ok bool) {
	user, password, ok := r.BasicAuth()
	if !ok {
		return "", "", false
	}
	clientID, err := url.QueryUnescape(user)
	if err != nil || clientID == "" {
		return "", "", false
	}
	clientSecret, err = url.QueryUnescape(password)
	if err != nil {
		return "", "", false
	}
	return clientID, clientSecret, true
}

// writeTokenは発行したトークンをトークンレスポンスとして書き込む
func writeToken(w http.ResponseWriter, r *http.Request, output *usecase.CreateOAuthTokenOutput) {
	scopes := make([]string, len(output.Scopes))
	for i, s := range output.Scopes {
		scopes[i] = string(s)
	}

	writeJSON(w, r, http.StatusOK, tokenResponse{
		AccessToken:  output.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(output.ExpiresIn.Seconds()),
		RefreshToken: output.RefreshToken,
		Scope:        strings.Join(scopes, " "),
	})
}

// writeErrorはトークン要求・トークンの失効のエラーを書き込む。クライアントの認証の失敗は401、それ以外は400で返す。
// 401には、本文で認証を試みたクライアントにも `WWW-Authenticate` を付ける。401には認証の方式を
// 示すヘッダーが要り (RFC 9110 §15.5.2)、ヘッダーで送れる方式はBasicだけである (RFC 6749 §5.2)。
//
// `error_description` は付けない。認可エンドポイントのエラーと同じく、エラーコードで原因を
// 絞れるためである
func writeError(w http.ResponseWriter, r *http.Request, code model.OAuthTokenErrorCode) {
	status := http.StatusBadRequest
	if code == model.OAuthTokenErrorInvalidClient {
		status = http.StatusUnauthorized
		w.Header().Set("WWW-Authenticate", `Basic realm="`+basicAuthRealm+`"`)
	}
	writeJSON(w, r, status, errorResponse{Error: code})
}

// writeJSONはJSONの応答を書き込む。トークンを含む応答と、トークン要求への応答をキャッシュに
// 残さないよう、エラーを含むすべての応答に `Cache-Control: no-store` を付ける (RFC 6749 §5.1)
func writeJSON(w http.ResponseWriter, r *http.Request, status int, body any) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.ErrorContext(r.Context(), "トークン要求の応答の書き込みに失敗", "error", err)
	}
}
