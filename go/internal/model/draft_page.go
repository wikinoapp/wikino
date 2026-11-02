package model

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

// DraftPageはページの下書きのドメインモデル
type DraftPage struct {
	ID               DraftPageID
	SpaceID          SpaceID
	PageID           PageID
	SpaceMemberID    SpaceMemberID
	TopicID          TopicID
	SuggestionPageID *SuggestionPageID
	Title            *string
	Body             string
	// BaseTitleとBaseBodyは下書きを書き始めた時点のタイトルと本文 (起点)。
	// BaseBodyがnilの下書きは起点が分からない。BaseBodyがあってBaseTitleがnilの下書きは、
	// タイトルの無いページから書き始めたものである
	BaseTitle                 *string
	BaseBody                  *string
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

// DraftPageContentは下書きのある時点のタイトルと本文。
// タイトルの無い状態は、リビジョンと同じく空のタイトルで表す
type DraftPageContent struct {
	Title string
	Body  string
}

// BaseContentは下書きの起点を返す。起点が分からない下書きではnilを返す
func (d *DraftPage) BaseContent() *DraftPageContent {
	if d.BaseBody == nil {
		return nil
	}
	var title string
	if d.BaseTitle != nil {
		title = *d.BaseTitle
	}
	return &DraftPageContent{Title: title, Body: *d.BaseBody}
}

// PrecedingContentは、previousの次に作るリビジョン (previousがnilなら最古のリビジョン) から見た
// 直前の状態を返す。直前のリビジョンがあればその内容、無ければ下書きの起点を返す。
// 起点も分からなければ直前の状態は無く、nilを返す。
// 差分の比較対象とリビジョンを作るかの判定が同じ定義を使うよう、決め方をここに置く
func (d *DraftPage) PrecedingContent(previous *DraftPageRevision) *DraftPageContent {
	if previous != nil {
		return &DraftPageContent{Title: previous.Title, Body: previous.Body}
	}
	return d.BaseContent()
}

// NeedsRevisionは、latestの次にcontentのリビジョンを作るべきかを返す。latestは下書きの最新のリビジョンで、
// 無ければnilを渡す。直前の状態 (PrecedingContent) と同じ内容ならリビジョンを作らない。
// 直前の状態が無い (起点が分からず、リビジョンも無い) 下書きでは作る
func (d *DraftPage) NeedsRevision(latest *DraftPageRevision, content DraftPageContent) bool {
	preceding := d.PrecedingContent(latest)
	return preceding == nil || *preceding != content
}
