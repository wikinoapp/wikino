package ogcard

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// designRevisionはカードのデザインのリビジョン
//
// 配色・レイアウト・フォントなど、同じ内容でも描画結果が変わる変更をしたら1つ上げる。
// バージョンが変わってURLが変わるため、SNSとCDNがキャッシュした古いデザインの画像を使い続けない。
const designRevision = 2

// versionLengthはバージョンの文字数 (16進数)
const versionLength = 16

// Versionはカードに描く内容とデザインのリビジョンから、カード画像のバージョンを計算する
//
// 内容が同じなら同じ値を返し、どれか1つでも変わると別の値を返す。
// カード画像のURLに含めることで、改名後にSNSとCDNが新しい画像を取りに来るようにする。
func (c Card) Version() string {
	h := sha256.New()
	// 区切り文字を含む名前で境界がずれて衝突しないよう、各値の前に長さを書く
	_, _ = fmt.Fprintf(h, "%d\n", designRevision)
	for _, s := range []string{c.SpaceName, c.TopicName, c.PageTitle} {
		_, _ = fmt.Fprintf(h, "%d:%s", len(s), s)
	}
	return hex.EncodeToString(h.Sum(nil))[:versionLength]
}
