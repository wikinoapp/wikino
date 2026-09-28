package apierror_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// problemResponseはレスポンス本文を、キーの有無まで確かめられるように読み込んだもの
type problemResponse map[string]any

func decodeProblem(t *testing.T, rr *httptest.ResponseRecorder) problemResponse {
	t.Helper()

	if got := rr.Header().Get("Content-Type"); got != apierror.ContentType {
		t.Errorf("Content-Type = %q、期待値 = %q", got, apierror.ContentType)
	}
	var body problemResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("本文をJSONとして読めない: %v (本文: %s)", err, rr.Body.String())
	}
	return body
}

func TestWriter_StatusProblems(t *testing.T) {
	t.Parallel()

	pw := apierror.NewWriter("https://example.com")

	tests := []struct {
		name       string
		method     string
		write      func(w http.ResponseWriter, r *http.Request)
		wantStatus int
		wantHeader map[string]string
	}{
		{
			name:   "トークンの無いリクエストの401",
			method: http.MethodGet,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.Unauthorized(w, r, apierror.BearerChallenge{})
			},
			wantStatus: http.StatusUnauthorized,
			wantHeader: map[string]string{
				"WWW-Authenticate": `Bearer resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/example"`,
			},
		},
		{
			name:   "無効なトークンの401",
			method: http.MethodGet,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.Unauthorized(w, r, apierror.BearerChallenge{Error: "invalid_token"})
			},
			wantStatus: http.StatusUnauthorized,
			wantHeader: map[string]string{
				"WWW-Authenticate": `Bearer error="invalid_token", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/example"`,
			},
		},
		{
			name:   "スコープ不足の403",
			method: http.MethodGet,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.Forbidden(w, r, apierror.BearerChallenge{Error: "insufficient_scope", Scope: []string{"page:read", "topic:read"}})
			},
			wantStatus: http.StatusForbidden,
			wantHeader: map[string]string{
				"WWW-Authenticate": `Bearer error="insufficient_scope", scope="page:read topic:read", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/example"`,
			},
		},
		{
			name:       "404",
			method:     http.MethodGet,
			write:      pw.NotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "405",
			method: http.MethodDelete,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.MethodNotAllowed(w, r, []string{http.MethodGet, http.MethodPatch})
			},
			wantStatus: http.StatusMethodNotAllowed,
			wantHeader: map[string]string{"Allow": "GET, PATCH"},
		},
		{
			name:       "413",
			method:     http.MethodPost,
			write:      pw.ContentTooLarge,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:   "PATCHの415はAccept-Patchを付ける",
			method: http.MethodPatch,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.UnsupportedMediaType(w, r, []string{"application/merge-patch+json"})
			},
			wantStatus: http.StatusUnsupportedMediaType,
			wantHeader: map[string]string{"Accept-Patch": "application/merge-patch+json", "Accept-Post": ""},
		},
		{
			name:   "POSTの415はAccept-Postを付ける",
			method: http.MethodPost,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.UnsupportedMediaType(w, r, []string{"application/json"})
			},
			wantStatus: http.StatusUnsupportedMediaType,
			wantHeader: map[string]string{"Accept-Post": "application/json", "Accept-Patch": ""},
		},
		{
			name:       "428",
			method:     http.MethodPatch,
			write:      pw.PreconditionRequired,
			wantStatus: http.StatusPreconditionRequired,
		},
		{
			name:   "429",
			method: http.MethodGet,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.TooManyRequests(w, r, 90*time.Second)
			},
			wantStatus: http.StatusTooManyRequests,
			wantHeader: map[string]string{"Retry-After": "90"},
		},
		{
			name:   "429のRetry-Afterは1秒未満を切り上げる",
			method: http.MethodGet,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.TooManyRequests(w, r, 1500*time.Millisecond)
			},
			wantStatus: http.StatusTooManyRequests,
			wantHeader: map[string]string{"Retry-After": "2"},
		},
		{
			name:       "500",
			method:     http.MethodGet,
			write:      pw.InternalServerError,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "503",
			method: http.MethodGet,
			write: func(w http.ResponseWriter, r *http.Request) {
				pw.ServiceUnavailable(w, r, time.Hour)
			},
			wantStatus: http.StatusServiceUnavailable,
			wantHeader: map[string]string{"Retry-After": "3600"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, "/api/v1/spaces/example/pages/1", nil)
			rr := httptest.NewRecorder()
			tt.write(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("ステータス = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
			for key, want := range tt.wantHeader {
				if got := rr.Header().Get(key); got != want {
					t.Errorf("%s = %q、期待値 = %q", key, got, want)
				}
			}

			body := decodeProblem(t, rr)
			if body["type"] != "about:blank" {
				t.Errorf("type = %v、期待値 = about:blank", body["type"])
			}
			if body["title"] != http.StatusText(tt.wantStatus) {
				t.Errorf("title = %v、期待値 = %q", body["title"], http.StatusText(tt.wantStatus))
			}
			if body["status"] != float64(tt.wantStatus) {
				t.Errorf("status = %v、期待値 = %d", body["status"], tt.wantStatus)
			}
			// 値が無い項目もキーを省かずnullにする
			if detail, ok := body["detail"]; !ok || detail != nil {
				t.Errorf("detail = %v (キーの有無 = %v)、期待値 = null", detail, ok)
			}
			if _, ok := body["errors"]; ok {
				t.Errorf("errors = %v、validation-failed以外には含めない", body["errors"])
			}
		})
	}
}

