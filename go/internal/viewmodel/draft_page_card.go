package viewmodel

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// DraftPageCard represents a draft page card shown in the home page's "draft pages" section.
// It carries the title (with fallback to the published page title handled in draftPageTitle),
// the owning space's identifier and name (for the link and the space label), the topic
// name plus its visibility icon (public/private), and the published page number used to link
// to the page editor.
//
// [Ja] DraftPageCard はホーム画面の「下書きのページ」セクションに表示する下書きカード。
// タイトル (未設定時は公開ページのタイトルにフォールバック)、スペース識別子と名前 (リンクとラベル用)、
// トピック名と公開範囲アイコン (公開 / 非公開)、ページ編集画面へのリンクに使う公開ページ番号を保持する。
type DraftPageCard struct {
	title           string
	SpaceName       string
	TopicName       string
	TopicIconName   IconName
	SpaceIdentifier SpaceIdentifier
	PageNumber      int32
}

// DisplayTitle returns the display title for the card. Falls back to the localized
// "untitled" label when neither the draft title nor the published page title is set.
//
// [Ja] DisplayTitle はカードの表示タイトルを返す。下書きタイトル・公開ページタイトルがいずれも未設定の場合は
// ローカライズされた「無題」を返す。
func (d DraftPageCard) DisplayTitle(ctx context.Context) string {
	if d.title != "" {
		return d.title
	}
	return i18n.T(ctx, "home_draft_pages_untitled")
}

// NewDraftPageCard はモデルからホーム画面用 DraftPageCard を生成する
func NewDraftPageCard(d *model.DraftPage) DraftPageCard {
	return DraftPageCard{
		title:           draftPageTitle(d),
		SpaceName:       d.Topic.Space.Name,
		TopicName:       d.Topic.Name,
		TopicIconName:   topicVisibilityIconName(d.Topic.Visibility),
		SpaceIdentifier: NewSpaceIdentifier(d.Topic.Space.Identifier),
		PageNumber:      int32(d.Page.Number),
	}
}

// NewDraftPageCards はモデルのスライスからホーム画面用 DraftPageCard のスライスを生成する
func NewDraftPageCards(drafts []*model.DraftPage) []DraftPageCard {
	result := make([]DraftPageCard, len(drafts))
	for i, d := range drafts {
		result[i] = NewDraftPageCard(d)
	}
	return result
}

// SpaceForIcon returns a Space view-model populated with the fields required to render
// the small SpaceIcon next to the space name on each draft card (identifier drives the
// deterministic background color and first character; name is kept available for future use).
//
// [Ja] SpaceForIcon は下書きカードの SpaceName 横に表示する SpaceIcon を描画するために
// 必要な Space ビューモデルを返す。identifier は背景色と頭文字の決定に使われ、name は
// 将来の利用に備えて保持する。
func (d DraftPageCard) SpaceForIcon() Space {
	return Space{
		Name:       d.SpaceName,
		Identifier: d.SpaceIdentifier,
	}
}
