package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/clientip"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
)

// apiRateLimiterは公開APIのリクエスト数を数えるインターフェース。
// ratelimit.Limiterがこのインターフェースを満たす
type apiRateLimiter interface {
	Check(ctx context.Context, input ratelimit.CheckInput) (*ratelimit.CheckResult, error)
}

// APIRateLimitPolicyはレート制限の数える単位・割り当て量・窓の長さ
type APIRateLimitPolicy struct {
	// Nameは `RateLimit-Policy`・`RateLimit` ヘッダーで示すポリシー名
	Name string
	// Limitは窓あたりに受け付けるリクエスト数
	Limit int
	// Windowは窓の長さ
	Window time.Duration
	// Keyはリクエストを数えるカウンターのキーを返す。falseを返したリクエストは制限せずに通す
	Key func(r *http.Request) (string, bool)
}

// APIUserRateLimitPolicyは、トークンで認証した公開APIのリクエストに掛けるユーザー単位のレート制限。
// 呼び出し主体をコンテキストから読むため、APITokenAuthの後に掛ける。トークンの無いリクエストは
// 制限せずに通す
var APIUserRateLimitPolicy = APIRateLimitPolicy{
	Name:   "user",
	Limit:  5000,
	Window: time.Hour,
	Key: func(r *http.Request) (string, bool) {
		principal := APIPrincipalFromContext(r.Context())
		if principal == nil {
			return "", false
		}
		return ratelimit.APIUserKey(string(principal.User.ID)), true
	},
}

// OAuthIPRateLimitPolicyは、OAuthのトークンエンドポイントとトークンの失効のエンドポイントに
// 掛けるIPアドレス単位のレート制限。クライアントの認証より前に数え、認可コード・リフレッシュ
// トークン・クライアントシークレットの総当たりと、発行の繰り返しによる負荷を抑える
func OAuthIPRateLimitPolicy(trustedProxies []netip.Prefix) APIRateLimitPolicy {
	return APIRateLimitPolicy{
		Name:   "ip",
		Limit:  60,
		Window: time.Minute,
		Key: func(r *http.Request) (string, bool) {
			return ratelimit.OAuthIPKey(clientip.GetTrustedClientIP(r, trustedProxies)), true
		},
	}
}

// APIRateLimitは、公開APIとOAuthのエンドポイントにポリシーの単位でレート制限を掛けるミドルウェア
type APIRateLimit struct {
	limiter  apiRateLimiter
	problems *apierror.Writer
	policy   APIRateLimitPolicy
}

// NewAPIRateLimitは新しいAPIRateLimitを作成する
func NewAPIRateLimit(limiter apiRateLimiter, problems *apierror.Writer, policy APIRateLimitPolicy) *APIRateLimit {
	return &APIRateLimit{
		limiter:  limiter,
		problems: problems,
		policy:   policy,
	}
}

// MiddlewareはHTTPミドルウェアを返す
func (l *APIRateLimit) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		key, ok := l.policy.Key(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		result, err := l.limiter.Check(ctx, ratelimit.CheckInput{
			Key:    key,
			Limit:  l.policy.Limit,
			Window: l.policy.Window,
		})
		if err != nil {
			// 既存のレート制限と同じく、カウンターの失敗でリクエストを止めない
			slog.ErrorContext(ctx, "APIのレート制限のチェックに失敗しました", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		resetAfter := setRateLimitHeaders(w.Header(), l.policy, result, time.Now())
		if !result.Allowed {
			slog.WarnContext(ctx, "APIのリクエストがレート制限により制限されました", "policy", l.policy.Name, "key", key)
			l.problems.TooManyRequests(w, r, resetAfter)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// setRateLimitHeadersは、IETFで標準化中の `RateLimit-Policy` / `RateLimit` ヘッダー
// (draft-ietf-httpapi-ratelimit-headers) を設定し、残量が回復するまでの時間を返す。
// 標準化が完了して形式が変わったときに差し替えられるよう、ヘッダーの生成をここにまとめる。
// 返す時間は `RateLimit` の `t` と同じ秒単位に丸め、429の `Retry-After` と値を揃える
func setRateLimitHeaders(h http.Header, policy APIRateLimitPolicy, result *ratelimit.CheckResult, now time.Time) time.Duration {
	resetSeconds := ceilSeconds(result.ResetAt.Sub(now))
	name := strconv.Quote(policy.Name)

	h.Set("RateLimit-Policy", name+";q="+strconv.Itoa(policy.Limit)+";w="+strconv.FormatInt(ceilSeconds(policy.Window), 10))
	h.Set("RateLimit", name+";r="+strconv.Itoa(result.Remaining)+";t="+strconv.FormatInt(resetSeconds, 10))

	return time.Duration(resetSeconds) * time.Second
}

// ceilSecondsはdを秒単位に切り上げる。負の値は0にする
func ceilSeconds(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64((d + time.Second - 1) / time.Second)
}
