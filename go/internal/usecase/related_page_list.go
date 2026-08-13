package usecase

import (
	"context"
	"fmt"
	"math"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// MaxCumulativeRelatedPagePages bounds each related-page listing that a draft refresh fetches from
// its first page. Ten pages preserve a useful loaded range while keeping the link list, its
// per-card backlinks, and the page backlinks under a predictable query and render budget.
//
// [Ja] MaxCumulativeRelatedPagePages は、下書き再取得が先頭からまとめて取得する各関連ページ一覧を
// 10 ページまでに制限する。十分な読み込み済み範囲を維持しつつ、リンク一覧・各カードのバックリンク・
// ページ自身のバックリンクに使うクエリ量と描画量を予測可能に保つためである。
const MaxCumulativeRelatedPagePages int32 = 10

// LinkedPageBacklinks pairs the backlinks of one listed page with their total count. The link list
// is rendered by the editor and by the page detail screen alike, so the type is shared by the
// usecases behind both.
//
// [Ja] LinkedPageBacklinks は一覧した 1 ページ分のバックリンクと総件数のペア。リンク一覧は編集画面と
// ページ表示画面の双方が描画するため、両者を担う UseCase で共有する。
type LinkedPageBacklinks struct {
	Pages      []*model.Page
	TotalCount int64
}

// relatedPageListInput holds what the editor and the page detail screen alike need to resolve the
// three related-page listings: the link list, the nested backlink list of each listed page, and the
// page's own backlink list.
//
// LinkPage / LinkedPageBacklinkPage / PageBacklinkPage are one-based and already resolved by the
// caller. LinkedPageNumber is zero when no card's nested backlink list is being advanced.
//
// [Ja] relatedPageListInput は、編集画面とページ表示画面が同じく必要とする 3 つの関連ページ一覧
// (リンク一覧・一覧した各ページのネストしたバックリンク一覧・ページ自身のバックリンク一覧) の解決に
// 必要な入力を保持する。
//
// LinkPage / LinkedPageBacklinkPage / PageBacklinkPage は 1 始まりで、呼び出し元が解決済みの値を渡す。
// LinkedPageNumber は、どのカードのネストしたバックリンク一覧も進めていないときにゼロになる。
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

	// IncludePrecedingPages makes each listing return every page from the first through the
	// requested one in a single slice, instead of the requested page alone. See listingSlice.
	//
	// [Ja] IncludePrecedingPages は各一覧が、要求ページだけではなく 1 ページ目から要求ページまでを
	// 1 つのスライスで返すようにする (listingSlice を参照)。
	IncludePrecedingPages bool
}

// relatedPageLists holds the resolved slices of the three related-page listings.
//
// [Ja] relatedPageLists は解決した 3 つの関連ページ一覧の範囲を保持する。
type relatedPageLists struct {
	linkedPages       []*model.Page
	linkedTotalCount  int64
	backlinksPerPage  map[model.PageID]*LinkedPageBacklinks
	pageBacklinks     []*model.Page
	pageBacklinkCount int64

	// pageGroups collects every page slice above so the caller can resolve their topics in one go.
	//
	// [Ja] pageGroups は上記のページスライスをまとめ、呼び出し元がトピックを一度に解決できるようにする。
	pageGroups [][]*model.Page
}

