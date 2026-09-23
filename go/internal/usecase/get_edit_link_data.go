package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetEditLinkDataUsecaseは編集画面のリンクデータ取得ユースケース
type GetEditLinkDataUsecase struct {
	pageRepo  *repository.PageRepository
	topicRepo *repository.TopicRepository
}

// NewGetEditLinkDataUsecaseはGetEditLinkDataUsecaseを生成する
func NewGetEditLinkDataUsecase(
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
) *GetEditLinkDataUsecase {
	return &GetEditLinkDataUsecase{
		pageRepo:  pageRepo,
		topicRepo: topicRepo,
	}
}

// GetEditLinkDataInputは編集画面の関連ページページネーション入力を保持する。
//
// CurrentPage / LinkedPageBacklinkPage / PageBacklinkPageは1始まりで、呼び出し元が解決済みの値を
// 渡す (Handlerはviewmodel.PageLinkState.Normalizedを使う)。ここで取得する範囲と一覧が描画する
// ページ番号を同じ値から導くためである。LinkedPageNumberは、どのカードのネストしたバックリンク
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

	// IncludePrecedingPagesは各一覧が1ページ目から要求ページまでを返すようにする。下書き
	// 再取得は一覧のコンテナごと差し替えるため、要求ページに加えて閲覧者がhtmxで追記した範囲も
	// 描画し直す必要があり、これを立てる。一覧を最初から描画する画面ではfalseのままにする。
	IncludePrecedingPages bool
}

// GetEditLinkDataOutputは編集画面のリンクデータ取得の出力
type GetEditLinkDataOutput struct {
	LinkedPages       []*model.Page
	LinkedTotalCount  int64
	BacklinksPerPage  map[model.PageID]*LinkedPageBacklinks
	PageBacklinks     []*model.Page
	PageBacklinkCount int64
	LinkTopics        []*model.Topic
}

// Executeは編集画面のリンク・バックリンクデータを取得する
func (uc *GetEditLinkDataUsecase) Execute(ctx context.Context, input GetEditLinkDataInput) (*GetEditLinkDataOutput, error) {
	var linkedPageIDs []model.PageID
	if input.DraftPage != nil {
		linkedPageIDs = input.DraftPage.LinkedPageIDs
	} else {
		linkedPageIDs = input.Page.LinkedPageIDs
	}

	// ここで省略するのはトピックの絞り込みだけである。編集画面は閲覧画面と違い、全トピックの
	// ページを一覧し続ける。閲覧画面と同じく開けるトピックに絞る対応は、移行の途中で編集中の
	// メンバーの見え方を変えることになるため後続タスクに回している。
	//
	// ゴミ箱と廃棄済みトピックのフィルタはクエリ自体に含まれるため編集画面にも効き、同じ一覧を
	// 担うRails版の `available` スコープと揃う。
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
