package middleware_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// validatorTestSpecはリクエスト検証のテスト用のOpenAPI記述。
// パラメーター・リクエスト本文・securityの検証を、実際のoperationが揃う前に確かめるために使う
const validatorTestSpec = `
openapi: 3.1.1
info:
  title: test
  version: 1.0.0
servers:
  - url: /api/v1
security:
  - oauth2: []
  - personalAccessToken: []
paths:
  /things:
    get:
      operationId: listThings
      security: []
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
            minimum: 1
            maximum: 100
      responses:
        "200":
          description: ok
    post:
      operationId: createThing
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [title]
              properties:
                title:
                  type: string
                  minLength: 1
                tags:
                  type: array
                  items:
                    type: string
                    maxLength: 3
      responses:
        "201":
          description: created
  /things/{thing_number}:
    patch:
      operationId: updateThing
      security: []
      parameters:
        - name: thing_number
          in: path
          required: true
          schema:
            type: integer
      requestBody:
        required: true
        content:
          application/merge-patch+json:
            schema:
              type: object
      responses:
        "200":
          description: ok
  /secret:
    get:
      operationId: getSecret
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
            minimum: 1
      responses:
        "200":
          description: ok
  /pages:
    get:
      operationId: listPages
      security:
        - oauth2: [page:read]
        - personalAccessToken: [page:read]
      responses:
        "200":
          description: ok
    post:
      operationId: createPage
      security:
        - oauth2: [page:write]
        - personalAccessToken: [page:write]
      responses:
        "201":
          description: created
  /oauth_only:
    get:
      operationId: getOAuthOnly
      security:
        - oauth2: [page:read]
      responses:
        "200":
          description: ok
components:
  securitySchemes:
    oauth2:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: /oauth/authorize
          tokenUrl: /oauth/token
          scopes:
            page:read: ページを読む
            page:write: ページを作成・更新する
    personalAccessToken:
      type: http
      scheme: bearer
`

func newTestAPIRequestValidator(t *testing.T) *middleware.APIRequestValidator {
	t.Helper()

	spec, err := openapi3.NewLoader().LoadFromData([]byte(validatorTestSpec))
	if err != nil {
		t.Fatalf("テスト用のOpenAPI記述を読み込めない: %v", err)
	}
	v, err := middleware.NewAPIRequestValidator(spec, apierror.NewWriter("https://example.com"))
	if err != nil {
		t.Fatalf("NewAPIRequestValidator() error = %v", err)
	}
	return v
}

type validatorProblem struct {
	Type   string `json:"type"`
	Status int    `json:"status"`
	Errors []struct {
		Detail    string  `json:"detail"`
		Pointer   *string `json:"pointer"`
		Parameter *string `json:"parameter"`
	} `json:"errors"`
}

