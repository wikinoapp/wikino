package model

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

// Pageはページのドメインモデル
type Page struct {
	ID                        PageID
	SpaceID                   SpaceID
	TopicID                   TopicID
	Number                    PageNumber
	Title                     *string
	Body                      string
	LinkedPageIDs             []PageID
	ModifiedAt                time.Time
	PublishedAt               *time.Time
	TrashedAt                 *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	PinnedAt                  *time.Time
	DiscardedAt               *time.Time
	FeaturedImageAttachmentID *AttachmentID
}

// ContentDigestはページのAPI応答に含む値 (所属トピック・タイトル・本文・更新日時) から
// 計算したダイジェストを返す。公開APIのETagに使うため、応答が変われば値も変える。
// IDと番号は同じURLのページについて変化しないため、計算に含めない
func (p *Page) ContentDigest() string {
	// 各項目の境界が曖昧にならないよう、連結ではなくJSONの配列にしてからハッシュを取る。
	// 文字列・nil・time.Timeだけの配列のため、Marshalは失敗しない
	b, _ := json.Marshal([]any{p.TopicID, p.Title, p.Body, p.ModifiedAt.UTC()})
	sum := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
