// Package oauth_device_authorizationは、OAuthのデバイス認可の要求のエンドポイント
// (`POST /oauth/device_authorization`、RFC 8628 §3.1) のHTTPハンドラーを提供します。
package oauth_device_authorization

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// basicAuthRealmは、クライアントの認証に失敗したときの `WWW-Authenticate: Basic` のrealm。
// トークンエンドポイントと同じ値にする
const basicAuthRealm = "wikino"

// Handlerはデバイス認可の要求のエンドポイントを提供します。
type Handler struct {
	createOAuthDeviceAuthorizationUC *usecase.CreateOAuthDeviceAuthorizationUsecase
}

// NewHandlerはデバイス認可の要求のハンドラーを生成します。
func NewHandler(createOAuthDeviceAuthorizationUC *usecase.CreateOAuthDeviceAuthorizationUsecase) *Handler {
	return &Handler{createOAuthDeviceAuthorizationUC: createOAuthDeviceAuthorizationUC}
}

// deviceAuthorizationParamNamesは、デバイス認可の要求のパラメーターとして受け取る名前
var deviceAuthorizationParamNames = []string{
	"client_id",
	"client_secret",
	"scope",
	"resource",
	"device_name",
}

// deviceAuthorizationResponseはデバイス認可の応答 (RFC 8628 §3.2)
type deviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

// errorResponseはデバイス認可の要求のエラーレスポンス (RFC 8628 §3.2、RFC 6749 §5.2)
type errorResponse struct {
	Error model.OAuthTokenErrorCode `json:"error"`
}

// parseDeviceAuthorizationParamsは、本文のパラメーターと `Authorization: Basic` からデバイス認可の
// 要求のパラメーターを取り出す。クエリは読まない (パラメーターは本文で送る。RFC 8628 §3.1)。
//
// デバイス認可はpublicクライアントだけが使え、publicクライアントは認証しない。クライアントが
// 認証を試みた場合はClientAuthenticatedを立て、UseCaseがクライアントの種別に応じて拒否する。
// Basicで認証するクライアントは本文のclient_idを省略できる (RFC 6749 §2.3.1) ため、
// そのクライアントIDはBasicの値から読む。読めない値は `invalid_client` としてエラーコードを返す
func parseDeviceAuthorizationParams(r *http.Request) (params usecase.OAuthDeviceAuthorizationParams, errCode model.OAuthTokenErrorCode) {
	form := r.PostForm

	var duplicated []string
	for _, name := range deviceAuthorizationParamNames {
		if len(form[name]) > 1 {
			duplicated = append(duplicated, name)
		}
	}

	params = usecase.OAuthDeviceAuthorizationParams{
		ClientID:            form.Get("client_id"),
		Scope:               form.Get("scope"),
		Resource:            form.Get("resource"),
		DeviceName:          form.Get("device_name"),
		ClientAuthenticated: form.Get("client_secret") != "",
		DuplicatedParams:    duplicated,
	}

	if r.Header.Get("Authorization") == "" {
		return params, ""
	}
	user, _, ok := r.BasicAuth()
	if !ok {
		return params, model.OAuthTokenErrorInvalidClient
	}
	clientID, err := url.QueryUnescape(user)
	if err != nil || clientID == "" {
		return params, model.OAuthTokenErrorInvalidClient
	}
	if params.ClientID != "" && params.ClientID != clientID {
		return params, model.OAuthTokenErrorInvalidRequest
	}
	params.ClientID = clientID
	params.ClientAuthenticated = true
	return params, ""
}

// writeDeviceAuthorizationは発行したコードをデバイス認可の応答として書き込む
func writeDeviceAuthorization(w http.ResponseWriter, r *http.Request, output *usecase.CreateOAuthDeviceAuthorizationOutput) {
	writeJSON(w, r, http.StatusOK, deviceAuthorizationResponse{
		DeviceCode:              output.DeviceCode,
		UserCode:                output.UserCode,
		VerificationURI:         output.VerificationURI,
		VerificationURIComplete: output.VerificationURIComplete,
		ExpiresIn:               int64(output.ExpiresIn.Seconds()),
		Interval:                int64(output.Interval.Seconds()),
	})
}

// writeErrorはデバイス認可の要求のエラーを書き込む。トークンエンドポイントと同じく、クライアントの
// 認証の失敗は `WWW-Authenticate` を付けた401、それ以外は400で返す (RFC 6749 §5.2)
func writeError(w http.ResponseWriter, r *http.Request, code model.OAuthTokenErrorCode) {
	status := http.StatusBadRequest
	if code == model.OAuthTokenErrorInvalidClient {
		status = http.StatusUnauthorized
		w.Header().Set("WWW-Authenticate", `Basic realm="`+basicAuthRealm+`"`)
	}
	writeJSON(w, r, status, errorResponse{Error: code})
}

// writeJSONはJSONの応答を書き込む。デバイスコードを含む応答をキャッシュに残さないよう、エラーを
// 含むすべての応答に `Cache-Control: no-store` を付ける
func writeJSON(w http.ResponseWriter, r *http.Request, status int, body any) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.ErrorContext(r.Context(), "デバイス認可の要求の応答の書き込みに失敗", "error", err)
	}
}
