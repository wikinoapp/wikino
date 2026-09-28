package session

import (
	"net/http"
	"regexp"
	"slices"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// Cookieを使わないリクエストの定義。
// リバースプロキシのGo振り分け・CSRFトークンの発行と検証の除外・フラッシュの消費除外がこの定義を共有する。
// パスを足すときは `CookielessPaths` に加え、リバースプロキシの `goHandledRegexPatterns` にも
// 同じパターンを同じメソッドで登録する (登録漏れはリバースプロキシのテストで検出する)。
//
//   - og:image配信: SNSのクローラーが取得し共有キャッシュに載せる画像のため、レスポンスに `Set-Cookie` を
//     付けない (Cloudflareは既定では `Set-Cookie` 付きのレスポンスをキャッシュせず、キャッシュする設定に
//     した場合は他人向けのCookieが配られる)
//   - 公開Web API: `Authorization: Bearer` のトークンで認証し、Cookieのセッションを使わない。
//     ブラウザが自動で送るCookieを認証に使わないため、CSRFトークンの照合も要らない
//   - OAuthのトークン・失効エンドポイントとメタデータ: Cookieのセッションを使わず、
//     クライアントがCSRFトークンを送れないため除外する
var (
	// AttachmentOGImagePathPatternは公開トピックのページから参照される添付ファイルのog:imageのパス。
	// /attachments/:id (ダウンロードURL) は対象外にするため、og_image末尾でパスを限定する
	AttachmentOGImagePathPattern = regexp.MustCompile(`^/attachments/[^/]+/og_image$`)

	// PageOGImagePathPatternはカバー画像の無い公開ページのカード画像のパス。
	// バージョンが古いURLもGoで受けて正規URLへ転送するため、バージョン部分は形式を限定しない
	PageOGImagePathPattern = regexp.MustCompile(`^/s/[^/]+/pages/\d+/og_image/[^/]+\.png$`)

	// APIPathPatternは公開Web APIのパス。バージョンに関わらず `/api/` 配下をすべて含める
	APIPathPattern = regexp.MustCompile(`^/api/`)

	// OAuthTokenPathPatternはCookieによるセッションを使わないOAuthのエンドポイント。
	// 同意画面の`/oauth/authorize`はセッションを使うため含めない
	OAuthTokenPathPattern = regexp.MustCompile(`^/oauth/(token|revoke)$`)

	// APIMetadataPathPatternはOAuthとAPIのメタデータのパス。
	// リバースプロキシのGo振り分けと同じ範囲に限定する
	APIMetadataPathPattern = regexp.MustCompile(`^/\.well-known/(oauth-authorization-server|oauth-protected-resource(/.*)?|api-catalog)$`)
)

// CookielessPathはCookieを使わないリクエストのパスとメソッドの組
type CookielessPath struct {
	Pattern *regexp.Regexp
	// Methodsは対象のメソッド。空なら全メソッドが対象
	Methods []string
}

// CookielessPathsはCookieを使わないリクエストの一覧
var CookielessPaths = []CookielessPath{
	{Pattern: AttachmentOGImagePathPattern, Methods: []string{http.MethodGet, http.MethodHead}},
	{Pattern: PageOGImagePathPattern, Methods: []string{http.MethodGet, http.MethodHead}},
	{Pattern: APIPathPattern},
	{Pattern: OAuthTokenPathPattern},
	{Pattern: APIMetadataPathPattern},
}

// IsCookielessRequestはmethodとpathのリクエストがCookieを使わないかを返す
func IsCookielessRequest(method, path string) bool {
	for _, cp := range CookielessPaths {
		if !cp.Pattern.MatchString(path) {
			continue
		}
		if len(cp.Methods) == 0 || slices.Contains(cp.Methods, method) {
			return true
		}
	}
	return false
}

// IsAPIPathはpathが公開Web APIのパスかを返す。
// APIリファレンス (`model.APIReferencePath`) は人が読むHTMLのため、`/api/` 配下でも含めない。
// エラーやメンテナンス中の応答をProblem DetailsではなくHTMLで返し、APIに共通の `noindex` を付けずに
// 検索エンジンにインデックスさせる。Cookieを使わない点はAPIと同じため、`CookielessPaths` には残す
func IsAPIPath(path string) bool {
	return APIPathPattern.MatchString(path) && path != model.APIReferencePath
}
