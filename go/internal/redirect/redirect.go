// Package redirectはリダイレクトURLのバリデーションと、それに依存するリダイレクトを提供する。
package redirect

import (
	"net/http"
	"net/url"
	"strings"
)

// ValidateBackURLはbackパラメータが安全な同一オリジンのパスかを返す。
// ブラウザが別オリジンとして解釈しうる値をすべて拒否する。絶対パスでない値、
// ネットワークパス参照、ブラウザがスラッシュへ正規化するバックスラッシュ、
// ブラウザが解析前に取り除く制御文字が対象である。
func ValidateBackURL(backURL string) bool {
	// 空文字は遷移先を指していない。
	if backURL == "" {
		return false
	}

	// 遷移先を同一オリジンに留めるため、絶対パスだけを許可する。
	if !strings.HasPrefix(backURL, "/") {
		return false
	}

	// 先頭の `//` はネットワークパス参照で、別のオリジンを指す。
	if strings.HasPrefix(backURL, "//") {
		return false
	}

	// ブラウザは特殊URLの解析時にバックスラッシュをスラッシュへ正規化するため、
	// `/\evil.example` は拒否しなければ外部のネットワークパス参照として解釈される。
	if strings.Contains(backURL, `\`) {
		return false
	}

	// ブラウザはURLの解析前にASCIIのタブと改行を取り除くため、`/<TAB>/evil.example` の
	// 水平タブは消えてネットワークパス参照が残る。GoはこのタブをそのままLocationヘッダーへ
	// 書き出すので、ここで拒否する必要がある。url.Parseは制御文字を含むURLをエラーにするため、
	// 1文字ずつ列挙せずにこの種類をまとめて弾ける。解析後のURLにはスキームもホストも
	// 無いことを求め、同一オリジンのパスだけが残るようにする。
	parsed, err := url.Parse(backURL)
	if err != nil {
		return false
	}
	if parsed.Scheme != "" || parsed.Opaque != "" || parsed.Host != "" {
		return false
	}

	return true
}

// GetSafeRedirectURLは安全なリダイレクトURLを返す
// backURLが無効な場合はデフォルトURL ("/") を返す
func GetSafeRedirectURL(backURL string) string {
	if ValidateBackURL(backURL) {
		return backURL
	}
	return "/"
}

// ToSignInはサインインのフローを続けられなくなった訪問者を、その出発点へ戻す。
// 安全な遷移先であるbackURLはクエリに載せ、サインインし直しても訪問者が求めたページへ着ける
// ようにする。それ以外の値は、訪問者が目にする画面のURLに残さず捨てる。
func ToSignIn(w http.ResponseWriter, r *http.Request, backURL string) {
	signInURL := "/sign_in"
	if ValidateBackURL(backURL) {
		signInURL += "?back=" + url.QueryEscape(backURL)
	}

	http.Redirect(w, r, signInURL, http.StatusFound)
}
