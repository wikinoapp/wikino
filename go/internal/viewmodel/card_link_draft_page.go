package viewmodel

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// CardLinkDraftPageはCardLinkDraftPageコンポーネント用のビューモデルで、ホーム画面の
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

// DisplayTitleはカードの表示タイトルを返す。下書きタイトル・公開ページタイトルがいずれも未設定の場合は
// ローカライズされた「無題」を返す。
func (d CardLinkDraftPage) DisplayTitle(ctx context.Context) string {
	if d.title != "" {
		return d.title
	}
	return i18n.T(ctx, "home_draft_pages_untitled")
}

// NewCardLinkDraftPageはモデルからCardLinkDraftPageを生成する。
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

// NewCardLinkDraftPagesはモデルのスライスからCardLinkDraftPageのスライスを生成する。
func NewCardLinkDraftPages(drafts []*model.DraftPage) []CardLinkDraftPage {
	result := make([]CardLinkDraftPage, len(drafts))
	for i, d := range drafts {
		result[i] = NewCardLinkDraftPage(d)
	}
	return result
}

// NewCardLinkDraftPagesWithoutSpaceはページ編集画面の下書き一覧カラム用の下書きカードを生成する。
// 編集画面は同一スペース内のため、SpaceNameを空にしてスペースラベルを省く (カードはトピック名と
// タイトルのみを表示する)。カードはページ編集画面へのリンクを保持するためSpaceIdentifierは残す。
func NewCardLinkDraftPagesWithoutSpace(drafts []*model.DraftPage) []CardLinkDraftPage {
	result := make([]CardLinkDraftPage, len(drafts))
	for i, d := range drafts {
		card := NewCardLinkDraftPage(d)
		card.SpaceName = ""
		result[i] = card
	}
	return result
}

// SpaceForIconは下書きカードのSpaceName横に表示するSpaceIconを描画するために
// 必要なSpaceビューモデルを返す。identifierは背景色と頭文字の決定に使われ、nameは
// 将来の利用に備えて保持する。
func (d CardLinkDraftPage) SpaceForIcon() Space {
	return Space{
		Name:       d.SpaceName,
		Identifier: d.SpaceIdentifier,
	}
}

// draftPageTitleは下書きページの表示タイトルを返す。下書き自身のタイトルが無ければ
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
