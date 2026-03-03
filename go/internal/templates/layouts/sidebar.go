package layouts

import "net/http"

const sidebarCookieName = "wikino_sidebar_open"

// SidebarDefaultClosed はリクエストのクッキーからサイドバーの初期状態を判定します。
// クッキーが存在しない場合は閉じた状態（true）を返します。
// クッキーが "true" の場合は開いた状態（false）を返します。
func SidebarDefaultClosed(r *http.Request) bool {
	cookie, err := r.Cookie(sidebarCookieName)
	if err != nil {
		return true
	}
	return cookie.Value != "true"
}