// 401・403の `resource_metadata` は、リクエストのパスのスペースの保護リソースのメタデータを指す
func TestWriter_ResourceMetadata(t *testing.T) {
	t.Parallel()

	pw := apierror.NewWriter("https://example.com/")

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "スペースのAPI",
			path: "/api/v1/spaces/example",
			want: `Bearer resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/example"`,
		},
		{
			name: "スペース配下のAPI",
			path: "/api/v1/spaces/example/pages/1",
			want: `Bearer resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/example"`,
		},
		{
			name: "識別子のエスケープを保つ",
			path: "/api/v1/spaces/a%22b/pages",
			want: `Bearer resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/a%22b"`,
		},
		{
			name: "スペースに属さないAPIには付けない",
			path: "/api/v1/user",
			want: "Bearer",
		},
		{
			name: "識別子の無いパスには付けない",
			path: "/api/v1/spaces/",
			want: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, write := range []func(http.ResponseWriter, *http.Request, apierror.BearerChallenge){pw.Unauthorized, pw.Forbidden} {
				req := httptest.NewRequest(http.MethodGet, tt.path, nil)
				rr := httptest.NewRecorder()
				write(rr, req, apierror.BearerChallenge{})

				if got := rr.Header().Get("WWW-Authenticate"); got != tt.want {
					t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, tt.want)
				}
			}
		})
	}
}

func TestWriter_BadRequest(t *testing.T) {
	t.Parallel()

	pw := apierror.NewWriter("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/example/pages", nil)
	rr := httptest.NewRecorder()
	pw.BadRequest(rr, req, "本文をJSONとして読めません")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusBadRequest)
	}
	body := decodeProblem(t, rr)
	if body["type"] != "about:blank" {
		t.Errorf("type = %v、期待値 = about:blank", body["type"])
	}
	if body["detail"] != "本文をJSONとして読めません" {
		t.Errorf("detail = %v", body["detail"])
	}
}

func TestWriter_ValidationFailed(t *testing.T) {
	t.Parallel()

	pw := apierror.NewWriter("https://example.com/")
	pointer := apierror.JSONPointer("title")
	parameter := "limit"

	req := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/example/pages", nil)
	rr := httptest.NewRecorder()
	pw.ValidationFailed(rr, req, []apierror.FieldError{
		{Detail: "タイトルを入力してください", Pointer: &pointer},
		{Detail: "limitは1以上にしてください", Parameter: &parameter},
	})

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータス = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	body := decodeProblem(t, rr)
	if body["type"] != "https://example.com/api/problems/validation-failed" {
		t.Errorf("type = %v", body["type"])
	}
	if body["title"] != "Validation Failed" {
		t.Errorf("title = %v", body["title"])
	}

	want := []any{
		map[string]any{"detail": "タイトルを入力してください", "pointer": "#/title", "parameter": nil},
		map[string]any{"detail": "limitは1以上にしてください", "pointer": nil, "parameter": "limit"},
	}
	assertJSONEqual(t, body["errors"], want)
}

func TestWriter_ValidationFailed_NilErrorsIsEmptyArray(t *testing.T) {
	t.Parallel()

	pw := apierror.NewWriter("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/example/pages", nil)
	rr := httptest.NewRecorder()
	pw.ValidationFailed(rr, req, nil)

	body := decodeProblem(t, rr)
	assertJSONEqual(t, body["errors"], []any{})
}

