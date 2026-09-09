package redirect

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestValidateBackURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		backURL string
		want    bool
	}{
		{
			name:    "空文字は無効",
			backURL: "",
			want:    false,
		},
		{
			name:    "相対パスは有効",
			backURL: "/home",
			want:    true,
		},
		{
			name:    "ルートパスは有効",
			backURL: "/",
			want:    true,
		},
		{
			name:    "クエリパラメータ付きは有効",
			backURL: "/search?q=test",
			want:    true,
		},
		{
			name:    "日本語パスは有効",
			backURL: "/users/テスト",
			want:    true,
		},
		{
			name:    "プロトコル相対URLは無効",
			backURL: "//evil.com",
			want:    false,
		},
		{
			name:    "先頭セグメントにバックスラッシュを含むURLは無効",
			backURL: `/\evil.com`,
			want:    false,
		},
		{
			name:    "先頭セグメントに連続するバックスラッシュを含むURLは無効",
			backURL: `/\\evil.com`,
			want:    false,
		},
		{
			name:    "水平タブを含むURLは無効",
			backURL: "/\t/evil.com",
			want:    false,
		},
		{
			name:    "改行を含むURLは無効",
			backURL: "/\n/evil.com",
			want:    false,
		},
		{
			name:    "復帰を含むURLは無効",
			backURL: "/\r/evil.com",
			want:    false,
		},
		{
			name:    "絶対URLは無効",
			backURL: "https://evil.com",
			want:    false,
		},
		{
			name:    "httpの絶対URLは無効",
			backURL: "http://evil.com",
			want:    false,
		},
		{
			name:    "javascript URLは無効",
			backURL: "javascript:alert(1)",
			want:    false,
		},
		{
			name:    "相対パスは無効",
			backURL: "home",
			want:    false,
		},
		{
			name:    "複雑なパスは有効",
			backURL: "/oauth/authorize?client_id=xxx&redirect_uri=https://example.com",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidateBackURL(tt.backURL); got != tt.want {
				t.Errorf("ValidateBackURL(%q) = %v, want %v", tt.backURL, got, tt.want)
			}
		})
	}
}

func TestGetSafeRedirectURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		backURL string
		want    string
	}{
		{
			name:    "有効なURLはそのまま返す",
			backURL: "/home",
			want:    "/home",
		},
		{
			name:    "空文字はデフォルトURL",
			backURL: "",
			want:    "/",
		},
		{
			name:    "危険なURLはデフォルトURL",
			backURL: "//evil.com",
			want:    "/",
		},
		{
			name:    "バックスラッシュを含むURLはデフォルトURL",
			backURL: `/\evil.com`,
			want:    "/",
		},
		{
			name:    "水平タブを含むURLはデフォルトURL",
			backURL: "/\t/evil.com",
			want:    "/",
		},
		{
			name:    "絶対URLはデフォルトURL",
			backURL: "https://evil.com",
			want:    "/",
		},
		{
			name:    "クエリパラメータ付きはそのまま返す",
			backURL: "/search?q=test",
			want:    "/search?q=test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := GetSafeRedirectURL(tt.backURL); got != tt.want {
				t.Errorf("GetSafeRedirectURL(%q) = %v, want %v", tt.backURL, got, tt.want)
			}
		})
	}
}

// TestToSignIn verifies which back parameters survive the redirect to the sign-in page. A safe
// destination is carried so that signing in again reaches it, and anything else is dropped rather
// than shown in the URL of the sign-in screen.
//
// [Ja] TestToSignIn は、サインインページへのリダイレクトでどの back が引き継がれるかを検証する。
// 安全な遷移先はサインインし直したときに着けるよう引き継ぎ、それ以外はサインイン画面の URL に
// 見せずに捨てる。
func TestToSignIn(t *testing.T) {
	t.Parallel()

	backURL := "/s/example/topics/1/pages/new?title=%E3%83%A1%E3%83%A2"

	tests := []struct {
		name         string
		backURL      string
		wantLocation string
	}{
		{
			name:         "安全なbackはクエリに載せる",
			backURL:      backURL,
			wantLocation: "/sign_in?back=" + url.QueryEscape(backURL),
		},
		{
			name:         "backが空ならクエリを付けない",
			backURL:      "",
			wantLocation: "/sign_in",
		},
		{
			name:         "絶対URLのbackは捨てる",
			backURL:      "https://evil.com",
			wantLocation: "/sign_in",
		},
		{
			name:         "ネットワークパス参照のbackは捨てる",
			backURL:      "//evil.com",
			wantLocation: "/sign_in",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/sign_in/two_factor/new", nil)
			rr := httptest.NewRecorder()

			ToSignIn(rr, req, tt.backURL)

			if rr.Code != http.StatusFound {
				t.Errorf("ステータスコード: got %d, want %d", rr.Code, http.StatusFound)
			}
			if location := rr.Header().Get("Location"); location != tt.wantLocation {
				t.Errorf("リダイレクト先: got %s, want %s", location, tt.wantLocation)
			}
		})
	}
}
