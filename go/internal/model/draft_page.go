package model

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

// DraftPageはページの下書きのドメインモデル
type DraftPage struct {
	ID                        DraftPageID
	SpaceID                   SpaceID
	PageID                    PageID
	SpaceMemberID             SpaceMemberID
	TopicID                   TopicID
	SuggestionPageID          *SuggestionPageID
	Title                     *string
	Body                      string
	LinkedPageIDs             []PageID
	FeaturedImageAttachmentID *AttachmentID
	ModifiedAt                time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time

	Page  *Page
	Topic *Topic
}

// ContentDigestは下書きの内容 (所属するトピック・タイトル・本文・更新日時) から計算した
// ダイジェストを返す。公開APIのETagと `If-Match` の照合に使うため、内容が変われば値も変える。
// ページ番号は同じURLの下書きについて変化しないため、計算に含めない。
// 所属するトピックはIDで含め、トピック名は含めない。トピック名を変えただけで
// そのトピックの全下書きの版が変わり、取得済みの版での更新が412になるのを避けるため
func (d *DraftPage) ContentDigest() string {
	// 各項目の境界が曖昧にならないよう、連結ではなくJSONの配列にしてからハッシュを取る。
	// 文字列・nil・time.Timeだけの配列のため、Marshalは失敗しない
	b, _ := json.Marshal([]any{d.TopicID, d.Title, d.Body, d.ModifiedAt.UTC()})
	sum := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
