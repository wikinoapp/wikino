package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/api"
	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler"
	"github.com/wikinoapp/wikino/go/internal/apihandler/openapi_description"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	apispace "github.com/wikinoapp/wikino/go/internal/apihandler/space"
	apitopic "github.com/wikinoapp/wikino/go/internal/apihandler/topic"
	apiuser "github.com/wikinoapp/wikino/go/internal/apihandler/user"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/api_reference"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newTestAPIHandlerは、serve.goと同じく `/api` にAPIのルーターをマウントしたハンドラーを返す
func newTestAPIHandler(t *testing.T, server apigen.StrictServerInterface) http.Handler {
	t.Helper()

	return newTestAPIHandlerWith(t, server, testAPITokenAuthenticator{}, exceededAPIRateLimiter{})
}

// testAPIAuthenticatorはnewTestAPIHandlerWithに渡すトークンの照合 (middlewareのNewAPITokenAuthが受け取るもの)
type testAPIAuthenticator interface {
	Execute(ctx context.Context, token string) (*model.APIPrincipal, error)
}

// testAPIRateLimiterはnewTestAPIHandlerWithに渡すレート制限 (middlewareのNewAPIRateLimitが受け取るもの)
type testAPIRateLimiter interface {
	Check(ctx context.Context, input ratelimit.CheckInput) (*ratelimit.CheckResult, error)
}

// newTestAPIHandlerWithは、トークンの照合とレート制限を差し替えてnewTestAPIHandlerと同じハンドラーを返す
func newTestAPIHandlerWith(t *testing.T, server apigen.StrictServerInterface, authenticator testAPIAuthenticator, limiter testAPIRateLimiter) http.Handler {
	t.Helper()

	problems := apierror.NewWriter("https://example.com")
	spec, err := apigen.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec() error = %v", err)
	}
	validator, err := middleware.NewAPIRequestValidator(spec, problems)
	if err != nil {
		t.Fatalf("NewAPIRequestValidator() error = %v", err)
	}

	tokenAuth := middleware.NewAPITokenAuth(authenticator, problems)
	rateLimit := middleware.NewAPIRateLimit(limiter, problems, middleware.APIUserRateLimitPolicy)

	r := chi.NewRouter()
	reference := api_reference.NewHandler(&config.Config{Domain: "example.com"})
	r.Mount(apiMountPath, newAPIRouter(server, reference.Show, problems, tokenAuth, rateLimit, validator))
	return r
}

// testRateLimitedAPITokenはtestAPITokenAuthenticatorが受け付ける唯一のトークン。
// このトークンの主体はexceededAPIRateLimiterによって常にレート制限を超えている
const testRateLimitedAPIToken = "wkp_rate_limited"

// testAPITokenAuthenticatorは、testRateLimitedAPITokenだけを受け付けるトークンの照合。
// トークン認証の振る舞いはmiddlewareとusecaseのテストで確かめるため、ここではルーターの構成だけを確かめる
type testAPITokenAuthenticator struct{}

func (testAPITokenAuthenticator) Execute(_ context.Context, token string) (*model.APIPrincipal, error) {
	if token != testRateLimitedAPIToken {
		return nil, nil
	}
	return &model.APIPrincipal{
		User:      &model.User{ID: "user-1"},
		TokenKind: model.APITokenKindPersonalAccessToken,
	}, nil
}

// exceededAPIRateLimiterは、常にレート制限を超えたと判定するレート制限。
// レート制限の振る舞いはmiddlewareのテストで確かめるため、ここではルーターの構成だけを確かめる
type exceededAPIRateLimiter struct{}

func (exceededAPIRateLimiter) Check(context.Context, ratelimit.CheckInput) (*ratelimit.CheckResult, error) {
	return &ratelimit.CheckResult{Allowed: false, ResetAt: time.Now().Add(time.Minute)}, nil
}

