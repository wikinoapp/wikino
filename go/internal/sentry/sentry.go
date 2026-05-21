// Package sentry provides integration with the Sentry error tracking service.
// [Ja] Sentry エラー追跡サービスとの連携機能を提供するパッケージ。
package sentry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// Config holds the Sentry settings.
// [Ja] Sentry の設定を保持する。
type Config struct {
	DSN              string
	Environment      string
	Release          string
	TracesSampleRate float64
	Debug            bool
}

const maskedValue = "[FILTERED]"

// sensitiveHeaders is the list of HTTP header names to mask (lowercase, exact match).
// [Ja] マスクすべき HTTP ヘッダー名のリスト (小文字、完全一致)。
var sensitiveHeaders = []string{
	"authorization",
	"cookie",
	"x-csrf-token",
}

// sensitiveBodyKeys is the list of request body keys to mask (partial match, lowercase).
// [Ja] マスクすべきリクエストボディのキー (部分一致、小文字)。
var sensitiveBodyKeys = []string{
	"password",
	"token",
	"secret",
}

// sensitiveQueryKeys is the list of query parameter keys to mask (partial match, lowercase).
// [Ja] マスクすべきクエリパラメータのキー (部分一致、小文字)。
var sensitiveQueryKeys = []string{
	"token",
	"key",
}

// ignoredErrorPatterns lists message-level patterns that skip Sentry capture (regular expression).
// These filter out client-disconnect noise and Go runtime's normal aborts.
//
// [Ja] メッセージレベルで Sentry 送信をスキップするパターン (正規表現)。
// クライアント切断由来のノイズや Go runtime の正常な中断をフィルタする。
var ignoredErrorPatterns = []string{
	"context canceled",
	"net/http: abort Handler",
}

// Init initializes Sentry. If the DSN is empty, initialization is skipped
// and nil is returned (used when Sentry is not used in development environments).
//
// [Ja] Sentry を初期化する。DSN が空の場合は初期化をスキップし nil を返す
// (開発環境で Sentry を使用しない場合)。
func Init(cfg Config) error {
	if cfg.DSN == "" {
		slog.Info("Sentry DSNが設定されていないため、Sentryは無効化されています")
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: cfg.TracesSampleRate,
		EnableTracing:    true,
		Debug:            cfg.Debug,
		BeforeSend:       beforeSend,
		IgnoreErrors:     ignoredErrorPatterns,
	})
	if err != nil {
		return err
	}

	slog.Info("Sentryを初期化しました",
		"environment", cfg.Environment,
		"release", cfg.Release,
		"traces_sample_rate", cfg.TracesSampleRate,
	)
	return nil
}

// beforeSend filters events before sending them to Sentry.
// Errors caused by client disconnects or normal aborts are dropped,
// and sensitive data is masked from the rest.
//
// [Ja] Sentry にイベントを送信する前にフィルタリングを行う。
// クライアント切断や正常な中断由来のエラーは破棄し、それ以外はセンシティブデータをマスクする。
func beforeSend(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if hint != nil && shouldDropError(hint.OriginalException) {
		return nil
	}

	if event.Request != nil {
		filterRequestHeaders(event.Request)
		filterRequestData(event.Request)
		filterQueryString(event.Request)
	}
	return event
}

// shouldDropError reports whether the error is caused by a client disconnect
// or runtime abort. If true, the event should not be sent to Sentry.
//
// [Ja] クライアント切断・runtime 中断由来のエラーかを判定する。
// 該当する場合は Sentry に送らない。
func shouldDropError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return true
	}
	if errors.Is(err, http.ErrAbortHandler) {
		return true
	}
	return false
}

// filterRequestHeaders masks sensitive HTTP headers in the request.
// [Ja] センシティブな HTTP ヘッダーをマスクする。
func filterRequestHeaders(req *sentry.Request) {
	if req.Headers == nil {
		return
	}

	for headerName := range req.Headers {
		lowerName := strings.ToLower(headerName)
		for _, sensitive := range sensitiveHeaders {
			if lowerName == sensitive {
				req.Headers[headerName] = maskedValue
				break
			}
		}
	}
}

// filterRequestData masks sensitive fields in the request body.
// [Ja] センシティブなリクエストボディのフィールドをマスクする。
func filterRequestData(req *sentry.Request) {
	req.Data = maskFormEncodedSensitiveValues(req.Data, sensitiveBodyKeys)
}

// filterQueryString masks sensitive query parameters in the request.
// [Ja] センシティブなクエリパラメータをマスクする。
func filterQueryString(req *sentry.Request) {
	req.QueryString = maskFormEncodedSensitiveValues(req.QueryString, sensitiveQueryKeys)
}

