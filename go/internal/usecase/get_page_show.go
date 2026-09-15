package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageShowUsecaseはページ表示画面 (GET /s/:space_identifier/pages/:page_number) に
// 表示するデータを集約する読み取りUseCase。
//
// 編集画面向けのGetPageDetailUsecaseはスペースメンバーでなければ何も返さないため流用せず、
// 別UseCaseとして実装する。本画面はゲストも到達するので、可視性はメンバーかどうかではなく
// トピックの公開設定とページのゴミ箱状態から判定する。
type GetPageShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	attachmentRepo  *repository.AttachmentRepository
}

// NewGetPageShowUsecaseはGetPageShowUsecaseを生成する。
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

// GetPageShowInputはページ表示画面のデータ取得の入力パラメータ。
// UserIDは未ログイン時にnilになる (公開トピックのページは未ログインでも閲覧できる)。
type GetPageShowInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          *model.UserID

	// LinkLimit / BacklinkLimit / PageBacklinkLimitは本文の下に描画する3つの一覧の件数上限。
	// 各ページ指定はフルページのページネーションで描画する範囲を選び、htmxは各「もっと見る」
	// リンクのフラグメントエンドポイントから同じ範囲を読み込む。
	//
	// LinkPage / LinkedPageBacklinkPage / PageBacklinkPageは1始まりで、呼び出し元が解決済みの
	// 値を渡す (Handlerはviewmodel.PageLinkState.Normalizedを使う)。ここで取得する範囲と一覧が
	// 描画するページ番号を同じ値から導くためである。LinkedPageNumberは、どのカードのネストした
	// バックリンク一覧も進めていないときにゼロになる。
	LinkLimit              int32
	BacklinkLimit          int32
	PageBacklinkLimit      int32
	LinkPage               int32
	LinkedPageNumber       int32
	LinkedPageBacklinkPage int32
	PageBacklinkPage       int32
}

// GetPageShowOutputはページ表示画面のデータ取得の出力。
type GetPageShowOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	Page        *model.Page
	Topic       *model.Topic

	// IsTrashedはページがゴミ箱に入っているかを表す。trueになるのは閲覧を許可された閲覧者
	// (page:trashを持つメンバー) の場合だけで、それ以外にはnot foundエラーを返す。したがって
	// テンプレートはtrueのときにゴミ箱アラートを出せばよい。
	IsTrashed bool

	CanUpdatePage bool

	// CanTrashPageは閲覧者がこのページをゴミ箱へ入れられるかを表し、ヘッダーの操作ドロップ
	// ダウンのゴミ箱項目の出し分けに使う。判定軸はpage:writeではなくpage:trashのため、ページを
	// 書き換えてよいメンバーというだけでは項目は出ない (Authorizer.CanTrashPageを参照)。
	CanTrashPage bool

	// FeaturedImageAttachmentはog:imageメタタグの組み立てに使うページのアイキャッチ画像の
	// 添付ファイル。アイキャッチ画像を持たない場合や、Repositoryが添付ファイルを解決できない場合は
	// nil。populateされるのはID / SpaceID / Filenameのみ
	// (AttachmentRepository.FindByIDAndSpaceを参照)。
	FeaturedImageAttachment *model.Attachment

	// LinkedPagesとPageBacklinksは本文の下に表示する2つの一覧の選択ページ、
	// BacklinksPerPageは一覧した各リンク先ページのバックリンク。いずれも閲覧者が開けるトピックに
	// 絞り込むため、開けないページがタイトルだけでも現れることはない。
	LinkedPages       []*model.Page
	LinkedTotalCount  int64
	BacklinksPerPage  map[model.PageID]*LinkedPageBacklinks
	PageBacklinks     []*model.Page
	PageBacklinkCount int64
	LinkTopics        []*model.Topic
}

// Executeはページ表示画面に表示するデータを取得する。現在の閲覧者に見せてはいけない場合は
// AppErrCodeResourceNotFoundの *model.AppErrorを返し、レスポンス上で「隠している」と
// 「存在しない」を区別しないようにする。
func (uc *GetPageShowUsecase) Execute(ctx context.Context, input GetPageShowInput) (*GetPageShowOutput, error) {
	data, err := fetchPageAccessDataAllowingGuest(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	// スペースのトピックは一度だけ解決し、このページの認可と一覧の絞り込みの双方で使い回す。
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

	// ゴミ箱に入ったページは、復元の判断ができるようゴミ箱を開ける権限を持つメンバーにだけ
	// 見せる。ゲストとpage:trashを持たないメンバーには本文ではなく404を返す。
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

// pageShowLinksは本文の下に描画する一覧の選択ページを保持する。
type pageShowLinks struct {
	linkedPages       []*model.Page
	linkedTotalCount  int64
	backlinksPerPage  map[model.PageID]*LinkedPageBacklinks
	pageBacklinks     []*model.Page
	pageBacklinkCount int64
	topics            []*model.Topic
}

// fetchLinksはリンク一覧・一覧した各ページのバックリンク・ページ自身のバックリンクを取得する。
// 編集画面と違い本画面は保存済みページのリンクを一覧する。メンバー自身の下書きが公開の閲覧者の
// 見え方を変えてはならず、ゲストはそもそも下書きを持たないため (Rails版Pages::ShowControllerも
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

// findFeaturedImageAttachmentはog:imageメタタグ用にページのアイキャッチ画像の添付ファイルを
// 解決する。アイキャッチ画像を持たないページや、Repositoryが添付ファイルを解決できないページでは
// エラーにせずnilを返す。メタタグは既定のOGP画像にフォールバックし、ページ自体は描画できる。
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
