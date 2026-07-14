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

// NewCardLinkDraftPagesWithoutSpace builds draft page cards for the page editor's draft list column.
// The editor stays within a single space, so the space label is omitted by leaving SpaceName empty;
// the card then shows only the topic name and title. SpaceIdentifier is kept because the card still
// links to the page editor.
//
// [Ja] NewCardLinkDraftPagesWithoutSpace はページ編集画面の下書き一覧カラム用の下書きカードを生成する。
// 編集画面は同一スペース内のため、SpaceName を空にしてスペースラベルを省く (カードはトピック名と
// タイトルのみを表示する)。カードはページ編集画面へのリンクを保持するため SpaceIdentifier は残す。
func NewCardLinkDraftPagesWithoutSpace(drafts []*model.DraftPage) []CardLinkDraftPage {
	result := make([]CardLinkDraftPage, len(drafts))
	for i, d := range drafts {
		card := NewCardLinkDraftPage(d)
		card.SpaceName = ""
		result[i] = card
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

// draftPageTitle returns a draft page's display title, falling back to the published page's
// title and then to an empty string when the draft has no title of its own. Shared by the draft
// page view-models in this package (card and index list).
//
// [Ja] draftPageTitle は下書きページの表示タイトルを返す。下書き自身のタイトルが無ければ
// 公開ページのタイトル、それも無ければ空文字列にフォールバックする。本パッケージの下書き
// ページ用ビューモデル (カード・一覧) で共有する。
func draftPageTitle(d *model.DraftPage) string {
	if d.Title != nil && *d.Title != "" {
		return *d.Title
	}
	if d.Page != nil && d.Page.Title != nil && *d.Page.Title != "" {
		return *d.Page.Title
	}
	return ""
}
