package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageShowUsecase aggregates the data shown on the page detail screen
// (GET /s/:space_identifier/pages/:page_number).
//
// It is a separate UseCase from GetPageDetailUsecase, which backs the editor and bails out unless
// the user is a space member. This screen is reachable by guests, so visibility is decided from the
// topic's visibility and the page's trash state instead of from membership alone.
//
// [Ja] GetPageShowUsecase はページ表示画面 (GET /s/:space_identifier/pages/:page_number) に
// 表示するデータを集約する読み取り UseCase。
//
// 編集画面向けの GetPageDetailUsecase はスペースメンバーでなければ何も返さないため流用せず、
// 別 UseCase として実装する。本画面はゲストも到達するので、可視性はメンバーかどうかではなく
// トピックの公開設定とページのゴミ箱状態から判定する。
type GetPageShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	attachmentRepo  *repository.AttachmentRepository
}

// NewGetPageShowUsecase creates a GetPageShowUsecase.
//
// [Ja] NewGetPageShowUsecase は GetPageShowUsecase を生成する。
func NewGetPageShowUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
) *GetPageShowUsecase {
	return &GetPageShowUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		attachmentRepo:  attachmentRepo,
	}
}

// GetPageShowInput holds the input parameters for fetching the page detail.
// UserID is nil when the user is not signed in (a public topic's page is viewable without signing in).
//
// [Ja] GetPageShowInput はページ表示画面のデータ取得の入力パラメータ。
// UserID は未ログイン時に nil になる (公開トピックのページは未ログインでも閲覧できる)。
type GetPageShowInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          *model.UserID

	// LinkLimit / BacklinkLimit / PageBacklinkLimit cap the three listings rendered under the body.
	// Their page fields select the slice rendered by a full-page pagination fallback; htmx loads
	// the same slices from the fragment endpoints behind each "load more" link.
	//
	// LinkPage / LinkedPageBacklinkPage / PageBacklinkPage are one-based and must already be
	// resolved by the caller (the handlers use viewmodel.PageLinkState.Normalized), so that the
	// slice fetched here and the page number the listing renders come from the same value.
	// LinkedPageNumber is zero when no card's nested backlink list is being advanced.
	//
	// [Ja] LinkLimit / BacklinkLimit / PageBacklinkLimit は本文の下に描画する 3 つの一覧の件数上限。
	// 各ページ指定はフルページのページネーションで描画する範囲を選び、htmx は各「もっと見る」
	// リンクのフラグメントエンドポイントから同じ範囲を読み込む。
	//
	// LinkPage / LinkedPageBacklinkPage / PageBacklinkPage は 1 始まりで、呼び出し元が解決済みの
	// 値を渡す (Handler は viewmodel.PageLinkState.Normalized を使う)。ここで取得する範囲と一覧が
	// 描画するページ番号を同じ値から導くためである。LinkedPageNumber は、どのカードのネストした
	// バックリンク一覧も進めていないときにゼロになる。
	LinkLimit              int32
	BacklinkLimit          int32
	PageBacklinkLimit      int32
	LinkPage               int32
	LinkedPageNumber       int32
	LinkedPageBacklinkPage int32
	PageBacklinkPage       int32
}

