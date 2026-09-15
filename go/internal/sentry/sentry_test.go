package sentry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/getsentry/sentry-go"
)

func TestInit_EmptyDSN(t *testing.T) {
	t.Parallel()

	// DSNが空の場合は初期化をスキップしnilを返す。
	cfg := Config{
		DSN:              "",
		Environment:      "test",
		Release:          "abc123",
		TracesSampleRate: 0.5,
		Debug:            false,
	}

	if err := Init(cfg); err != nil {
		t.Errorf("DSNが空のInit() = %v、期待値 = nil", err)
	}
}

func TestInit_InvalidDSN(t *testing.T) {
	t.Parallel()

	// 無効なDSNの場合はエラーを返す。
	cfg := Config{
		DSN:              "invalid-dsn",
		Environment:      "test",
		Release:          "abc123",
		TracesSampleRate: 0.5,
		Debug:            false,
	}

	if err := Init(cfg); err == nil {
		t.Error("不正なDSNのInit()がエラーを返さなかった")
	}
}

func TestCaptureError_NilContext(t *testing.T) {
	t.Parallel()

	// コンテキストにHubが無くてもパニックしないこと。
	ctx := context.Background()
	err := errors.New("test error")

	CaptureError(ctx, err)
}

func TestCaptureMessage_NilContext(t *testing.T) {
	t.Parallel()

	// コンテキストにHubが無くてもパニックしないこと。
	ctx := context.Background()

	CaptureMessage(ctx, "test message")
}

func TestBeforeSend_FiltersRequestHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		headers  map[string]string
		expected map[string]string
	}{
		{
			name: "Authorizationヘッダーをマスク",
			headers: map[string]string{
				"Authorization": "Bearer secret-token",
				"Content-Type":  "application/json",
			},
			expected: map[string]string{
				"Authorization": "[FILTERED]",
				"Content-Type":  "application/json",
			},
		},
		{
			name: "Cookieヘッダーをマスク",
			headers: map[string]string{
				"Cookie":       "session_id=abc123",
				"Content-Type": "text/html",
			},
			expected: map[string]string{
				"Cookie":       "[FILTERED]",
				"Content-Type": "text/html",
			},
		},
		{
			name: "X-CSRF-Tokenヘッダーをマスク",
			headers: map[string]string{
				"X-CSRF-Token": "csrf-token-value",
				"Accept":       "text/html",
			},
			expected: map[string]string{
				"X-CSRF-Token": "[FILTERED]",
				"Accept":       "text/html",
			},
		},
		{
			name: "大文字小文字を区別しない",
			headers: map[string]string{
				"authorization": "Bearer secret-token",
				"COOKIE":        "session_id=abc123",
				"x-csrf-token":  "csrf-value",
			},
			expected: map[string]string{
				"authorization": "[FILTERED]",
				"COOKIE":        "[FILTERED]",
				"x-csrf-token":  "[FILTERED]",
			},
		},
		{
			name: "センシティブでないヘッダーは変更しない",
			headers: map[string]string{
				"Content-Type": "application/json",
				"User-Agent":   "Mozilla/5.0",
			},
			expected: map[string]string{
				"Content-Type": "application/json",
				"User-Agent":   "Mozilla/5.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event := &sentry.Event{
				Request: &sentry.Request{
					Headers: tt.headers,
				},
			}

			result := beforeSend(event, nil)
			if result == nil {
				t.Fatal("beforeSendは通常イベントを破棄してはならない")
			}

			for key, expectedValue := range tt.expected {
				if result.Request.Headers[key] != expectedValue {
					t.Errorf("ヘッダー%s = %q、期待値 = %q", key, result.Request.Headers[key], expectedValue)
				}
			}
		})
	}
}

