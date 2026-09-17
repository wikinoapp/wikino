// Package sentryはSentryエラー追跡サービスとの連携機能を提供する。
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

// Sentryの設定を保持する。
type Config struct {
	DSN              string
	Environment      string
	Release          string
	TracesSampleRate float64
	Debug            bool
}

const maskedValue = "[FILTERED]"

// マスクすべきHTTPヘッダー名のリスト (小文字、完全一致)。
var sensitiveHeaders = []string{
	"authorization",
	"cookie",
	"x-csrf-token",
}

// マスクすべきリクエストボディのキー (トークン境界一致、小文字)。
var sensitiveBodyKeys = []string{
	"password",
	"token",
	"secret",
}

// マスクすべきクエリパラメータのキー (トークン境界一致、小文字)。
var sensitiveQueryKeys = []string{
	"token",
	"key",
}

// マスクすべきタグのキー (トークン境界一致、小文字)。Sentryのタグには
// sentryslog経由でslog属性がそのまま乗るため、構造化属性としてログに載せた
// PII (例: メール送信失敗ログの "email" 属性) がマスクされないままSentryに
// 届いてしまう。標準エラー出力側のログには元の値が残るため、デバッグは
// そちらで行える。
var sensitiveTagKeys = []string{
	"email",
	"password",
	"secret",
	"token",
}

// メッセージレベルでSentry送信をスキップするパターン (正規表現)。
// クライアント切断由来のノイズやGo runtimeの正常な中断をフィルタする。
var ignoredErrorPatterns = []string{
	"context canceled",
	"net/http: abort Handler",
}

// Sentryを初期化する。DSNが空の場合は初期化をスキップしnilを返す
// (開発環境でSentryを使用しない場合)。
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

// Sentryにイベントを送信する前にフィルタリングを行う。
// クライアント切断や正常な中断由来のエラーは破棄し、リバースプロキシ経由の
// 502ノイズは "source" タグで識別して捨てる。残りはセンシティブデータをマスクする。
func beforeSend(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if hint != nil && shouldDropError(hint.OriginalException) {
		return nil
	}

	// リバースプロキシ経由で出された502由来のイベントは捨てる。Rails側の
	// 障害はRailsのSentryプロジェクトで扱うべきため。sentryslogはslog
	// 属性 "source" をevent.Tagsにそのまま乗せるので、タグ照合で判別できる。
	if event.Tags[SourceAttrKey] == ReverseProxySource {
		return nil
	}

	filterTags(event)

	if event.Request != nil {
		filterRequestHeaders(event.Request)
		filterRequestData(event.Request)
		filterQueryString(event.Request)
	}
	return event
}

// クライアント切断・runtime中断由来のエラーかを判定する。
// 該当する場合はSentryに送らない。
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

// センシティブなタグ (メールアドレス等) をマスクする。下のリクエスト系
// フィルタと異なり、sentryslogがslog属性をevent.Tagsに乗せて生成した
// イベントもカバーする。キー判定はボディ / クエリのフィルタと同じく
// matchesSensitiveTokenKeyによるトークン境界一致で行う。
func filterTags(event *sentry.Event) {
	for key := range event.Tags {
		if matchesSensitiveTokenKey(key, sensitiveTagKeys) {
			event.Tags[key] = maskedValue
		}
	}
}

// センシティブなHTTPヘッダーをマスクする。
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

// センシティブなリクエストボディのフィールドをマスクする。
func filterRequestData(req *sentry.Request) {
	req.Data = maskFormEncodedSensitiveValues(req.Data, sensitiveBodyKeys)
}

// センシティブなクエリパラメータをマスクする。
func filterQueryString(req *sentry.Request) {
	req.QueryString = maskFormEncodedSensitiveValues(req.QueryString, sensitiveQueryKeys)
}

// form-encoded文字列をパースし、キーがsensitiveKeysのいずれかにトークン境界で
// マッチした場合に値を [FILTERED] に置換したform-encoded文字列を返す。
// パース失敗時は安全側に倒して全体を [FILTERED] にする。空入力はそのまま返し、
// 1件もマスクしなかった場合は元のエンコーディングを保つ。
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

// キーを非英数字 (`_` / `-` / `.` など) で分割した結果に、sensitiveKeysの
// いずれかがトークンとして完全一致で含まれるかを判定する。大文字小文字は区別しない。
//
// 例 (sensitiveKeysに "key" を含む場合):
//   - "api_key", "API-KEY", "client.key" → true  ("key" トークンを含む)
//   - "okeydokey", "monkey", "monkey_emoji" → false ("key" は単語の一部に紛れているだけ)
//
// camelCase ("apiKey" など) は分割対象外のため、本フィルタが期待どおり動くには
// キー命名をsnake_case / kebab_caseで統一しておく必要がある。
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

// matchesSensitiveTokenKeyがトークン境界として扱うruneかを返す。
// 小文字化後のASCII a-z / 0-9以外はすべて区切り文字として扱うため、
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

// バッファリングされたイベントをSentryに送信する。アプリケーション終了時に呼び出す。
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// エラーをSentryに送信する。ctxにHubが付いていれば (例: HTTPリクエストごとに
// sentryhttpが付与したHub) そのHub上でキャプチャされ、ユーザーやタグといった
// リクエストスコープの情報がイベントに付与される。Hubが無ければグローバルHubにフォールバックする。
func CaptureError(ctx context.Context, err error) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureException(err)
		return
	}
	sentry.CaptureException(err)
}

// メッセージをSentryに送信する。Hubの選択ルールはCaptureErrorと同じ。
func CaptureMessage(ctx context.Context, message string) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureMessage(message)
		return
	}
	sentry.CaptureMessage(message)
}