// fetchRelatedPageLists resolves the three related-page listings from one set of page numbers. The
// editor and the page detail screen differ only in where the link set comes from (an editable draft
// or the saved page) and in how far the listings are narrowed down, so both pass those in and share
// everything else.
//
// [Ja] fetchRelatedPageLists は 1 組のページ番号から 3 つの関連ページ一覧を解決する。編集画面と
// ページ表示画面の違いはリンク集合の出どころ (編集中の下書きか保存済みページか) と一覧の絞り込みだけ
// なので、その 2 つを引数で受け取り、残りは共有する。
func fetchRelatedPageLists(ctx context.Context, pageRepo *repository.PageRepository, input relatedPageListInput) (*relatedPageLists, error) {
	if input.IncludePrecedingPages && !cumulativeRelatedPagePagesInRange(input) {
		return nil, fmt.Errorf("関連ページ一覧の累積取得ページ数が上限 %d を超えている", MaxCumulativeRelatedPagePages)
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

	// Only the selected card's nested backlinks are replaced; every other card keeps the first page
	// returned by the bulk query. The first page needs no replacement because the bulk query already
	// returned it, so only a later page is worth another round trip.
	//
	// [Ja] 差し替えるのは選択したカードのネストしたバックリンクだけで、それ以外のカードは一括クエリが
	// 返した 1 ページ目を維持する。1 ページ目は一括クエリが既に返しているため差し替え不要で、
	// 2 ページ目以降だけがもう一度の往復に見合う。
	if listing != nil && input.LinkedPageNumber > 0 && input.LinkedPageBacklinkPage > 1 {
		for _, linkedPage := range listing.paginatedLinks.Pages {
			if int32(linkedPage.Number) != input.LinkedPageNumber {
				continue
			}

			nestedPage, nestedLimit := listingSlice(input.LinkedPageBacklinkPage, input.BacklinkLimit, input.IncludePrecedingPages)
			paginated, err := pageRepo.FindBacklinkedPagesPaginated(ctx, linkedPage.ID, input.SpaceID, input.Visibility, nestedPage, nestedLimit, listing.excludePageIDs)
			if err != nil {
				return nil, fmt.Errorf("バックリンクの取得に失敗: %w", err)
			}
			listing.backlinks[linkedPage.ID] = paginated
			break
		}
	}

	pageBacklinkPage, pageBacklinkLimit := listingSlice(input.PageBacklinkPage, input.PageBacklinkLimit, input.IncludePrecedingPages)
	paginatedBacklinks, err := pageRepo.FindBacklinkedPagesPaginated(ctx, input.PageID, input.SpaceID, input.Visibility, pageBacklinkPage, pageBacklinkLimit, nil)
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

// linkedPageListingInput holds what fetchLinkedPageListing needs. The fields mean the same as in
// relatedPageListInput.
//
// [Ja] linkedPageListingInput は fetchLinkedPageListing に必要な入力を保持する。各フィールドの意味は
// relatedPageListInput と同じ。
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

// linkedPageListing is one slice of the link list together with the backlinks of every page it
// lists, and the page IDs those backlinks were filtered by.
//
// [Ja] linkedPageListing は、リンク一覧の 1 範囲と、そこに載る各ページのバックリンク、およびその
// バックリンクを絞り込むのに使った PageID をまとめたもの。
type linkedPageListing struct {
	paginatedLinks *repository.PaginatedPages
	backlinks      map[model.PageID]*repository.PaginatedPages
	excludePageIDs []model.PageID
}

// fetchLinkedPageListing resolves the link list and the backlinks of each listed page in two bulk
// queries, and returns nil when the page links nowhere.
//
// The initial render of both screens and the fragment that continues the link list all go through
// here, so the visibility filter and the exclusion of pages already rendered beside their backlinks
// stay in one place. Those two are what keeps a page the viewer may not open out of the listing, so
// a screen that resolved them differently from another would leak titles rather than merely look
// inconsistent.
//
// [Ja] fetchLinkedPageListing はリンク一覧と、そこに載る各ページのバックリンクを 2 回の一括クエリで
// 解決する。ページがどこにもリンクしていない場合は nil を返す。
//
// 2 画面の初回描画とリンク一覧の続きを返すフラグメントがいずれもここを通るため、可視性の絞り込みと、
// バックリンクの隣に既に描画するページの除外が 1 箇所に集まる。この 2 つは閲覧者が開けないページを
// 一覧から締め出すためのもので、画面ごとに解決が食い違うと、単に表示がずれるのではなくタイトルが
// 漏れることになる。
func fetchLinkedPageListing(ctx context.Context, pageRepo *repository.PageRepository, input linkedPageListingInput) (*linkedPageListing, error) {
	if len(input.LinkedPageIDs) == 0 {
		return nil, nil
	}

	linkPage, linkLimit := listingSlice(input.LinkPage, input.LinkLimit, input.IncludePrecedingPages)

	paginatedLinks, err := pageRepo.FindLinkedPagesPaginated(ctx, input.LinkedPageIDs, input.SpaceID, input.Visibility, linkPage, linkLimit)
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

// newLinkedPageBacklinksMap converts the repository's paginated backlinks into the type the
// usecases expose, and collects the page slices so the caller can resolve their topics in one go.
//
// [Ja] newLinkedPageBacklinksMap は Repository のページングされたバックリンクを UseCase が公開する型へ
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

// listingSlice resolves the page and limit one listing is fetched with.
//
// A caller that replaces a whole listing container asks for every page from the first through the
// requested one, because the reader may have appended later pages through htmx and re-rendering the
// container from the requested page alone would drop the rest. Callers that render a listing from
// scratch take the requested page only.
//
// fetchRelatedPageLists rejects cumulative page numbers above MaxCumulativeRelatedPagePages before
// reaching this calculation. The math.MaxInt32 saturation remains as a local arithmetic safeguard
// so this helper cannot wrap a limit negative if reused incorrectly.
//
// [Ja] listingSlice は 1 つの一覧を取得するページと件数上限を解決する。
//
// 一覧のコンテナごと差し替える呼び出し元は、1 ページ目から要求ページまでをまとめて要求する。閲覧者が
// htmx で後続ページを追記している可能性があり、要求ページだけでコンテナを描画し直すと残りが消えて
// しまうためである。一覧を最初から描画する呼び出し元は要求ページだけを取る。
//
// fetchRelatedPageLists は、この計算より前に MaxCumulativeRelatedPagePages を超える累積ページ番号を
// 拒否する。math.MaxInt32 の頭打ちは、このヘルパーが誤って再利用されても件数上限が負値へ回り込まない
// ための局所的な算術防御として残す。
func listingSlice(page, limit int32, includePrecedingPages bool) (int32, int32) {
	if !includePrecedingPages {
		return page, limit
	}

	span := int64(page) * int64(limit)
	if span > math.MaxInt32 {
		span = math.MaxInt32
	}

	return 1, int32(span)
}

// cumulativeRelatedPagePagesInRange checks all three independently paginated listings before a
// cumulative refresh reaches the repositories.
//
// [Ja] cumulativeRelatedPagePagesInRange は、累積再取得が Repository へ到達する前に、独立して
// ページングする 3 つの一覧をすべて検査する。
func cumulativeRelatedPagePagesInRange(input relatedPageListInput) bool {
	return input.LinkPage <= MaxCumulativeRelatedPagePages &&
		input.LinkedPageBacklinkPage <= MaxCumulativeRelatedPagePages &&
		input.PageBacklinkPage <= MaxCumulativeRelatedPagePages
}

// buildExcludePageIDs builds the page IDs excluded from backlink results: the source page and the
// linked pages already rendered beside those backlinks.
//
// [Ja] buildExcludePageIDs は、バックリンク結果から除外する PageID を構築する。対象は元ページ自身と、
// バックリンクの隣に既に描画するリンク先ページである。
func buildExcludePageIDs(currentPageID model.PageID, linkedPages []*model.Page) []model.PageID {
	ids := make([]model.PageID, 0, 1+len(linkedPages))
	ids = append(ids, currentPageID)
	for _, p := range linkedPages {
		ids = append(ids, p.ID)
	}
	return ids
}

// collectTopicIDsFromPages collects the unique topic IDs needed to render several page slices.
//
// [Ja] collectTopicIDsFromPages は複数のページ範囲を描画するために必要な一意の TopicID を収集する。
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
