package redirect

import "testing"

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