func TestAPIRequestValidator_PassesValidRequests(t *testing.T) {
	t.Parallel()

	v := newTestAPIRequestValidator(t)

	tests := []struct {
		name        string
		method      string
		target      string
		contentType string
		body        string
		wantBody    string
		// wantContentTypeはハンドラーが受け取るContent-Type。空ならcontentTypeと同じ
		wantContentType string
	}{
		{name: "パラメーターの無いGET", method: http.MethodGet, target: "/api/v1/things"},
		{name: "正しいクエリパラメーター", method: http.MethodGet, target: "/api/v1/things?limit=100"},
		{name: "正しい本文のPOST", method: http.MethodPost, target: "/api/v1/things", contentType: "application/json", body: `{"title":"a"}`, wantBody: `{"title":"a"}`},
		{name: "charset付きのContent-Type", method: http.MethodPost, target: "/api/v1/things", contentType: "application/json; charset=utf-8", body: `{"title":"a"}`, wantBody: `{"title":"a"}`},
		{name: "大文字を含むContent-Type", method: http.MethodPost, target: "/api/v1/things", contentType: "Application/JSON", body: `{"title":"a"}`, wantBody: `{"title":"a"}`, wantContentType: "application/json"},
		{name: "区切りの前後に空白のあるContent-Type", method: http.MethodPost, target: "/api/v1/things", contentType: "application/json ; Charset=UTF-8", body: `{"title":"a"}`, wantBody: `{"title":"a"}`, wantContentType: "application/json; charset=UTF-8"},
		{name: "記述に無いパスはルーターに任せる", method: http.MethodGet, target: "/api/v1/unknown"},
		{name: "記述に無いメソッドはルーターに任せる", method: http.MethodDelete, target: "/api/v1/things"},
		{name: "絶対形式のリクエストもパスで照合する", method: http.MethodGet, target: "https://example.com/api/v1/things?limit=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false
			var gotBody, gotContentType string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				gotContentType = r.Header.Get("Content-Type")
				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("本文を読めない: %v", err)
				}
				gotBody = string(b)
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()
			v.Middleware(next).ServeHTTP(rr, req)

			if !called {
				t.Fatalf("次のハンドラーが呼ばれない: ステータス = %d、本文 = %s", rr.Code, rr.Body.String())
			}
			// 検証で読み込んだ本文を、ハンドラーが読み直せる
			if gotBody != tt.wantBody {
				t.Errorf("ハンドラーが受け取った本文 = %q、期待値 = %q", gotBody, tt.wantBody)
			}
			// 生成コードも記述のキーとの一致でメディアタイプを判定するため、正規化した値を渡す
			wantContentType := tt.wantContentType
			if wantContentType == "" {
				wantContentType = tt.contentType
			}
			if gotContentType != wantContentType {
				t.Errorf("ハンドラーが受け取ったContent-Type = %q、期待値 = %q", gotContentType, wantContentType)
			}
		})
	}
}