func TestWriter_WriteError(t *testing.T) {
	t.Parallel()

	pw := apierror.NewWriter("https://example.com")

	ve := model.NewValidationError()
	ve.AddGlobal("ページを作成できません")
	ve.AddField("title", "タイトルを入力してください")
	ve.AddField("body", "本文が長すぎます")

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantType   string
		wantDetail any
		wantErrors any
	}{
		{
			name:       "ValidationErrorは422でグローバル・フィールドの順に並べる",
			err:        ve,
			wantStatus: http.StatusUnprocessableEntity,
			wantType:   "https://example.com/api/problems/validation-failed",
			wantDetail: nil,
			wantErrors: []any{
				map[string]any{"detail": "ページを作成できません", "pointer": nil, "parameter": nil},
				map[string]any{"detail": "本文が長すぎます", "pointer": "#/body", "parameter": nil},
				map[string]any{"detail": "タイトルを入力してください", "pointer": "#/title", "parameter": nil},
			},
		},
		{
			name:       "ラップされたValidationErrorも422",
			err:        errors.Join(errors.New("外側"), ve),
			wantStatus: http.StatusUnprocessableEntity,
			wantType:   "https://example.com/api/problems/validation-failed",
		},
		{
			name:       "ParameterErrorは422でパラメーター名を付ける",
			err:        fmt.Errorf("外側: %w", &apierror.ParameterError{Parameter: "cursor", Detail: "カーソルを読めません"}),
			wantStatus: http.StatusUnprocessableEntity,
			wantType:   "https://example.com/api/problems/validation-failed",
			wantDetail: nil,
			wantErrors: []any{
				map[string]any{"detail": "カーソルを読めません", "pointer": nil, "parameter": "cursor"},
			},
		},
		{
			name:       "未存在は404",
			err:        &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: "見つかりません"},
			wantStatus: http.StatusNotFound,
			wantType:   "about:blank",
		},
		{
			name:       "権限不足もリソースの存在を秘匿して404",
			err:        &model.AppError{Code: model.AppErrCodeForbidden, UserMsg: "権限がありません"},
			wantStatus: http.StatusNotFound,
			wantType:   "about:blank",
		},
		{
			name:       "競合は409でUserMsgをdetailにする",
			err:        &model.AppError{Code: model.AppErrCodeConflict, UserMsg: "処理中です"},
			wantStatus: http.StatusConflict,
			wantType:   "about:blank",
			wantDetail: "処理中です",
		},
		{
			name:       "前提の不一致は412でUserMsgをdetailにする",
			err:        &model.AppError{Code: model.AppErrCodePreconditionFailed, UserMsg: "ページが更新されています"},
			wantStatus: http.StatusPreconditionFailed,
			wantType:   "about:blank",
			wantDetail: "ページが更新されています",
		},
		{
			name:       "If-Matchの欠落は428",
			err:        fmt.Errorf("外側: %w", apierror.ErrPreconditionRequired),
			wantStatus: http.StatusPreconditionRequired,
			wantType:   "about:blank",
			wantDetail: nil,
		},
		{
			name:       "内部エラーのAppErrorは500でUserMsgを出さない",
			err:        &model.AppError{Code: model.AppErrCodeInternal, UserMsg: "失敗しました", Internal: errors.New("内部の詳細")},
			wantStatus: http.StatusInternalServerError,
			wantType:   "about:blank",
			wantDetail: nil,
		},
		{
			name:       "素のエラーは500で内容を出さない",
			err:        errors.New("sql: connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantType:   "about:blank",
			wantDetail: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/example/pages", nil)
			rr := httptest.NewRecorder()
			pw.WriteError(rr, req, tt.err)

			if rr.Code != tt.wantStatus {
				t.Errorf("ステータス = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
			body := decodeProblem(t, rr)
			if body["type"] != tt.wantType {
				t.Errorf("type = %v、期待値 = %q", body["type"], tt.wantType)
			}
			if body["detail"] != tt.wantDetail {
				t.Errorf("detail = %v、期待値 = %v", body["detail"], tt.wantDetail)
			}
			if tt.wantErrors != nil {
				assertJSONEqual(t, body["errors"], tt.wantErrors)
			}
		})
	}
}

func TestJSONPointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tokens []string
		want   string
	}{
		{tokens: nil, want: "#"},
		{tokens: []string{"title"}, want: "#/title"},
		{tokens: []string{"items", "0", "name"}, want: "#/items/0/name"},
		{tokens: []string{"a/b", "c~d"}, want: "#/a~1b/c~0d"},
	}

	for _, tt := range tests {
		if got := apierror.JSONPointer(tt.tokens...); got != tt.want {
			t.Errorf("JSONPointer(%q) = %q、期待値 = %q", tt.tokens, got, tt.want)
		}
	}
}

func assertJSONEqual(t *testing.T, got, want any) {
	t.Helper()

	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("got をJSONにできない: %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("want をJSONにできない: %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("errors = %s、期待値 = %s", gotJSON, wantJSON)
	}
}