// GetPageShowOutput is the output of fetching the page detail.
//
// [Ja] GetPageShowOutput はページ表示画面のデータ取得の出力。
type GetPageShowOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	Page        *model.Page
	Topic       *model.Topic

	// IsTrashed reports whether the page is in the trash. It is only ever true for a viewer allowed
	// to see it (a member holding page:trash); everyone else gets a not-found error instead. The
	// template can therefore render the trash alert whenever this is true.
	//
	// [Ja] IsTrashed はページがゴミ箱に入っているかを表す。true になるのは閲覧を許可された閲覧者
	// (page:trash を持つメンバー) の場合だけで、それ以外には not found エラーを返す。したがって
	// テンプレートは true のときにゴミ箱アラートを出せばよい。
	IsTrashed bool

	CanUpdatePage bool

	// CanTrashPage reports whether the viewer may move this page into the trash, and gates the trash
	// item of the header's action dropdown. It rides on page:trash rather than page:write, so a
	// member who may rewrite the page does not thereby get the item (see Authorizer.CanTrashPage).
	//
	// [Ja] CanTrashPage は閲覧者がこのページをゴミ箱へ入れられるかを表し、ヘッダーの操作ドロップ
	// ダウンのゴミ箱項目の出し分けに使う。判定軸は page:write ではなく page:trash のため、ページを
	// 書き換えてよいメンバーというだけでは項目は出ない (Authorizer.CanTrashPage を参照)。
	CanTrashPage bool

	// FeaturedImageAttachment is the page's cover image attachment, used to build the og:image meta
	// tag. It is nil when the page has no cover image, or when the repository cannot resolve the
	// attachment. Only ID / SpaceID / Filename are populated (see
	// AttachmentRepository.FindByIDAndSpace).
	//
	// [Ja] FeaturedImageAttachment は og:image メタタグの組み立てに使うページのアイキャッチ画像の
	// 添付ファイル。アイキャッチ画像を持たない場合や、Repository が添付ファイルを解決できない場合は
	// nil。populate されるのは ID / SpaceID / Filename のみ
	// (AttachmentRepository.FindByIDAndSpace を参照)。
	FeaturedImageAttachment *model.Attachment

	// LinkedPages and PageBacklinks are the selected page of the two listings shown under the body,
	// and BacklinksPerPage holds the backlinks of each listed linked page. All three are narrowed
	// down to the topics the viewer may open, so a page the viewer cannot open never appears even
	// as a title.
	//
	// [Ja] LinkedPages と PageBacklinks は本文の下に表示する 2 つの一覧の選択ページ、
	// BacklinksPerPage は一覧した各リンク先ページのバックリンク。いずれも閲覧者が開けるトピックに
	// 絞り込むため、開けないページがタイトルだけでも現れることはない。
	LinkedPages       []*model.Page
	LinkedTotalCount  int64
	BacklinksPerPage  map[model.PageID]*LinkedPageBacklinks
	PageBacklinks     []*model.Page
	PageBacklinkCount int64
	LinkTopics        []*model.Topic
}