// maskFormEncodedSensitiveValues parses a form-encoded string and returns a
// form-encoded string whose values are replaced with [FILTERED] when the key
// matches any of sensitiveKeys by token boundary. If parsing fails the entire
// input is replaced with [FILTERED] to fail safely. An empty input is returned
// unchanged, and the original encoding is preserved when no value is masked.
//
// [Ja] form-encoded 文字列をパースし、キーが sensitiveKeys のいずれかにトークン境界で
// マッチした場合に値を [FILTERED] に置換した form-encoded 文字列を返す。
// パース失敗時は安全側に倒して全体を [FILTERED] にする。空入力はそのまま返し、
// 1 件もマスクしなかった場合は元のエンコーディングを保つ。
func maskFormEncodedSensitiveValues(input string, sensitiveKeys []string) string {
	if input == "" {
		return input
	}

	values, err := url.ParseQuery(input)
	if err != nil {
		return maskedValue
	}

	filtered := false
	for key := range values {
		if matchesSensitiveTokenKey(key, sensitiveKeys) {
			values.Set(key, maskedValue)
			filtered = true
		}
	}

	if !filtered {
		return input
	}
	return values.Encode()
}

// matchesSensitiveTokenKey reports whether the key, after splitting on
// non-alphanumeric characters (`_`, `-`, `.`, etc.), contains any of
// sensitiveKeys as an exact token. Matching is case-insensitive.
//
// Examples (sensitiveKeys includes "key"):
//   - "api_key", "API-KEY", "client.key" → true  (the "key" token is present)
//   - "okeydokey", "monkey", "monkey_emoji" → false ("key" is only a substring)
//
// camelCase (e.g. "apiKey") is not split, so callers should keep keys in
// snake_case / kebab_case for this filter to work as intended.
//
// [Ja] キーを非英数字 (`_` / `-` / `.` など) で分割した結果に、sensitiveKeys の
// いずれかがトークンとして完全一致で含まれるかを判定する。大文字小文字は区別しない。
//
// 例 (sensitiveKeys に "key" を含む場合):
//   - "api_key", "API-KEY", "client.key" → true  ("key" トークンを含む)
//   - "okeydokey", "monkey", "monkey_emoji" → false ("key" は単語の一部に紛れているだけ)
//
// camelCase ("apiKey" など) は分割対象外のため、本フィルタが期待どおり動くには
// キー命名を snake_case / kebab_case で統一しておく必要がある。
func matchesSensitiveTokenKey(key string, sensitiveKeys []string) bool {
	lowerKey := strings.ToLower(key)
	for _, token := range strings.FieldsFunc(lowerKey, isSensitiveKeySeparator) {
		for _, sensitive := range sensitiveKeys {
			if token == sensitive {
				return true
			}
		}
	}
	return false
}

// isSensitiveKeySeparator reports whether the rune is treated as a token
// boundary by matchesSensitiveTokenKey. Anything outside ASCII a-z / 0-9
// (after lower-casing) acts as a separator, so `_`, `-`, `.`, and any other
// punctuation all split tokens uniformly.
//
// [Ja] matchesSensitiveTokenKey がトークン境界として扱う rune かを返す。
// 小文字化後の ASCII a-z / 0-9 以外はすべて区切り文字として扱うため、
// `_` / `-` / `.` などの記号は一様にトークン区切りになる。
func isSensitiveKeySeparator(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return false
	}
	if r >= '0' && r <= '9' {
		return false
	}
	return true
}

// Flush sends buffered events to Sentry. Call this on application shutdown.
// [Ja] バッファリングされたイベントを Sentry に送信する。アプリケーション終了時に呼び出す。
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// CaptureError sends an error to Sentry. If a Hub is attached to ctx (e.g. set by
// sentryhttp per HTTP request) the event is captured on that Hub so that request-scoped
// scope data (user, tags) is included; otherwise the global Hub is used.
//
// [Ja] エラーを Sentry に送信する。ctx に Hub が付いていれば (例: HTTP リクエストごとに
// sentryhttp が付与した Hub) その Hub 上でキャプチャされ、ユーザーやタグといった
// リクエストスコープの情報がイベントに付与される。Hub が無ければグローバル Hub にフォールバックする。
func CaptureError(ctx context.Context, err error) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureException(err)
		return
	}
	sentry.CaptureException(err)
}

// CaptureMessage sends a message to Sentry. Hub selection follows the same rule as CaptureError.
// [Ja] メッセージを Sentry に送信する。Hub の選択ルールは CaptureError と同じ。
func CaptureMessage(ctx context.Context, message string) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureMessage(message)
		return
	}
	sentry.CaptureMessage(message)
}
