package viewmodel

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// CardLinkDraftPage is the view-model for the CardLinkDraftPage component, a reusable draft page
// card shared by the home page's "draft pages" section and the page editor's draft list column.
// It carries the title (with fallback to the published page title handled in draftPageTitle), the
// owning space's identifier and name (for the link and the space label), the topic name plus its
// visibility icon (public/private), and the published page number used to link to the page editor.
//
// [Ja] CardLinkDraftPage は CardLinkDraftPage コンポーネント用のビューモデルで、ホーム画面の
// 「下書きのページ」セクションとページ編集画面の下書き一覧カラムで共有する再利用可能な下書きカード。
// タイトル (未設定時は公開ページのタイトルにフォールバック)、スペース識別子と名前 (リンクとラベル用)、
// トピック名と公開範囲アイコン (公開 / 非公開)、ページ編集画面へのリンクに使う公開ページ番号を保持する。
type CardLinkDraftPage struct {
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
func (d CardLinkDraftPage) DisplayTitle(ctx context.Context) string {
	if d.title != "" {
		return d.title
	}
	return i18n.T(ctx, "home_draft_pages_untitled")
}

// NewCardLinkDraftPage builds a CardLinkDraftPage view-model from a draft page model.
// [Ja] NewCardLinkDraftPage はモデルから CardLinkDraftPage を生成する。
func NewCardLinkDraftPage(d *model.DraftPage) CardLinkDraftPage {
	return CardLinkDraftPage{
		title:           draftPageTitle(d),
		SpaceName:       d.Topic.Space.Name,
		TopicName:       d.Topic.Name,
		TopicIconName:   topicVisibilityIconName(d.Topic.Visibility),
		SpaceIdentifier: NewSpaceIdentifier(d.Topic.Space.Identifier),
		PageNumber:      int32(d.Page.Number),
	}
}

// NewCardLinkDraftPages builds a slice of CardLinkDraftPage view-models from a slice of models.
// [Ja] NewCardLinkDraftPages はモデルのスライスから CardLinkDraftPage のスライスを生成する。
func NewCardLinkDraftPages(drafts []*model.DraftPage) []CardLinkDraftPage {
	result := make([]CardLinkDraftPage, len(drafts))
	for i, d := range drafts {
		result[i] = NewCardLinkDraftPage(d)
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
func (d CardLinkDraftPage) SpaceForIcon() Space {
	return Space{
		Name:       d.SpaceName,
		Identifier: d.SpaceIdentifier,
	}
}
