package sign_in_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, true)

	req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)
	req.Header.Set("Accept-Language", "ja")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()

	if !strings.Contains(body, `action="/sign_in"`) {
		t.Error("レスポンスにログインフォームの送信先が見つからない")
	}

	if !strings.Contains(body, "test-csrf-token") {
		t.Error("レスポンスにCSRFトークンが見つからない")
	}

	if !strings.Contains(body, `name="email"`) {
		t.Error("レスポンスにメールアドレスの入力フィールドが見つからない")
	}

	if !strings.Contains(body, `name="password"`) {
		t.Error("レスポンスにパスワードの入力フィールドが見つからない")
	}

	if !strings.Contains(body, "test-site-key") {
		t.Error("レスポンスにTurnstileのサイトキーが見つからない")
	}

	if !strings.Contains(body, `name="back"`) {
		t.Error("レスポンスにbackのhiddenフィールドが見つからない")
	}
}

func TestNew_WithBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, true)

	tests := []struct {
		name       string
		backURL    string
		wantInBody string
	}{
		{
			name:       "backパラメータあり",
			backURL:    "/oauth/authorize?client_id=xxx",
			wantInBody: `name="back" value="/oauth/authorize?client_id=xxx"`,
		},
		{
			name:       "backパラメータなし",
			backURL:    "",
			wantInBody: `name="back" value=""`,
		},
		{
			name:       "日本語パスのbackパラメータ",
			backURL:    "/users/テスト",
			wantInBody: `name="back" value="/users/テスト"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			targetURL := "/sign_in"
			if tt.backURL != "" {
				targetURL = "/sign_in?back=" + tt.backURL
			}
			req := httptest.NewRequest(http.MethodGet, targetURL, nil)
			req.Header.Set("Accept-Language", "ja")

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.New(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			if !strings.Contains(body, tt.wantInBody) {
				t.Errorf("backパラメータがhiddenフィールドに含まれていません\n期待値: %s", tt.wantInBody)
			}
		})
	}
}

// ホームへ戻るロゴリンクはアイコンのみのため、アクセシブルネームは内容ではなく
// 翻訳済みのaria-labelが供給する。
func TestNew_LogoLinkHasAccessibleName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		wantLabel string
	}{
		{
			name:      "日本語",
			locale:    i18n.LangJa,
			wantLabel: "Wikinoのホーム",
		},
		{
			name:      "英語",
			locale:    i18n.LangEn,
			wantLabel: "Wikino home",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			handler := setupHandler(t, tx, true)

			req := httptest.NewRequest(http.MethodGet, "/sign_in", nil)

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			ctx = i18n.SetLocale(ctx, tt.locale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.New(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			if !strings.Contains(rr.Body.String(), `aria-label="`+tt.wantLabel+`"`) {
				t.Errorf("ロゴリンクにaria-label %qが含まれていない", tt.wantLabel)
			}
		})
	}
}

// backパラメータはログイン後の遷移先を決めるだけなので、/sign_inを遷移先ごとの正規アドレスに
// 分割してはならない。どのバリエーションもクエリ無しの /sign_inを宣言する。
func TestNew_CanonicalOmitsBackParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	handler := setupHandler(t, tx, true)

	for _, target := range []string{"/sign_in", "/sign_in?back=/s/example/pages/1"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, target, nil)
			req.Header.Set("Accept-Language", "ja")

			ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.New(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			for _, want := range []string{
				`<link rel="canonical" href="https://localhost/sign_in">`,
				`<meta property="og:url" content="https://localhost/sign_in">`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
			if strings.Contains(body, `canonical" href="https://localhost/sign_in?`) {
				t.Error("canonical URLにbackパラメータが付いている")
			}
		})
	}
}