func TestBeforeSend_FiltersRequestData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     string
		expected string
	}{
		{
			name:     "passwordフィールドをマスク",
			data:     "username=user&password=secret123",
			expected: "password=%5BFILTERED%5D&username=user",
		},
		{
			name:     "tokenフィールドをマスク",
			data:     "email=test@example.com&reset_token=abc123",
			expected: "email=test%40example.com&reset_token=%5BFILTERED%5D",
		},
		{
			name:     "secretフィールドをマスク",
			data:     "api_secret=mysecret&name=test",
			expected: "api_secret=%5BFILTERED%5D&name=test",
		},
		{
			name:     "複数のセンシティブフィールドをマスク",
			data:     "password=pass123&api_token=token123&client_secret=sec123",
			expected: "api_token=%5BFILTERED%5D&client_secret=%5BFILTERED%5D&password=%5BFILTERED%5D",
		},
		{
			name:     "大文字小文字を区別しない",
			data:     "PASSWORD=secret&Token=abc&SECRET=xyz",
			expected: "PASSWORD=%5BFILTERED%5D&SECRET=%5BFILTERED%5D&Token=%5BFILTERED%5D",
		},
		{
			name:     "センシティブでないフィールドは変更しない",
			data:     "username=user&email=test@example.com",
			expected: "username=user&email=test@example.com",
		},
		{
			// tokenizeは1トークンで "token" に完全一致せず、トークン境界マッチ
			// では対象外。
			name:     "tokenizeのように単語の一部に含むだけはマスクしない",
			data:     "tokenize=true&secretive=note",
			expected: "tokenize=true&secretive=note",
		},
		{
			// snake_case分割でpassword / secretトークンが含まれるためマスクされること。
			name:     "snake_caseの内側にセンシティブトークンを含む場合はマスク",
			data:     "old_password=old&account_secret=hidden",
			expected: "account_secret=%5BFILTERED%5D&old_password=%5BFILTERED%5D",
		},
		{
			name:     "空のデータ",
			data:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event := &sentry.Event{
				Request: &sentry.Request{
					Data: tt.data,
				},
			}

			result := beforeSend(event, nil)
			if result == nil {
				t.Fatal("beforeSendは通常イベントを破棄してはならない")
			}

			if result.Request.Data != tt.expected {
				t.Errorf("Data = %q、期待値 = %q", result.Request.Data, tt.expected)
			}
		})
	}
}

func TestBeforeSend_FiltersQueryString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "tokenパラメータをマスク",
			query:    "page=1&token=secret123",
			expected: "page=1&token=%5BFILTERED%5D",
		},
		{
			name:     "keyパラメータをマスク",
			query:    "id=123&api_key=mykey",
			expected: "api_key=%5BFILTERED%5D&id=123",
		},
		{
			name:     "複数のセンシティブパラメータをマスク",
			query:    "access_token=abc&secret_key=xyz&page=1",
			expected: "access_token=%5BFILTERED%5D&page=1&secret_key=%5BFILTERED%5D",
		},
		{
			name:     "大文字小文字を区別しない",
			query:    "TOKEN=secret&API_KEY=mykey",
			expected: "API_KEY=%5BFILTERED%5D&TOKEN=%5BFILTERED%5D",
		},
		{
			name:     "センシティブでないパラメータは変更しない",
			query:    "page=1&limit=10&sort=created_at",
			expected: "page=1&limit=10&sort=created_at",
		},
		{
			// monkey / okeydokeyは1トークンで "key" に完全一致せず、保持される。
			name:     "keyを単語の一部に含むだけのパラメータはマスクしない",
			query:    "monkey=banana&okeydokey=hi",
			expected: "monkey=banana&okeydokey=hi",
		},
		{
			// snake_case / kebab-caseで分割した結果にkey / tokenトークンが含まれるためマスクされること。
			name:     "kebabやsnake_caseの内側にセンシティブトークンを含む場合はマスク",
			query:    "api-key=k&csrf_token=t",
			expected: "api-key=%5BFILTERED%5D&csrf_token=%5BFILTERED%5D",
		},
		{
			name:     "空のクエリ",
			query:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event := &sentry.Event{
				Request: &sentry.Request{
					QueryString: tt.query,
				},
			}

			result := beforeSend(event, nil)
			if result == nil {
				t.Fatal("beforeSendは通常イベントを破棄してはならない")
			}

			if result.Request.QueryString != tt.expected {
				t.Errorf("QueryString = %q、期待値 = %q", result.Request.QueryString, tt.expected)
			}
		})
	}
}