func TestAPIRouter(t *testing.T) {
	t.Parallel()

	handler := newTestAPIHandler(t, apihandler.NewServer(
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(usecase.NewListAPIPagesUsecase(nil, nil, nil), usecase.NewGetAPIPageUsecase(nil, nil, nil), nil, nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiuser.NewHandler(),
	))

	t.Run("OpenAPI記述をトークン無しで配信する", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = %d (本文: %s)", rr.Code, http.StatusOK, rr.Body.String())
		}
		if got := rr.Header().Get("Content-Type"); got != "application/yaml" {
			t.Errorf("Content-Type = %q、期待値 = %q", got, "application/yaml")
		}
		if !bytes.Equal(rr.Body.Bytes(), api.OpenAPIDescription) {
			t.Error("本文がapi/openapi.yamlと一致しない")
		}
	})

	t.Run("v1のルートにトークン認証を掛ける", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
		req.Header.Set("Authorization", "Bearer wkp_invalid")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assertProblem(t, rr, http.StatusUnauthorized)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="invalid_token"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("トークンの要るoperationにトークンが無ければ401", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assertProblem(t, rr, http.StatusUnauthorized)
		if got, want := rr.Header().Get("WWW-Authenticate"), "Bearer"; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("トークン認証の後にレート制限を掛ける", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
		req.Header.Set("Authorization", "Bearer "+testRateLimitedAPIToken)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assertProblem(t, rr, http.StatusTooManyRequests)
		if rr.Header().Get("Retry-After") == "" {
			t.Error("Retry-Afterが無い")
		}
	})

	t.Run("APIリファレンスをトークン無しで配信する", func(t *testing.T) {
		t.Parallel()

		for _, method := range []string{http.MethodGet, http.MethodHead} {
			req := httptest.NewRequest(method, "/api/reference/v1", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("%s: ステータス = %d、期待値 = %d (本文: %s)", method, rr.Code, http.StatusOK, rr.Body.String())
			}
			if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
				t.Errorf("%s: Content-Type = %q、HTMLを期待", method, got)
			}
		}
	})

	problemTests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantAllow  string
	}{
		{name: "v1のルートの無いパスは404", method: http.MethodGet, target: "/api/v1/unknown", wantStatus: http.StatusNotFound},
		{name: "未知のバージョンは404", method: http.MethodGet, target: "/api/v2/openapi.yaml", wantStatus: http.StatusNotFound},
		{name: "ルートのあるパスへの別のメソッドは405とAllow", method: http.MethodPost, target: "/api/v1/openapi.yaml", wantStatus: http.StatusMethodNotAllowed, wantAllow: "GET"},
		{name: "バージョンの無いAPIリファレンスは404", method: http.MethodGet, target: "/api/reference", wantStatus: http.StatusNotFound},
		{name: "APIリファレンスの下のパスは404", method: http.MethodGet, target: "/api/reference/v1/unknown", wantStatus: http.StatusNotFound},
		{name: "APIリファレンスへのPOSTは405とAllow", method: http.MethodPost, target: "/api/reference/v1", wantStatus: http.StatusMethodNotAllowed, wantAllow: "GET, HEAD"},
	}
	for _, tt := range problemTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, tt.target, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assertProblem(t, rr, tt.wantStatus)
			if got := rr.Header().Get("Allow"); got != tt.wantAllow {
				t.Errorf("Allow = %q、期待値 = %q", got, tt.wantAllow)
			}
		})
	}
}

// failedBodyReaderは本文の読み取り失敗を再現する
type failedBodyReader struct{}

func (failedBodyReader) Read([]byte) (int, error) {
	return 0, errors.New("本文の読み取りに失敗")
}

func TestAPIRouter_BodyLimitProblems(t *testing.T) {
	t.Parallel()

	problems := apierror.NewWriter("https://example.com")
	handler := middleware.APIResponseHeader(middleware.NewBodyLimit(problems)(newTestAPIHandler(t, apihandler.NewServer(
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(usecase.NewListAPIPagesUsecase(nil, nil, nil), usecase.NewGetAPIPageUsecase(nil, nil, nil), nil, nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiuser.NewHandler(),
	))))

	tests := []struct {
		name       string
		request    func() *http.Request
		wantStatus int
	}{
		{
			name: "Content-Lengthによる早期拒否は413",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/openapi.yaml", strings.NewReader("a"))
				req.ContentLength = middleware.DefaultMaxBodyBytes + 1
				return req
			},
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "長さ不明の本文の上限超過は413",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/openapi.yaml", strings.NewReader(strings.Repeat("a", middleware.DefaultMaxBodyBytes+1)))
				req.ContentLength = -1
				return req
			},
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "本文の読み取り失敗は400",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/openapi.yaml", nil)
				req.Body = io.NopCloser(failedBodyReader{})
				req.ContentLength = -1
				return req
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, tt.request())
			assertProblem(t, rr, tt.wantStatus)
			if got := rr.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q、期待値 = no-store", got)
			}
		})
	}
}

