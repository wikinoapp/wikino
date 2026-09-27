package session_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/session"
)

func TestIsOGImagePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "添付ファイルのog:image", path: "/attachments/550e8400-e29b-41d4-a716-446655440000/og_image", want: true},
		{name: "ページのカード画像", path: "/s/example/pages/1/og_image/0123456789abcdef.png", want: true},
		{name: "形式の違うバージョンのカード画像", path: "/s/example/pages/1/og_image/old.png", want: true},
		{name: "添付ファイルのダウンロードURL", path: "/attachments/550e8400-e29b-41d4-a716-446655440000", want: false},
		{name: "og_imageの後ろにパスが続く添付ファイル", path: "/attachments/550e8400-e29b-41d4-a716-446655440000/og_image/extra", want: false},
		{name: "拡張子がpngでないカード画像", path: "/s/example/pages/1/og_image/0123456789abcdef.jpg", want: false},
		{name: "バージョンの無いカード画像", path: "/s/example/pages/1/og_image", want: false},
		{name: "ページ表示画面", path: "/s/example/pages/1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := session.IsOGImagePath(tt.path); got != tt.want {
				t.Errorf("IsOGImagePath(%q) = %v、期待値 = %v", tt.path, got, tt.want)
			}
		})
	}
}
