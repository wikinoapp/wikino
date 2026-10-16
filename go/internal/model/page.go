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

// IsPublishedはページが公開済みかを返す
func (p *Page) IsPublished() bool {
	return p.PublishedAt != nil
}

// ContentDigestはページの内容 (所属するトピック・タイトル・本文・更新日時・公開状態) から
// 計算したダイジェストを返す。公開APIのETagと `If-Match` の照合に使うため、内容が変われば値も変える。
// IDと番号は同じURLのページについて変化しないため、計算に含めない。
// 所属するトピックはIDで含め、トピック名は含めない。トピック名を変えただけで
// そのトピックの全ページの版が変わり、取得済みの版での更新が412になるのを避けるため。
// 公開日時は応答に含めないため、公開済みかどうかだけを含める
func (p *Page) ContentDigest() string {
	// 各項目の境界が曖昧にならないよう、連結ではなくJSONの配列にしてからハッシュを取る。
	// 文字列・nil・真偽値・time.Timeだけの配列のため、Marshalは失敗しない
	b, _ := json.Marshal([]any{p.TopicID, p.Title, p.Body, p.ModifiedAt.UTC(), p.IsPublished()})
	sum := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