func TestBeforeSend_FiltersTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tags     map[string]string
		expected map[string]string
	}{
		{
			// sentryslogはslog属性をevent.Tagsに乗せるため、メール
			// 送信失敗ログの "email" 属性はここに届く。
			name: "emailタグをマスク",
			tags: map[string]string{
				"email": "user@example.com",
			},
			expected: map[string]string{
				"email": "[FILTERED]",
			},
		},
		{
			name: "password/secret/tokenタグをマスク",
			tags: map[string]string{
				"password": "secret123",
				"secret":   "hidden",
				"token":    "abc123",
			},
			expected: map[string]string{
				"password": "[FILTERED]",
				"secret":   "[FILTERED]",
				"token":    "[FILTERED]",
			},
		},
		{
			// snake_case分割でemail / tokenトークンが含まれるためマスクされること。
			name: "snake_caseの内側にセンシティブトークンを含む場合はマスク",
			tags: map[string]string{
				"user_email": "user@example.com",
				"api_token":  "abc123",
			},
			expected: map[string]string{
				"user_email": "[FILTERED]",
				"api_token":  "[FILTERED]",
			},
		},
		{
			name: "大文字小文字を区別しない",
			tags: map[string]string{
				"EMAIL": "user@example.com",
				"Token": "abc123",
			},
			expected: map[string]string{
				"EMAIL": "[FILTERED]",
				"Token": "[FILTERED]",
			},
		},
		{
			// emailing / tokenizeは1トークンでemail / tokenに完全一致
			// せず、トークン境界マッチでは対象外。
			name: "単語の一部に含むだけのキーはマスクしない",
			tags: map[string]string{
				"emailing": "enabled",
				"tokenize": "true",
			},
			expected: map[string]string{
				"emailing": "enabled",
				"tokenize": "true",
			},
		},
		{
			// wikino_source / kindのような運用上有用なタグはそのまま
			// 通すこと。
			name: "センシティブでないタグは変更しない",
			tags: map[string]string{
				"wikino_source": "river",
				"kind":          "send_email_confirmation",
				"job.queue":     "default",
			},
			expected: map[string]string{
				"wikino_source": "river",
				"kind":          "send_email_confirmation",
				"job.queue":     "default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event := &sentry.Event{
				Tags: tt.tags,
			}

			result := beforeSend(event, nil)
			if result == nil {
				t.Fatal("beforeSendは通常イベントを破棄してはならない")
			}

			for key, expectedValue := range tt.expected {
				if result.Tags[key] != expectedValue {
					t.Errorf("タグ%s = %q、期待値 = %q", key, result.Tags[key], expectedValue)
				}
			}
		})
	}
}

func TestBeforeSend_HandlesNilTags(t *testing.T) {
	t.Parallel()

	// タグの無いイベント (素のCaptureException等) でもpanicしないこと。
	event := &sentry.Event{
		Tags: nil,
	}

	result := beforeSend(event, nil)
	if result == nil {
		t.Fatal("beforeSendは通常イベントを破棄してはならない")
	}

	if result.Tags != nil {
		t.Error("nilのTagsはnilのままであるべき")
	}
}

func TestBeforeSend_HandlesNilRequest(t *testing.T) {
	t.Parallel()

	event := &sentry.Event{
		Request: nil,
	}

	result := beforeSend(event, nil)
	if result == nil {
		t.Fatal("beforeSendは通常イベントを破棄してはならない")
	}

	if result.Request != nil {
		t.Error("nilのRequestはnilのままであるべき")
	}
}

func TestBeforeSend_HandlesInvalidData(t *testing.T) {
	t.Parallel()

	event := &sentry.Event{
		Request: &sentry.Request{
			Data: "%invalid-data%",
		},
	}

	result := beforeSend(event, nil)
	if result == nil {
		t.Fatal("beforeSendは通常イベントを破棄してはならない")
	}

	if result.Request.Data != "[FILTERED]" {
		t.Errorf("無効なデータ = %q、期待値 = [FILTERED]", result.Request.Data)
	}
}

func TestBeforeSend_HandlesInvalidQueryString(t *testing.T) {
	t.Parallel()

	event := &sentry.Event{
		Request: &sentry.Request{
			QueryString: "%invalid-query%",
		},
	}

	result := beforeSend(event, nil)
	if result == nil {
		t.Fatal("beforeSendは通常イベントを破棄してはならない")
	}

	if result.Request.QueryString != "[FILTERED]" {
		t.Errorf("無効なクエリ = %q、期待値 = [FILTERED]", result.Request.QueryString)
	}
}

func TestBeforeSend_DropsIgnorableErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "context.Canceledは破棄",
			err:  context.Canceled,
		},
		{
			name: "context.Canceledのラップは破棄",
			err:  fmt.Errorf("接続エラー: %w", context.Canceled),
		},
		{
			name: "http.ErrAbortHandlerは破棄",
			err:  http.ErrAbortHandler,
		},
		{
			name: "http.ErrAbortHandlerのラップは破棄",
			err:  fmt.Errorf("ハンドラー中断: %w", http.ErrAbortHandler),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			event := &sentry.Event{
				Request: &sentry.Request{},
			}
			hint := &sentry.EventHint{OriginalException: tt.err}

			if result := beforeSend(event, hint); result != nil {
				t.Errorf("無視対象のエラーのイベント = %+v、期待値 = nil", result)
			}
		})
	}
}

