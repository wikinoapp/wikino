package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetEditLinkDataUsecase は編集画面のリンクデータ取得ユースケース
type GetEditLinkDataUsecase struct {
	pageRepo  *repository.PageRepository
	topicRepo *repository.TopicRepository
}

// NewGetEditLinkDataUsecase は GetEditLinkDataUsecase を生成する
func NewGetEditLinkDataUsecase(
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
) *GetEditLinkDataUsecase {
	return &GetEditLinkDataUsecase{
		pageRepo:  pageRepo,
		topicRepo: topicRepo,
	}
}

// GetEditLinkDataInput holds the related-page pagination input for the editor.
//
// CurrentPage / LinkedPageBacklinkPage / PageBacklinkPage are one-based and must already be
// resolved by the caller (the handlers use viewmodel.PageLinkState.Normalized), so that the slice
// fetched here and the page number the listing renders come from the same value. LinkedPageNumber
// is zero when no card's nested backlink list is being advanced.
//
// [Ja] GetEditLinkDataInput は編集画面の関連ページページネーション入力を保持する。
//
// CurrentPage / LinkedPageBacklinkPage / PageBacklinkPage は 1 始まりで、呼び出し元が解決済みの値を
// 渡す (Handler は viewmodel.PageLinkState.Normalized を使う)。ここで取得する範囲と一覧が描画する
// ページ番号を同じ値から導くためである。LinkedPageNumber は、どのカードのネストしたバックリンク
// 一覧も進めていないときにゼロになる。
type GetEditLinkDataInput struct {
	Page                   *model.Page
	DraftPage              *model.DraftPage
	SpaceID                model.SpaceID
	CurrentPage            int32
	LinkLimit              int32
	BacklinkLimit          int32
	PageBacklinkLimit      int32
	LinkedPageNumber       int32
	LinkedPageBacklinkPage int32
	PageBacklinkPage       int32

	// IncludePrecedingPages makes each listing return every page from the first through the
	// requested one. The draft refresh sets it because it replaces the listing containers wholesale,
	// so it has to re-render what the reader appended through htmx as well as the requested page.
	// A screen that renders the listings from scratch leaves it at false.
	//
	// [Ja] IncludePrecedingPages は各一覧が 1 ページ目から要求ページまでを返すようにする。下書き
	// 再取得は一覧のコンテナごと差し替えるため、要求ページに加えて閲覧者が htmx で追記した範囲も
	// 描画し直す必要があり、これを立てる。一覧を最初から描画する画面では false のままにする。
	IncludePrecedingPages bool
}

// GetEditLinkDataOutput は編集画面のリンクデータ取得の出力
type GetEditLinkDataOutput struct {
	LinkedPages       []*model.Page
	LinkedTotalCount  int64
	BacklinksPerPage  map[model.PageID]*LinkedPageBacklinks
	PageBacklinks     []*model.Page
	PageBacklinkCount int64
	LinkTopics        []*model.Topic
}

// Execute は編集画面のリンク・バックリンクデータを取得する
func (uc *GetEditLinkDataUsecase) Execute(ctx context.Context, input GetEditLinkDataInput) (*GetEditLinkDataOutput, error) {
	var linkedPageIDs []model.PageID
	if input.DraftPage != nil {
		linkedPageIDs = input.DraftPage.LinkedPageIDs
	} else {
		linkedPageIDs = input.Page.LinkedPageIDs
	}

	// Only the topic narrowing is skipped here. The editor keeps listing pages from every topic,
	// unlike the viewing screens that narrow the listing down to the topics the viewer may open.
	// Aligning the editor with that rule is left to a follow-up task, since it would change what an
	// editing member sees mid-migration.
	//
	// The trash and discarded-topic filters live in the queries themselves, so they apply to the
	// editor as well, matching the Rails `available` scope behind the same listing.
	//
	// [Ja] ここで省略するのはトピックの絞り込みだけである。編集画面は閲覧画面と違い、全トピックの
	// ページを一覧し続ける。閲覧画面と同じく開けるトピックに絞る対応は、移行の途中で編集中の
	// メンバーの見え方を変えることになるため後続タスクに回している。
	//
	// ゴミ箱と廃棄済みトピックのフィルタはクエリ自体に含まれるため編集画面にも効き、同じ一覧を
	// 担う Rails 版の `available` スコープと揃う。
	visibility := repository.AllTopicsVisible()

	lists, err := fetchRelatedPageLists(ctx, uc.pageRepo, relatedPageListInput{
		PageID:                 input.Page.ID,
		LinkedPageIDs:          linkedPageIDs,
		SpaceID:                input.SpaceID,
		Visibility:             visibility,
		LinkPage:               input.CurrentPage,
		LinkLimit:              input.LinkLimit,
		LinkedPageNumber:       input.LinkedPageNumber,
		LinkedPageBacklinkPage: input.LinkedPageBacklinkPage,
		BacklinkLimit:          input.BacklinkLimit,
		PageBacklinkPage:       input.PageBacklinkPage,
		PageBacklinkLimit:      input.PageBacklinkLimit,
		IncludePrecedingPages:  input.IncludePrecedingPages,
	})
	if err != nil {
		return nil, err
	}

	// すべてのページのTopicIDを収集してトピックを一括取得
	topicIDs := collectTopicIDsFromPages(lists.pageGroups...)
	topics, err := uc.topicRepo.FindByIDsAndSpace(ctx, topicIDs, input.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("トピックの一括取得に失敗: %w", err)
	}

	return &GetEditLinkDataOutput{
		LinkedPages:       lists.linkedPages,
		LinkedTotalCount:  lists.linkedTotalCount,
		BacklinksPerPage:  lists.backlinksPerPage,
		PageBacklinks:     lists.pageBacklinks,
		PageBacklinkCount: lists.pageBacklinkCount,
		LinkTopics:        topics,
	}, nil
}
