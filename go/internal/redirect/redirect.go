// Package redirect provides the validation of redirect URLs and the redirects that depend on it.
//
// [Ja] Package redirect はリダイレクト URL のバリデーションと、それに依存するリダイレクトを提供する。
package redirect

import (
	"net/http"
	"net/url"
	"strings"
)

// ValidateBackURL reports whether the back parameter is a safe same-origin path.
// It rejects everything a browser could read as another origin: a value that is not an absolute
// path, a network-path reference, a backslash that a browser normalizes to a slash, and a control
// character that a browser strips before parsing.
//
// [Ja] ValidateBackURL は back パラメータが安全な同一オリジンのパスかを返す。
// ブラウザが別オリジンとして解釈しうる値をすべて拒否する。絶対パスでない値、
// ネットワークパス参照、ブラウザがスラッシュへ正規化するバックスラッシュ、
// ブラウザが解析前に取り除く制御文字が対象である。
func ValidateBackURL(backURL string) bool {
	// An empty value names no destination.
	//
	// [Ja] 空文字は遷移先を指していない。
	if backURL == "" {
		return false
	}

	// Only an absolute path is allowed, so that the destination stays on this origin.
	//
	// [Ja] 遷移先を同一オリジンに留めるため、絶対パスだけを許可する。
	if !strings.HasPrefix(backURL, "/") {
		return false
	}

	// A leading `//` is a network-path reference and names another origin.
	//
	// [Ja] 先頭の `//` はネットワークパス参照で、別のオリジンを指す。
	if strings.HasPrefix(backURL, "//") {
		return false
	}

	// Browsers normalize backslashes to slashes when parsing special URLs, so `/\evil.example`
	// would otherwise be interpreted as an external network-path reference.
	//
	// [Ja] ブラウザは特殊 URL の解析時にバックスラッシュをスラッシュへ正規化するため、
	// `/\evil.example` は拒否しなければ外部のネットワークパス参照として解釈される。
	if strings.Contains(backURL, `\`) {
		return false
	}

	// Browsers remove ASCII tab and newline before parsing a URL, so a horizontal tab in
	// `/<TAB>/evil.example` disappears and leaves a network-path reference. Go writes that tab into
	// the Location header untouched, so it has to be rejected here. url.Parse fails on any control
	// character, which covers the whole class instead of one character at a time. The parsed URL is
	// then required to name neither a scheme nor a host, so that only a same-origin path is left.
	//
	// [Ja] ブラウザは URL の解析前に ASCII のタブと改行を取り除くため、`/<TAB>/evil.example` の
	// 水平タブは消えてネットワークパス参照が残る。Go はこのタブをそのまま Location ヘッダーへ
	// 書き出すので、ここで拒否する必要がある。url.Parse は制御文字を含む URL をエラーにするため、
	// 1 文字ずつ列挙せずにこの種類をまとめて弾ける。解析後の URL にはスキームもホストも
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

// GetSafeRedirectURL は安全なリダイレクトURLを返す
// backURL が無効な場合はデフォルトURL（"/"）を返す
func GetSafeRedirectURL(backURL string) string {
	if ValidateBackURL(backURL) {
		return backURL
	}
	return "/"
}

// ToSignIn sends a visitor whose sign-in flow cannot continue back to its start. A backURL that is
// a safe destination is carried in the query, so that signing in again still lands the visitor on
// the page they asked for; any other value is dropped rather than left in the URL of a screen the
// visitor sees.
//
// [Ja] ToSignIn はサインインのフローを続けられなくなった訪問者を、その出発点へ戻す。
// 安全な遷移先である backURL はクエリに載せ、サインインし直しても訪問者が求めたページへ着ける
// ようにする。それ以外の値は、訪問者が目にする画面の URL に残さず捨てる。
func ToSignIn(w http.ResponseWriter, r *http.Request, backURL string) {
	signInURL := "/sign_in"
	if ValidateBackURL(backURL) {
		signInURL += "?back=" + url.QueryEscape(backURL)
	}

	http.Redirect(w, r, signInURL, http.StatusFound)
}
