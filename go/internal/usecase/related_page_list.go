package usecase

import (
	"context"
	"fmt"
	"math"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// MaxCumulativeRelatedPagePagesは、下書き再取得が先頭からまとめて取得する各関連ページ一覧を
// 10ページまでに制限する。十分な読み込み済み範囲を維持しつつ、リンク一覧・各カードのバックリンク・
// ページ自身のバックリンクに使うクエリ量と描画量を予測可能に保つためである。
const MaxCumulativeRelatedPagePages int32 = 10

// LinkedPageBacklinksは一覧した1ページ分のバックリンクと総件数のペア。リンク一覧は編集画面と
// ページ表示画面の双方が描画するため、両者を担うUseCaseで共有する。
type LinkedPageBacklinks struct {
	Pages      []*model.Page
	TotalCount int64
}

// relatedPageListInputは、編集画面とページ表示画面が同じく必要とする3つの関連ページ一覧
// (リンク一覧・一覧した各ページのネストしたバックリンク一覧・ページ自身のバックリンク一覧) の解決に
// 必要な入力を保持する。
//
// LinkPage / LinkedPageBacklinkPage / PageBacklinkPageは1始まりで、呼び出し元が解決済みの値を渡す。
// LinkedPageNumberは、どのカードのネストしたバックリンク一覧も進めていないときにゼロになる。
type relatedPageListInput struct {
	PageID                 model.PageID
	LinkedPageIDs          []model.PageID
	SpaceID                model.SpaceID
	Visibility             repository.TopicVisibility
	LinkPage               int32
	LinkLimit              int32
	LinkedPageNumber       int32
	LinkedPageBacklinkPage int32
	BacklinkLimit          int32
	PageBacklinkPage       int32
	PageBacklinkLimit      int32

	// IncludePrecedingPagesは各一覧が、要求ページだけではなく1ページ目から要求ページまでを
	// 1つのスライスで返すようにする (listingWindowを参照)。
	IncludePrecedingPages bool
}

// relatedPageListsは解決した3つの関連ページ一覧の範囲を保持する。
type relatedPageLists struct {
	linkedPages       []*model.Page
	linkedTotalCount  int64
	backlinksPerPage  map[model.PageID]*LinkedPageBacklinks
	pageBacklinks     []*model.Page
	pageBacklinkCount int64

	// pageGroupsは上記のページスライスをまとめ、呼び出し元がトピックを一度に解決できるようにする。
	pageGroups [][]*model.Page
}

// fetchRelatedPageListsは1組のページ番号から3つの関連ページ一覧を解決する。編集画面と
// ページ表示画面の違いはリンク集合の出どころ (編集中の下書きか保存済みページか) と一覧の絞り込みだけ
// なので、その2つを引数で受け取り、残りは共有する。
func fetchRelatedPageLists(ctx context.Context, pageRepo *repository.PageRepository, input relatedPageListInput) (*relatedPageLists, error) {
	if input.IncludePrecedingPages && !cumulativeRelatedPagePagesInRange(input) {
		return nil, fmt.Errorf("関連ページ一覧の累積取得ページ数が上限 %dを超えている", MaxCumulativeRelatedPagePages)
	}

	listing, err := fetchLinkedPageListing(ctx, pageRepo, linkedPageListingInput{
		PageID:                input.PageID,
		LinkedPageIDs:         input.LinkedPageIDs,
		SpaceID:               input.SpaceID,
		Visibility:            input.Visibility,
		LinkPage:              input.LinkPage,
		LinkLimit:             input.LinkLimit,
		BacklinkLimit:         input.BacklinkLimit,
		IncludePrecedingPages: input.IncludePrecedingPages,
	})
	if err != nil {
		return nil, err
	}

	// 差し替えるのは選択したカードのネストしたバックリンクだけで、それ以外のカードは一括クエリが
	// 返した1ページ目を維持する。1ページ目は一括クエリが既に返しているため差し替え不要で、
	// 2ページ目以降だけがもう一度の往復に見合う。
	if listing != nil && input.LinkedPageNumber > 0 && input.LinkedPageBacklinkPage > 1 {
		for _, linkedPage := range listing.paginatedLinks.Pages {
			if int32(linkedPage.Number) != input.LinkedPageNumber {
				continue
			}

			nestedOffset, nestedLimit := listingWindow(input.LinkedPageBacklinkPage, input.BacklinkLimit, input.IncludePrecedingPages)
			paginated, err := pageRepo.FindBacklinkedPagesPaginated(ctx, linkedPage.ID, input.SpaceID, input.Visibility, nestedOffset, nestedLimit, listing.excludePageIDs)
			if err != nil {
				return nil, fmt.Errorf("バックリンクの取得に失敗: %w", err)
			}
			listing.backlinks[linkedPage.ID] = paginated
			break
		}
	}

	pageBacklinkOffset, pageBacklinkLimit := listingWindow(input.PageBacklinkPage, input.PageBacklinkLimit, input.IncludePrecedingPages)
	paginatedBacklinks, err := pageRepo.FindBacklinkedPagesPaginated(ctx, input.PageID, input.SpaceID, input.Visibility, pageBacklinkOffset, pageBacklinkLimit, nil)
	if err != nil {
		return nil, fmt.Errorf("ページレベルのバックリンクの取得に失敗: %w", err)
	}

	lists := &relatedPageLists{
		pageBacklinks:     paginatedBacklinks.Pages,
		pageBacklinkCount: paginatedBacklinks.TotalCount,
		backlinksPerPage:  map[model.PageID]*LinkedPageBacklinks{},
	}

	if listing != nil {
		lists.linkedPages = listing.paginatedLinks.Pages
		lists.linkedTotalCount = listing.paginatedLinks.TotalCount
		lists.pageGroups = append(lists.pageGroups, listing.paginatedLinks.Pages)

		backlinksPerPage, backlinkGroups := newLinkedPageBacklinksMap(listing.backlinks)
		lists.backlinksPerPage = backlinksPerPage
		lists.pageGroups = append(lists.pageGroups, backlinkGroups...)
	}

	lists.pageGroups = append(lists.pageGroups, paginatedBacklinks.Pages)

	return lists, nil
}

// linkedPageListingInputはfetchLinkedPageListingに必要な入力を保持する。各フィールドの意味は
// relatedPageListInputと同じ。
type linkedPageListingInput struct {
	PageID                model.PageID
	LinkedPageIDs         []model.PageID
	SpaceID               model.SpaceID
	Visibility            repository.TopicVisibility
	LinkPage              int32
	LinkLimit             int32
	BacklinkLimit         int32
	IncludePrecedingPages bool
}

// linkedPageListingは、リンク一覧の1範囲と、そこに載る各ページのバックリンク、およびその
// バックリンクを絞り込むのに使ったPageIDをまとめたもの。
type linkedPageListing struct {
	paginatedLinks *repository.PaginatedPages
	backlinks      map[model.PageID]*repository.PaginatedPages
	excludePageIDs []model.PageID
}

// fetchLinkedPageListingはリンク一覧と、そこに載る各ページのバックリンクを2回の一括クエリで
// 解決する。ページがどこにもリンクしていない場合はnilを返す。
//
// 2画面の初回描画とリンク一覧の続きを返すフラグメントがいずれもここを通るため、可視性の絞り込みと、
// バックリンクの隣に既に描画するページの除外が1箇所に集まる。この2つは閲覧者が開けないページを
// 一覧から締め出すためのもので、画面ごとに解決が食い違うと、単に表示がずれるのではなくタイトルが
// 漏れることになる。
func fetchLinkedPageListing(ctx context.Context, pageRepo *repository.PageRepository, input linkedPageListingInput) (*linkedPageListing, error) {
	if len(input.LinkedPageIDs) == 0 {
		return nil, nil
	}

	linkOffset, linkLimit := listingWindow(input.LinkPage, input.LinkLimit, input.IncludePrecedingPages)

	paginatedLinks, err := pageRepo.FindLinkedPagesPaginated(ctx, input.LinkedPageIDs, input.SpaceID, input.Visibility, linkOffset, linkLimit)
	if err != nil {
		return nil, fmt.Errorf("リンク先ページの取得に失敗: %w", err)
	}

	excludePageIDs := buildExcludePageIDs(input.PageID, paginatedLinks.Pages)

	backlinks, err := pageRepo.FindBacklinksForPages(ctx, paginatedLinks.Pages, input.SpaceID, input.Visibility, input.BacklinkLimit, excludePageIDs)
	if err != nil {
		return nil, fmt.Errorf("バックリンクの取得に失敗: %w", err)
	}

	return &linkedPageListing{
		paginatedLinks: paginatedLinks,
		backlinks:      backlinks,
		excludePageIDs: excludePageIDs,
	}, nil
}

// newLinkedPageBacklinksMapはRepositoryのページングされたバックリンクをUseCaseが公開する型へ
// 変換し、呼び出し元がトピックを一度に解決できるようページ範囲も集める。
func newLinkedPageBacklinksMap(paginatedMap map[model.PageID]*repository.PaginatedPages) (map[model.PageID]*LinkedPageBacklinks, [][]*model.Page) {
	backlinksPerPage := make(map[model.PageID]*LinkedPageBacklinks, len(paginatedMap))
	pageGroups := make([][]*model.Page, 0, len(paginatedMap))

	for pageID, paginated := range paginatedMap {
		backlinksPerPage[pageID] = &LinkedPageBacklinks{
			Pages:      paginated.Pages,
			TotalCount: paginated.TotalCount,
		}
		pageGroups = append(pageGroups, paginated.Pages)
	}

	return backlinksPerPage, pageGroups
}

// listingWindowは1つの関連ページ一覧を取得するoffsetと件数上限を解決する。初回ページは
// initialLimit件、後続ページはそれより1件多くする。現在の3カラムグリッドでは累積カード数が
// 14、29、44と増えるため、追記後も「もっと見る」タイルが完成した最終行の末尾に留まる。
//
// 一覧のコンテナごと差し替える呼び出し元は、1ページ目から要求ページまでをまとめて要求する。閲覧者が
// htmxで後続ページを追記している可能性があり、要求ページだけでコンテナを描画し直すと残りが消えて
// しまうためである。一覧を最初から描画する呼び出し元は要求ページだけを取る。
//
// fetchRelatedPageListsは、この計算より前にMaxCumulativeRelatedPagePagesを超える累積ページ番号を
// 拒否する。math.MaxInt32の頭打ちは、このヘルパーが誤って再利用されても件数上限やoffsetが負値へ
// 回り込まないための局所的な算術防御として残す。
func listingWindow(page, initialLimit int32, includePrecedingPages bool) (int32, int32) {
	if initialLimit <= 0 {
		return 0, 0
	}
	if page <= 1 {
		return 0, initialLimit
	}

	followingLimit := int64(initialLimit) + 1
	if followingLimit > math.MaxInt32 {
		followingLimit = math.MaxInt32
	}

	span := int64(initialLimit) + int64(page-1)*followingLimit
	if span > math.MaxInt32 {
		span = math.MaxInt32
	}

	if includePrecedingPages {
		return 0, int32(span)
	}
	offset := int64(initialLimit) + int64(page-2)*followingLimit
	if offset > math.MaxInt32 {
		offset = math.MaxInt32
	}

	return int32(offset), int32(followingLimit)
}

// cumulativeRelatedPagePagesInRangeは、累積再取得がRepositoryへ到達する前に、独立して
// ページングする3つの一覧をすべて検査する。
func cumulativeRelatedPagePagesInRange(input relatedPageListInput) bool {
	return input.LinkPage <= MaxCumulativeRelatedPagePages &&
		input.LinkedPageBacklinkPage <= MaxCumulativeRelatedPagePages &&
		input.PageBacklinkPage <= MaxCumulativeRelatedPagePages
}

// buildExcludePageIDsは、バックリンク結果から除外するPageIDを構築する。対象は元ページ自身と、
// バックリンクの隣に既に描画するリンク先ページである。
func buildExcludePageIDs(currentPageID model.PageID, linkedPages []*model.Page) []model.PageID {
	ids := make([]model.PageID, 0, 1+len(linkedPages))
	ids = append(ids, currentPageID)
	for _, p := range linkedPages {
		ids = append(ids, p.ID)
	}
	return ids
}

// collectTopicIDsFromPagesは複数のページ範囲を描画するために必要な一意のTopicIDを収集する。
func collectTopicIDsFromPages(pageSlices ...[]*model.Page) []model.TopicID {
	topicIDSet := make(map[model.TopicID]struct{})
	for _, pages := range pageSlices {
		for _, p := range pages {
			topicIDSet[p.TopicID] = struct{}{}
		}
	}
	topicIDs := make([]model.TopicID, 0, len(topicIDSet))
	for id := range topicIDSet {
		topicIDs = append(topicIDs, id)
	}
	return topicIDs
}