// errorServerは、ハンドラーが返したエラーの変換を確かめるためのStrictServerInterface
type errorServer struct {
	err error
}

func (s errorServer) GetOpenAPIDescription(context.Context, apigen.GetOpenAPIDescriptionRequestObject) (apigen.GetOpenAPIDescriptionResponseObject, error) {
	return nil, s.err
}

func (s errorServer) ListPages(context.Context, apigen.ListPagesRequestObject) (apigen.ListPagesResponseObject, error) {
	return nil, s.err
}

func (s errorServer) GetPage(context.Context, apigen.GetPageRequestObject) (apigen.GetPageResponseObject, error) {
	return nil, s.err
}

func (s errorServer) CreatePage(context.Context, apigen.CreatePageRequestObject) (apigen.CreatePageResponseObject, error) {
	return nil, s.err
}

func (s errorServer) UpdatePage(context.Context, apigen.UpdatePageRequestObject) (apigen.UpdatePageResponseObject, error) {
	return nil, s.err
}

func (s errorServer) GetSpace(context.Context, apigen.GetSpaceRequestObject) (apigen.GetSpaceResponseObject, error) {
	return nil, s.err
}

func (s errorServer) ListTopics(context.Context, apigen.ListTopicsRequestObject) (apigen.ListTopicsResponseObject, error) {
	return nil, s.err
}

func (s errorServer) GetTopic(context.Context, apigen.GetTopicRequestObject) (apigen.GetTopicResponseObject, error) {
	return nil, s.err
}

func (s errorServer) GetUser(context.Context, apigen.GetUserRequestObject) (apigen.GetUserResponseObject, error) {
	return nil, s.err
}

func TestAPIRouter_HandlerErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "未存在のAppErrorは404", err: &model.AppError{Code: model.AppErrCodeResourceNotFound}, wantStatus: http.StatusNotFound},
		{name: "ValidationErrorは422", err: model.NewValidationError(), wantStatus: http.StatusUnprocessableEntity},
		{name: "予期しないエラーは500", err: errors.New("予期しないエラー"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestAPIHandler(t, errorServer{err: tt.err})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assertProblem(t, rr, tt.wantStatus)
		})
	}
}

func TestWithRequestURL(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/spaces/alice-wiki/topics?limit=5", nil)
	var gotLink string
	handler := withRequestURL(func(ctx context.Context, _ http.ResponseWriter, _ *http.Request, _ any) (any, error) {
		gotLink = apipagination.NextLink(ctx, "next-cursor")
		return nil, nil
	}, "ListTopics")

	if _, err := handler(req.Context(), httptest.NewRecorder(), req, nil); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if want := `</api/v1/spaces/alice-wiki/topics?cursor=next-cursor&limit=5>; rel="next"`; gotLink != want {
		t.Errorf("Link = %q、期待値 = %q", gotLink, want)
	}
}

func assertProblem(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int) {
	t.Helper()

	if rr.Code != wantStatus {
		t.Errorf("ステータス = %d、期待値 = %d (本文: %s)", rr.Code, wantStatus, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != apierror.ContentType {
		t.Errorf("Content-Type = %q、期待値 = %q", got, apierror.ContentType)
	}
	var problem struct {
		Status int `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
		t.Fatalf("本文をJSONとして読めない: %v (本文: %s)", err, rr.Body.String())
	}
	if problem.Status != wantStatus {
		t.Errorf("status = %d、期待値 = %d", problem.Status, wantStatus)
	}
}
