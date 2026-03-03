package layouts

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSidebarDefaultClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cookieValue string
		hasCookie   bool
		want        bool
	}{
		{
			name:      "クッキーが存在しない場合は閉じた状態を返す",
			hasCookie: false,
			want:      true,
		},
		{
			name:        "クッキーが true の場合は開いた状態を返す",
			cookieValue: "true",
			hasCookie:   true,
			want:        false,
		},
		{
			name:        "クッキーが false の場合は閉じた状態を返す",
			cookieValue: "false",
			hasCookie:   true,
			want:        true,
		},
		{
			name:        "クッキーが不正な値の場合は閉じた状態を返す",
			cookieValue: "invalid",
			hasCookie:   true,
			want:        true,
		},
		{
			name:        "クッキーが空文字の場合は閉じた状態を返す",
			cookieValue: "",
			hasCookie:   true,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.hasCookie {
				r.AddCookie(&http.Cookie{
					Name:  sidebarCookieName,
					Value: tt.cookieValue,
				})
			}

			got := SidebarDefaultClosed(r)
			if got != tt.want {
				t.Errorf("SidebarDefaultClosed() = %v, want %v", got, tt.want)
			}
		})
	}
}
