package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
)

// stubAPIRateLimiterは、決めた結果を返し、受け取った入力を記録するレート制限
type stubAPIRateLimiter struct {
	result *ratelimit.CheckResult
	err    error
	inputs []ratelimit.CheckInput
}

func (s *stubAPIRateLimiter) Check(_ context.Context, input ratelimit.CheckInput) (*ratelimit.CheckResult, error) {
	s.inputs = append(s.inputs, input)
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func TestAPIRateLimit_Middleware(t *testing.T) {
	t.Parallel()

	policy := middleware.APIRateLimitPolicy{Name: "space_member", Limit: 100, Window: time.Minute, Key: middleware.APISpaceMemberRateLimitPolicy.Key}
	principal := &model.APIPrincipal{
		User:        &model.User{ID: "user-1"},
		SpaceMember: &model.SpaceMember{ID: "space-member-1", UserID: "user-1"},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}

	tests := []struct {
		name      string
		principal *model.APIPrincipal
		limiter   *stubAPIRateLimiter
		// wantInputは制限のチェックに渡す入力。nilならチェックしないことを確かめる
		wantInput      *ratelimit.CheckInput
		wantStatus     int
		wantPolicy     string
		wantRateLimit  string
		wantRetryAfter string
	}{
		{
			name:       "トークンの無いリクエストは制限せずに通す",
			limiter:    &stubAPIRateLimiter{},
			wantStatus: http.StatusOK,
		},
		{
			name:      "制限の範囲内なら残量をヘッダーで返して通す",
			principal: principal,
			limiter: &stubAPIRateLimiter{result: &ratelimit.CheckResult{
				Allowed:   true,
				Count:     1,
				Remaining: 99,
				ResetAt:   time.Now().Add(30 * time.Second),
			}},
			wantInput:     &ratelimit.CheckInput{Key: "api:space_member:space-member-1", Limit: 100, Window: time.Minute},
			wantStatus:    http.StatusOK,
			wantPolicy:    `"space_member";q=100;w=60`,
			wantRateLimit: `"space_member";r=99;t=30`,
		},
		{
			name:      "制限を超えたら429とRetry-Afterを返す",
			principal: principal,
			limiter: &stubAPIRateLimiter{result: &ratelimit.CheckResult{
				Allowed:   false,
				Count:     101,
				Remaining: 0,
				ResetAt:   time.Now().Add(30 * time.Second),
			}},
			wantInput:      &ratelimit.CheckInput{Key: "api:space_member:space-member-1", Limit: 100, Window: time.Minute},
			wantStatus:     http.StatusTooManyRequests,
			wantPolicy:     `"space_member";q=100;w=60`,
			wantRateLimit:  `"space_member";r=0;t=30`,
			wantRetryAfter: "30",
		},
		{
			name:       "チェックに失敗したらヘッダーを付けずに通す",
			principal:  principal,
			limiter:    &stubAPIRateLimiter{err: errors.New("DBに接続できない")},
			wantInput:  &ratelimit.CheckInput{Key: "api:space_member:space-member-1", Limit: 100, Window: time.Minute},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/spaces/example/members/me", nil)
			if tt.principal != nil {
				req = req.WithContext(middleware.SetAPIPrincipalToContext(req.Context(), tt.principal))
			}
			rr := httptest.NewRecorder()
			middleware.NewAPIRateLimit(tt.limiter, apierror.NewWriter("https://example.com"), policy).Middleware(next).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("ステータス = %d、期待値 = %d (本文: %s)", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantInput == nil {
				if len(tt.limiter.inputs) != 0 {
					t.Errorf("制限のチェック回数 = %d、期待値 = 0", len(tt.limiter.inputs))
				}
			} else if len(tt.limiter.inputs) != 1 || tt.limiter.inputs[0] != *tt.wantInput {
				t.Errorf("制限のチェックの入力 = %+v、期待値 = [%+v]", tt.limiter.inputs, *tt.wantInput)
			}

			if got := rr.Header().Get("RateLimit-Policy"); got != tt.wantPolicy {
				t.Errorf("RateLimit-Policy = %q、期待値 = %q", got, tt.wantPolicy)
			}
			if got := rr.Header().Get("RateLimit"); got != tt.wantRateLimit {
				t.Errorf("RateLimit = %q、期待値 = %q", got, tt.wantRateLimit)
			}
			if got := rr.Header().Get("Retry-After"); got != tt.wantRetryAfter {
				t.Errorf("Retry-After = %q、期待値 = %q", got, tt.wantRetryAfter)
			}

			if tt.wantStatus != http.StatusTooManyRequests {
				return
			}
			if got := rr.Header().Get("Content-Type"); got != apierror.ContentType {
				t.Errorf("Content-Type = %q、期待値 = %q", got, apierror.ContentType)
			}
			var problem apierror.Problem
			if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
				t.Fatalf("本文をProblem Detailsとして読めない: %v (本文: %s)", err, rr.Body.String())
			}
			if problem.Status != http.StatusTooManyRequests || problem.Type != "about:blank" {
				t.Errorf("Problem Details = %+v、期待値はstatusが429でtypeが about:blank", problem)
			}
		})
	}
}

