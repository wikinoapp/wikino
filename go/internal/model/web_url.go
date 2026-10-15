package model

import (
	"fmt"
	"net/url"
)

// PageWebURLは、Web画面でページを開くURLを返す。appURLはWikinoのオリジン (例: `https://example.com`)。
// パスはWeb画面のパスヘルパー (`templates.PagePath`) と同じ形にする。
// 公開APIの応答の `url` に使い、スペースの独自ドメインを入れるときはこの関数とPageEditWebURLを直す
func PageWebURL(appURL string, space *Space, number PageNumber) string {
	return fmt.Sprintf("%s/s/%s/pages/%d", appURL, url.PathEscape(string(space.Identifier)), number)
}

// PageEditWebURLは、Web画面でページの編集画面を開くURLを返す。
// パスはWeb画面のパスヘルパー (`templates.PageEditPath`) と同じ形にする。
// 下書きは編集画面で開くため、公開APIの下書きの応答の `url` に使う
func PageEditWebURL(appURL string, space *Space, number PageNumber) string {
	return PageWebURL(appURL, space, number) + "/edit"
}