// Execute fetches the data shown on the page detail screen. It returns a *model.AppError with
// AppErrCodeResourceNotFound whenever the page must not be shown to the current viewer, so that the
// handler cannot tell "hidden" and "missing" apart in the response.
//
// [Ja] Execute はページ表示画面に表示するデータを取得する。現在の閲覧者に見せてはいけない場合は
// AppErrCodeResourceNotFound の *model.AppError を返し、レスポンス上で「隠している」と
// 「存在しない」を区別しないようにする。
func (uc *GetPageShowUsecase) Execute(ctx context.Context, input GetPageShowInput) (*GetPageShowOutput, error) {
	data, err := fetchPageAccessDataAllowingGuest(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	// The topics of the space are resolved once and reused for the authorization of this page and
	// for narrowing the listings down, so that a page hidden from the viewer cannot slip into the
	// link list of a page they may open.
	//
	// [Ja] スペースのトピックは一度だけ解決し、このページの認可と一覧の絞り込みの双方で使い回す。
	// これにより、閲覧者に見せないページが、開けるページのリンク一覧に紛れ込むことがなくなる。
	access, err := fetchTopicAccess(ctx, uc.pageAccessRepos(), data.space.ID, data.spaceMember)
	if err != nil {
		return nil, err
	}

	authorizer := access.authorizer(data.page.TopicID)

	if !authorizer.CanShowTopic(data.topic) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// A trashed page stays readable only for members who can open the trash, so that they can decide
	// whether to restore it. Guests and members without page:trash get a 404 instead of the body.
	//
	// [Ja] ゴミ箱に入ったページは、復元の判断ができるようゴミ箱を開ける権限を持つメンバーにだけ
	// 見せる。ゲストと page:trash を持たないメンバーには本文ではなく 404 を返す。
	isTrashed := data.page.TrashedAt != nil
	if isTrashed && !authorizer.CanShowTrash() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	featuredImageAttachment, err := uc.findFeaturedImageAttachment(ctx, data.page, data.space.ID)
	if err != nil {
		return nil, err
	}

	links, err := uc.fetchLinks(ctx, data.page, data.space.ID, access, input)
	if err != nil {
		return nil, err
	}

	return &GetPageShowOutput{
		Space:                   data.space,
		SpaceMember:             data.spaceMember,
		Page:                    data.page,
		Topic:                   data.topic,
		IsTrashed:               isTrashed,
		CanUpdatePage:           authorizer.CanUpdatePage(),
		CanTrashPage:            authorizer.CanTrashPage(),
		FeaturedImageAttachment: featuredImageAttachment,
		LinkedPages:             links.linkedPages,
		LinkedTotalCount:        links.linkedTotalCount,
		BacklinksPerPage:        links.backlinksPerPage,
		PageBacklinks:           links.pageBacklinks,
		PageBacklinkCount:       links.pageBacklinkCount,
		LinkTopics:              links.topics,
	}, nil
}

// pageShowLinks holds the selected page of the listings rendered under the body.
//
// [Ja] pageShowLinks は本文の下に描画する一覧の選択ページを保持する。
type pageShowLinks struct {
	linkedPages       []*model.Page
	linkedTotalCount  int64
	backlinksPerPage  map[model.PageID]*LinkedPageBacklinks
	pageBacklinks     []*model.Page
	pageBacklinkCount int64
	topics            []*model.Topic
}

// fetchLinks fetches the link list, the backlinks of each listed page and the page's own backlinks.
// Unlike the editor, this screen lists the links of the saved page: a member's own draft must not
// change what a public visitor sees, and guests hold no draft at all (Rails' Pages::ShowController
// passes the page record itself for the same reason).
//
// [Ja] fetchLinks はリンク一覧・一覧した各ページのバックリンク・ページ自身のバックリンクを取得する。
// 編集画面と違い本画面は保存済みページのリンクを一覧する。メンバー自身の下書きが公開の閲覧者の
// 見え方を変えてはならず、ゲストはそもそも下書きを持たないため (Rails 版 Pages::ShowController も
// 同じ理由でページレコード自身を渡している)。
func (uc *GetPageShowUsecase) fetchLinks(ctx context.Context, pg *model.Page, spaceID model.SpaceID, access *topicAccess, input GetPageShowInput) (*pageShowLinks, error) {
	lists, err := fetchRelatedPageLists(ctx, uc.pageRepo, relatedPageListInput{
		PageID:                 pg.ID,
		LinkedPageIDs:          pg.LinkedPageIDs,
		SpaceID:                spaceID,
		Visibility:             access.visibility(),
		LinkPage:               input.LinkPage,
		LinkLimit:              input.LinkLimit,
		LinkedPageNumber:       input.LinkedPageNumber,
		LinkedPageBacklinkPage: input.LinkedPageBacklinkPage,
		BacklinkLimit:          input.BacklinkLimit,
		PageBacklinkPage:       input.PageBacklinkPage,
		PageBacklinkLimit:      input.PageBacklinkLimit,
	})
	if err != nil {
		return nil, err
	}

	topicMap := access.topicMapForPages(lists.pageGroups...)
	topics := make([]*model.Topic, 0, len(topicMap))
	for _, topic := range topicMap {
		topics = append(topics, topic)
	}

	return &pageShowLinks{
		linkedPages:       lists.linkedPages,
		linkedTotalCount:  lists.linkedTotalCount,
		backlinksPerPage:  lists.backlinksPerPage,
		pageBacklinks:     lists.pageBacklinks,
		pageBacklinkCount: lists.pageBacklinkCount,
		topics:            topics,
	}, nil
}

func (uc *GetPageShowUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

// findFeaturedImageAttachment resolves the page's cover image attachment for the og:image meta tag.
// A page without a cover image, or one whose attachment the repository cannot resolve, yields nil
// rather than an error: the meta tag falls back to the default OGP image and the page itself still
// renders.
//
// [Ja] findFeaturedImageAttachment は og:image メタタグ用にページのアイキャッチ画像の添付ファイルを
// 解決する。アイキャッチ画像を持たないページや、Repository が添付ファイルを解決できないページでは
// エラーにせず nil を返す。メタタグは既定の OGP 画像にフォールバックし、ページ自体は描画できる。
func (uc *GetPageShowUsecase) findFeaturedImageAttachment(ctx context.Context, pg *model.Page, spaceID model.SpaceID) (*model.Attachment, error) {
	if pg.FeaturedImageAttachmentID == nil {
		return nil, nil
	}

	attachment, err := uc.attachmentRepo.FindByIDAndSpace(ctx, *pg.FeaturedImageAttachmentID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の添付ファイルの取得に失敗: %w", err)
	}

	return attachment, nil
}