func TestAPIRequestValidator_RejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	v := newTestAPIRequestValidator(t)
	str := func(s string) *string { return &s }

	type wantFieldError struct {
		pointer   *string
		parameter *string
	}

	tests := []struct {
		name        string
		method      string
		target      string
		contentType string
		body        string
		wantStatus  int
		wantHeader  map[string]string
		wantErrors  []wantFieldError
	}{
		{
			name:       "範囲外のクエリパラメーターは422",
			method:     http.MethodGet,
			target:     "/api/v1/things?limit=0",
			wantStatus: http.StatusUnprocessableEntity,
			wantErrors: []wantFieldError{{parameter: str("limit")}},
		},
		{
			name:       "型の違うクエリパラメーターは422",
			method:     http.MethodGet,
			target:     "/api/v1/things?limit=abc",
			wantStatus: http.StatusUnprocessableEntity,
			wantErrors: []wantFieldError{{parameter: str("limit")}},
		},
		{
			name:        "型の違うパスパラメーターは422",
			method:      http.MethodPatch,
			target:      "/api/v1/things/abc",
			contentType: "application/merge-patch+json",
			body:        `{}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrors:  []wantFieldError{{parameter: str("thing_number")}},
		},
		{
			name:        "本文のスキーマ違反はフィールドごとに422",
			method:      http.MethodPost,
			target:      "/api/v1/things",
			contentType: "application/json",
			body:        `{"title":"","tags":["ok","long"]}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrors:  []wantFieldError{{pointer: str("#/tags/1")}, {pointer: str("#/title")}},
		},
		{
			name:        "必須の項目が無い本文は422",
			method:      http.MethodPost,
			target:      "/api/v1/things",
			contentType: "application/json",
			body:        `{}`,
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrors:  []wantFieldError{{pointer: str("#")}},
		},
		{
			name:        "JSONとして読めない本文は400",
			method:      http.MethodPost,
			target:      "/api/v1/things",
			contentType: "application/json",
			body:        `{"title":`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "必須の本文が無いリクエストは422",
			method:      http.MethodPost,
			target:      "/api/v1/things",
			contentType: "application/json",
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrors:  []wantFieldError{{pointer: str("#")}},
		},
		{
			name:        "受け付けないメディアタイプのPOSTは415とAccept-Post",
			method:      http.MethodPost,
			target:      "/api/v1/things",
			contentType: "application/x-www-form-urlencoded",
			body:        "title=a",
			wantStatus:  http.StatusUnsupportedMediaType,
			wantHeader:  map[string]string{"Accept-Post": "application/json"},
		},
		{
			name:        "形式として読めないContent-Typeは415",
			method:      http.MethodPost,
			target:      "/api/v1/things",
			contentType: "application/",
			body:        `{"title":"a"}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantHeader:  map[string]string{"Accept-Post": "application/json"},
		},
		{
			name:        "受け付けないメディアタイプのPATCHは415とAccept-Patch",
			method:      http.MethodPatch,
			target:      "/api/v1/things/1",
			contentType: "application/json",
			body:        `{}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantHeader:  map[string]string{"Accept-Patch": "application/merge-patch+json"},
		},
		{
			name:       "トークンの要るoperationにトークンが無ければ401",
			method:     http.MethodGet,
			target:     "/api/v1/secret",
			wantStatus: http.StatusUnauthorized,
			wantHeader: map[string]string{"WWW-Authenticate": "Bearer"},
		},
		{
			name:       "認証の失敗はパラメーターの問題より先に返す",
			method:     http.MethodGet,
			target:     "/api/v1/secret?limit=0",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("検証に失敗したリクエストで次のハンドラーが呼ばれた")
			})

			req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			rr := httptest.NewRecorder()
			v.Middleware(next).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("ステータス = %d、期待値 = %d (本文: %s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if got := rr.Header().Get("Content-Type"); got != apierror.ContentType {
				t.Errorf("Content-Type = %q、期待値 = %q", got, apierror.ContentType)
			}
			for key, want := range tt.wantHeader {
				if got := rr.Header().Get(key); got != want {
					t.Errorf("%s = %q、期待値 = %q", key, got, want)
				}
			}

			var problem validatorProblem
			if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
				t.Fatalf("本文をJSONとして読めない: %v", err)
			}
			if problem.Status != tt.wantStatus {
				t.Errorf("status = %d、期待値 = %d", problem.Status, tt.wantStatus)
			}
			if tt.wantErrors == nil {
				return
			}

			if problem.Type != "https://example.com/api/problems/validation-failed" {
				t.Errorf("type = %q", problem.Type)
			}
			if len(problem.Errors) != len(tt.wantErrors) {
				t.Fatalf("errorsの件数 = %d、期待値 = %d (本文: %s)", len(problem.Errors), len(tt.wantErrors), rr.Body.String())
			}
			for i, want := range tt.wantErrors {
				got := problem.Errors[i]
				if got.Detail == "" {
					t.Errorf("errors[%d].detail が空", i)
				}
				// 位置はpointer・parameterで示し、説明には重ねない (kin-openapiの理由の形式が変わると失敗する)
				if strings.HasPrefix(got.Detail, "at '") || strings.HasPrefix(got.Detail, "error at ") {
					t.Errorf("errors[%d].detail = %q、位置を除いた説明を期待", i, got.Detail)
				}
				if !equalStringPtr(got.Pointer, want.pointer) {
					t.Errorf("errors[%d].pointer = %v、期待値 = %v", i, derefString(got.Pointer), derefString(want.pointer))
				}
				if !equalStringPtr(got.Parameter, want.parameter) {
					t.Errorf("errors[%d].parameter = %v、期待値 = %v", i, derefString(got.Parameter), derefString(want.parameter))
				}
			}
		})
	}
}