func TestAPIRateLimitPolicy_Key(t *testing.T) {
	t.Parallel()

	// 同じユーザーの、2つのスペースのメンバーとしての主体
	pat := &model.APIPrincipal{
		User:        &model.User{ID: "user-1"},
		SpaceMember: &model.SpaceMember{ID: "space-member-1", SpaceID: "space-1", UserID: "user-1"},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	oauth := &model.APIPrincipal{
		User:        pat.User,
		SpaceMember: pat.SpaceMember,
		TokenKind:   model.APITokenKindOAuthAccessToken,
	}
	otherSpace := &model.APIPrincipal{
		User:        pat.User,
		SpaceMember: &model.SpaceMember{ID: "space-member-2", SpaceID: "space-2", UserID: "user-1"},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}

	tests := []struct {
		name      string
		policy    middleware.APIRateLimitPolicy
		principal *model.APIPrincipal
		wantKey   string
		wantOK    bool
	}{
		{name: "メンバー単位の制限は個人アクセストークンの持ち主のメンバーで数える", policy: middleware.APISpaceMemberRateLimitPolicy, principal: pat, wantKey: "api:space_member:space-member-1", wantOK: true},
		{name: "メンバー単位の制限は同じメンバーのOAuthのトークンを個人アクセストークンと合わせて数える", policy: middleware.APISpaceMemberRateLimitPolicy, principal: oauth, wantKey: "api:space_member:space-member-1", wantOK: true},
		{name: "メンバー単位の制限は同じユーザーでもスペースが違えば別に数える", policy: middleware.APISpaceMemberRateLimitPolicy, principal: otherSpace, wantKey: "api:space_member:space-member-2", wantOK: true},
		{name: "メンバー単位の制限はトークンの無いリクエストを数えない", policy: middleware.APISpaceMemberRateLimitPolicy, wantOK: false},
		{name: "OAuthのIPアドレス単位の制限は未信頼の転送ヘッダーを無視する", policy: middleware.OAuthIPRateLimitPolicy(nil), wantKey: "oauth:ip:192.0.2.1", wantOK: true},
		{name: "OAuthのIPアドレス単位の制限は信頼済みプロキシを経由した送信元で数える", policy: middleware.OAuthIPRateLimitPolicy([]netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}), wantKey: "oauth:ip:203.0.113.1", wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
			req.RemoteAddr = "192.0.2.1:12345"
			req.Header.Set("CF-Connecting-IP", "198.51.100.1")
			req.Header.Set("X-Forwarded-For", "198.51.100.2, 203.0.113.1")
			if tt.principal != nil {
				req = req.WithContext(middleware.SetAPIPrincipalToContext(req.Context(), tt.principal))
			}

			key, ok := tt.policy.Key(req)
			if key != tt.wantKey || ok != tt.wantOK {
				t.Errorf("Key() = (%q, %v)、期待値 = (%q, %v)", key, ok, tt.wantKey, tt.wantOK)
			}
		})
	}
}
