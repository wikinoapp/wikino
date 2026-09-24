package session

import "regexp"

// og:image配信のパス。SNSのクローラーが取得し共有キャッシュに載せる画像のため、
// レスポンスに `Set-Cookie` を付けない (Cloudflareは既定では `Set-Cookie` 付きのレスポンスを
// キャッシュせず、キャッシュする設定にした場合は他人向けのCookieが配られる)。
// リバースプロキシのGo振り分け・CSRFトークンの発行除外・フラッシュの消費除外がこの定義を共有する。
// パスを足すときは `OGImagePathPatterns` に加え、リバースプロキシの `goHandledRegexPatterns` にも
// GET・HEADで登録する (登録漏れはリバースプロキシのテストで検出する)。
var (
	// AttachmentOGImagePathPatternは公開トピックのページから参照される添付ファイルのog:imageのパス。
	// /attachments/:id (ダウンロードURL) は対象外にするため、og_image末尾でパスを限定する
	AttachmentOGImagePathPattern = regexp.MustCompile(`^/attachments/[^/]+/og_image$`)

	// PageOGImagePathPatternはカバー画像の無い公開ページのカード画像のパス。
	// バージョンが古いURLもGoで受けて正規URLへ転送するため、バージョン部分は形式を限定しない
	PageOGImagePathPattern = regexp.MustCompile(`^/s/[^/]+/pages/\d+/og_image/[^/]+\.png$`)

	// OGImagePathPatternsはog:image配信のパスの一覧
	OGImagePathPatterns = []*regexp.Regexp{
		AttachmentOGImagePathPattern,
		PageOGImagePathPattern,
	}
)

// IsOGImagePathはpathがog:image配信のパスかを返す。
// 対象のメソッド (GET・HEADなど) は呼び出し側で判定する。
func IsOGImagePath(path string) bool {
	for _, pattern := range OGImagePathPatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}