func TestAPIRequestValidator_ChecksTokenScopes(t *testing.T) {
	t.Parallel()

	v := newTestAPIRequestValidator(t)
	patPrincipal := func(scopes ...model.Scope) *model.APIPrincipal {
		return &model.APIPrincipal{TokenKind: model.APITokenKindPersonalAccessToken, Scopes: scopes}
	}
	oauthPrincipal := func(scopes ...model.Scope) *model.APIPrincipal {
		return &model.APIPrincipal{TokenKind: model.APITokenKindOAuthAccessToken, Scopes: scopes}
	}

	tests := []struct {
		name       string
		method     string
		target     string
		principal  *model.APIPrincipal
		wantStatus int
		// wantChallengeは401・403の `WWW-Authenticate`
		wantChallenge string
	}{
		{
			name:       "スコープを要らないoperationは、スコープの無いトークンで呼べる",
			method:     http.MethodGet,
			target:     "/api/v1/secret",
			principal:  patPrincipal(),
			wantStatus: http.StatusOK,
		},
		{
			name:       "要るスコープを持つトークンで呼べる",
			method:     http.MethodGet,
			target:     "/api/v1/pages",
			principal:  patPrincipal(model.ScopePageRead),
			wantStatus: http.StatusOK,
		},
		{
			name:       "含意を展開したスコープで呼べる",
			method:     http.MethodGet,
			target:     "/api/v1/pages",
			principal:  patPrincipal(model.ScopePageWrite, model.ScopePageRead),
			wantStatus: http.StatusOK,
		},
		{
			name:          "トークンの無いリクエストは401",
			method:        http.MethodGet,
			target:        "/api/v1/pages",
			wantStatus:    http.StatusUnauthorized,
			wantChallenge: "Bearer",
		},
		{
			name:          "スコープが足りなければ403と要るスコープ",
			method:        http.MethodPost,
			target:        "/api/v1/pages",
			principal:     patPrincipal(model.ScopePageRead, model.ScopeTopicRead),
			wantStatus:    http.StatusForbidden,
			wantChallenge: `Bearer error="insufficient_scope", scope="page:write"`,
		},
		{
			name:       "OAuthのアクセストークンはoauth2のsecuritySchemeで呼べる",
			method:     http.MethodGet,
			target:     "/api/v1/oauth_only",
			principal:  oauthPrincipal(model.ScopePageRead),
			wantStatus: http.StatusOK,
		},
		{
			name:       "OAuthのアクセストークンは個人アクセストークンと並ぶ選択肢でも呼べる",
			method:     http.MethodGet,
			target:     "/api/v1/pages",
			principal:  oauthPrincipal(model.ScopePageRead),
			wantStatus: http.StatusOK,
		},
		{
			name:          "OAuthのアクセストークンもスコープが足りなければ403と要るスコープ",
			method:        http.MethodPost,
			target:        "/api/v1/pages",
			principal:     oauthPrincipal(model.ScopePageRead),
			wantStatus:    http.StatusForbidden,
			wantChallenge: `Bearer error="insufficient_scope", scope="page:write"`,
		},
		{
			name:          "トークンの種類に合うsecuritySchemeが無ければ401",
			method:        http.MethodGet,
			target:        "/api/v1/oauth_only",
			principal:     patPrincipal(model.ScopePageRead),
			wantStatus:    http.StatusUnauthorized,
			wantChallenge: `Bearer error="invalid_token"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, tt.target, nil)
			if tt.principal != nil {
				req = req.WithContext(middleware.SetAPIPrincipalToContext(req.Context(), tt.principal))
			}
			rr := httptest.NewRecorder()
			v.Middleware(next).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("ステータス = %d、期待値 = %d (本文: %s)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if got := rr.Header().Get("WWW-Authenticate"); got != tt.wantChallenge {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, tt.wantChallenge)
			}
			if tt.wantStatus != http.StatusOK {
				if got := rr.Header().Get("Content-Type"); got != apierror.ContentType {
					t.Errorf("Content-Type = %q、期待値 = %q", got, apierror.ContentType)
				}
			}
		})
	}
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func derefString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