func TestBeforeSend_DropsReverseProxySourceEvents(t *testing.T) {
	t.Parallel()

	// リバースプロキシ経由の502イベントはSourceAttrKey=ReverseProxySource
	// を持つため、beforeSendで破棄する (Rails側の障害はRailsのSentry
	// プロジェクトで扱うべきため)。
	event := &sentry.Event{
		Tags:    map[string]string{SourceAttrKey: ReverseProxySource},
		Request: &sentry.Request{},
	}

	if result := beforeSend(event, nil); result != nil {
		t.Errorf("source=%sのイベントがdropされていない: %+v", ReverseProxySource, result)
	}
}

func TestBeforeSend_KeepsOtherSourceEvents(t *testing.T) {
	t.Parallel()

	// SourceAttrKeyに乗っていても、値がReverseProxySource以外のときは
	// dropせずそのままSentryに届くこと。
	event := &sentry.Event{
		Tags:    map[string]string{SourceAttrKey: "some_other_source"},
		Request: &sentry.Request{},
	}

	if result := beforeSend(event, nil); result == nil {
		t.Errorf("source=%qのイベントはdropしないこと", "some_other_source")
	}
}

func TestBeforeSend_KeepsEventsWithoutSourceTag(t *testing.T) {
	t.Parallel()

	// SourceAttrKeyが無い通常のイベントはそのまま通すこと。
	event := &sentry.Event{
		Tags:    map[string]string{"other_tag": "value"},
		Request: &sentry.Request{},
	}

	if result := beforeSend(event, nil); result == nil {
		t.Error("SourceAttrKeyの無いイベントはdropしないこと")
	}
}

func TestBeforeSend_KeepsNonIgnorableErrors(t *testing.T) {
	t.Parallel()

	event := &sentry.Event{
		Request: &sentry.Request{},
	}
	hint := &sentry.EventHint{OriginalException: errors.New("通常のエラー")}

	if result := beforeSend(event, hint); result == nil {
		t.Error("無視対象でないエラーはイベントを保持すべき")
	}
}

func TestShouldDropError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nilは破棄しない", err: nil, want: false},
		{name: "context.Canceledは破棄", err: context.Canceled, want: true},
		{name: "http.ErrAbortHandlerは破棄", err: http.ErrAbortHandler, want: true},
		{name: "ラップされたcontext.Canceledは破棄", err: fmt.Errorf("wrap: %w", context.Canceled), want: true},
		{name: "通常のエラーは破棄しない", err: errors.New("通常のエラー"), want: false},
		{name: "context.DeadlineExceededは破棄しない", err: context.DeadlineExceeded, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldDropError(tt.err); got != tt.want {
				t.Errorf("shouldDropError(%v) = %v、期待値 = %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestFilterRequestHeaders_NilHeaders(t *testing.T) {
	t.Parallel()

	req := &sentry.Request{
		Headers: nil,
	}

	filterRequestHeaders(req)

	if req.Headers != nil {
		t.Error("nilのHeadersはnilのままであるべき")
	}
}

func TestMatchesSensitiveTokenKey(t *testing.T) {
	t.Parallel()

	sensitive := []string{"token", "key", "password", "secret"}

	tests := []struct {
		name string
		key  string
		want bool
	}{
		// 完全一致。
		{name: "完全一致のtokenはマッチ", key: "token", want: true},
		{name: "完全一致のpasswordはマッチ", key: "password", want: true},

		// _ / - / . 区切りでトークン境界マッチ。
		{name: "snake_caseのapi_keyはマッチ", key: "api_key", want: true},
		{name: "kebab-caseのcsrf-tokenはマッチ", key: "csrf-token", want: true},
		{name: "ドット区切りのclient.secretはマッチ", key: "client.secret", want: true},
		{name: "大文字小文字を区別しない", key: "API_KEY", want: true},

		// 部分文字列として含むだけのケースはマッチさせない。
		{name: "keyを単語の一部に含むだけのmonkeyはマッチしない", key: "monkey", want: false},
		{name: "keyを単語の一部に含むだけのokeydokeyはマッチしない", key: "okeydokey", want: false},
		{name: "tokenを単語の一部に含むだけのtokenizeはマッチしない", key: "tokenize", want: false},
		{name: "secretを単語の一部に含むだけのsecretiveはマッチしない", key: "secretive", want: false},
		{name: "通常のフィールド名はマッチしない", key: "email", want: false},
		{name: "空文字列はマッチしない", key: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := matchesSensitiveTokenKey(tt.key, sensitive); got != tt.want {
				t.Errorf("matchesSensitiveTokenKey(%q) = %v、期待値 = %v", tt.key, got, tt.want)
			}
		})
	}
}
