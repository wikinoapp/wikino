package i18n_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
)

func TestT_BasicTranslation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		messageID string
		want      string
	}{
		{
			name:      "日本語でデフォルトタイトルを取得",
			locale:    i18n.LangJa,
			messageID: "default_title",
			want:      "Wikino",
		},
		{
			name:      "英語でデフォルトタイトルを取得",
			locale:    i18n.LangEn,
			messageID: "default_title",
			want:      "Wikino",
		},
		{
			name:      "日本語でログインタイトルを取得",
			locale:    i18n.LangJa,
			messageID: "sign_in_title",
			want:      "ログイン",
		},
		{
			name:      "英語でログインタイトルを取得",
			locale:    i18n.LangEn,
			messageID: "sign_in_title",
			want:      "Sign in",
		},
		{
			name:      "日本語でバリデーションエラーを取得",
			locale:    i18n.LangJa,
			messageID: "validation_required",
			want:      "入力してください",
		},
		{
			name:      "英語でバリデーションエラーを取得",
			locale:    i18n.LangEn,
			messageID: "validation_required",
			want:      "This field is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, tt.locale)

			got := i18n.T(ctx, tt.messageID)
			if got != tt.want {
				t.Errorf("T() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestT_MissingTranslation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	messageID := "nonexistent_message_id"
	got := i18n.T(ctx, messageID)

	// 翻訳が見つからない場合はメッセージIDを返す
	if got != messageID {
		t.Errorf("T() = %v, want %v", got, messageID)
	}
}

func TestT_WithTemplateData(t *testing.T) {
	t.Parallel()

	// テンプレートデータを使用した翻訳のテスト
	// 現在の翻訳ファイルにはプレースホルダーを含むメッセージがないため、
	// テンプレートデータを渡してもエラーにならないことを確認
	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// テンプレートデータを渡してもパニックしない
	got := i18n.T(ctx, "sign_in_title", map[string]any{"Name": "テスト"})
	if got != "ログイン" {
		t.Errorf("T() = %v, want %v", got, "ログイン")
	}
}

func TestT_NilContext(t *testing.T) {
	t.Parallel()

	// コンテキストにLocalizerがない場合
	ctx := context.Background()

	// エラーにならずにメッセージIDを返すか、翻訳を返すことを確認
	got := i18n.T(ctx, "sign_in_title")

	// デフォルト言語（日本語）で翻訳されるか、メッセージIDが返される
	if got != "ログイン" && got != "sign_in_title" {
		t.Errorf("T() = %v, want 'ログイン' or 'sign_in_title'", got)
	}
}

func TestGetLocale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(ctx context.Context) context.Context
		want  string
	}{
		{
			name: "日本語が設定されている場合",
			setup: func(ctx context.Context) context.Context {
				return i18n.SetLocale(ctx, i18n.LangJa)
			},
			want: i18n.LangJa,
		},
		{
			name: "英語が設定されている場合",
			setup: func(ctx context.Context) context.Context {
				return i18n.SetLocale(ctx, i18n.LangEn)
			},
			want: i18n.LangEn,
		},
		{
			name: "設定されていない場合はデフォルト言語",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			want: i18n.DefaultLang,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tt.setup(context.Background())
			got := i18n.GetLocale(ctx)
			if got != tt.want {
				t.Errorf("GetLocale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetLocale(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// 日本語を設定
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	if got := i18n.GetLocale(ctx); got != i18n.LangJa {
		t.Errorf("SetLocale() で日本語が設定されませんでした: got %v want %v", got, i18n.LangJa)
	}

	// 英語に変更
	ctx = i18n.SetLocale(ctx, i18n.LangEn)
	if got := i18n.GetLocale(ctx); got != i18n.LangEn {
		t.Errorf("SetLocale() で英語が設定されませんでした: got %v want %v", got, i18n.LangEn)
	}
}

func TestDetectLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		acceptLanguage string
		want           string
	}{
		{
			name:           "日本語のみ",
			acceptLanguage: "ja",
			want:           i18n.LangJa,
		},
		{
			name:           "日本語優先",
			acceptLanguage: "ja,en;q=0.9",
			want:           i18n.LangJa,
		},
		{
			name:           "英語のみ",
			acceptLanguage: "en",
			want:           i18n.LangEn,
		},
		{
			name:           "英語優先",
			acceptLanguage: "en-US,en;q=0.9",
			want:           i18n.LangEn,
		},
		{
			name:           "日本語と英語の両方（日本語優先）",
			acceptLanguage: "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7",
			want:           i18n.LangJa,
		},
		{
			name:           "サポートされていない言語",
			acceptLanguage: "fr,de",
			want:           i18n.DefaultLang,
		},
		{
			name:           "空のヘッダー",
			acceptLanguage: "",
			want:           i18n.DefaultLang,
		},
		{
			name:           "日本語地域コード付き",
			acceptLanguage: "ja-JP",
			want:           i18n.LangJa,
		},
		{
			name:           "英語地域コード付き",
			acceptLanguage: "en-GB",
			want:           i18n.LangEn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Accept-Language", tt.acceptLanguage)

			got := i18n.DetectLanguage(req)
			if got != tt.want {
				t.Errorf("DetectLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		acceptLanguage string
		wantLocale     string
	}{
		{
			name:           "日本語ヘッダー",
			acceptLanguage: "ja",
			wantLocale:     i18n.LangJa,
		},
		{
			name:           "英語ヘッダー",
			acceptLanguage: "en",
			wantLocale:     i18n.LangEn,
		},
		{
			name:           "ヘッダーなし",
			acceptLanguage: "",
			wantLocale:     i18n.DefaultLang,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedLocale string
			var capturedTranslation string

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedLocale = i18n.GetLocale(r.Context())
				capturedTranslation = i18n.T(r.Context(), "sign_in_title")
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			rr := httptest.NewRecorder()

			i18n.Middleware(testHandler).ServeHTTP(rr, req)

			if capturedLocale != tt.wantLocale {
				t.Errorf("Middleware() locale = %v, want %v", capturedLocale, tt.wantLocale)
			}

			// 翻訳が正しく動作していることを確認
			if capturedLocale == i18n.LangJa && capturedTranslation != "ログイン" {
				t.Errorf("Middleware() translation = %v, want %v", capturedTranslation, "ログイン")
			}
			if capturedLocale == i18n.LangEn && capturedTranslation != "Sign in" {
				t.Errorf("Middleware() translation = %v, want %v", capturedTranslation, "Sign in")
			}
		})
	}
}

func TestGetLocalizer(t *testing.T) {
	t.Parallel()

	t.Run("Localizerが設定されていない場合は新規作成", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, i18n.LangEn)

		localizer := i18n.GetLocalizer(ctx)
		if localizer == nil {
			t.Error("GetLocalizer() returned nil")
		}
	})

	t.Run("Localizerが設定されている場合はそれを返す", func(t *testing.T) {
		t.Parallel()

		// ミドルウェア経由でLocalizerを設定
		var capturedLocalizer1, capturedLocalizer2 interface{}

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLocalizer1 = i18n.GetLocalizer(r.Context())
			capturedLocalizer2 = i18n.GetLocalizer(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "ja")
		rr := httptest.NewRecorder()

		i18n.Middleware(testHandler).ServeHTTP(rr, req)

		// 同じLocalizerインスタンスが返されることを確認
		if capturedLocalizer1 != capturedLocalizer2 {
			t.Error("GetLocalizer() should return the same localizer instance")
		}
	})
}

func TestConstants(t *testing.T) {
	t.Parallel()

	// 定数の値を確認
	if i18n.LangJa != "ja" {
		t.Errorf("LangJa = %v, want %v", i18n.LangJa, "ja")
	}

	if i18n.LangEn != "en" {
		t.Errorf("LangEn = %v, want %v", i18n.LangEn, "en")
	}

	if i18n.DefaultLang != i18n.LangJa {
		t.Errorf("DefaultLang = %v, want %v", i18n.DefaultLang, i18n.LangJa)
	}
}

func TestT_TwoFactorAuthTranslations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		messageID string
		wantJa    string
		wantEn    string
	}{
		{
			name:      "二要素認証タイトル",
			locale:    i18n.LangJa,
			messageID: "sign_in_two_factor_title",
			wantJa:    "二要素認証の確認",
			wantEn:    "Two-factor authentication",
		},
		{
			name:      "リカバリーコードラベル",
			locale:    i18n.LangJa,
			messageID: "sign_in_two_factor_recovery_code_label",
			wantJa:    "リカバリーコード",
			wantEn:    "Recovery code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (日本語)", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, i18n.LangJa)

			got := i18n.T(ctx, tt.messageID)
			if got != tt.wantJa {
				t.Errorf("T() = %v, want %v", got, tt.wantJa)
			}
		})

		t.Run(tt.name+" (英語)", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, i18n.LangEn)

			got := i18n.T(ctx, tt.messageID)
			if got != tt.wantEn {
				t.Errorf("T() = %v, want %v", got, tt.wantEn)
			}
		})
	}
}
