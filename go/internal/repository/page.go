package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// PageRepositoryはページリポジトリ
type PageRepository struct {
	q *query.Queries
}

// NewPageRepositoryはPageRepositoryを生成する
func NewPageRepository(q *query.Queries) *PageRepository {
	return &PageRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *PageRepository) WithTx(tx *sql.Tx) *PageRepository {
	return &PageRepository{q: r.q.WithTx(tx)}
}

// ReassignLinkedPageIDは、スペースのページのリンクの記録で、oldPageIDのページへのリンクを
// newPageIDのページへ付け替える。本文と更新日時は変えない
func (r *PageRepository) ReassignLinkedPageID(ctx context.Context, spaceID model.SpaceID, oldPageID, newPageID model.PageID) error {
	return r.q.ReassignPageLinkedPageID(ctx, query.ReassignPageLinkedPageIDParams{
		SpaceID:   string(spaceID),
		OldPageID: string(oldPageID),
		NewPageID: string(newPageID),
	})
}

// FindBySpaceAndNumberはスペースIDとページ番号でページを取得する (廃棄されていないページのみ)
func (r *PageRepository) FindBySpaceAndNumber(ctx context.Context, spaceID model.SpaceID, number model.PageNumber) (*model.Page, error) {
	row, err := r.q.FindPageBySpaceAndNumber(ctx, query.FindPageBySpaceAndNumberParams{
		SpaceID: string(spaceID),
		Number:  int32(number),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByIDForUpdateはページを取得し、呼び出し元のトランザクションが終わるまでその行をロックする
// (廃棄されていないページのみ)。ページが存在しない場合はnilを返す
func (r *PageRepository) FindByIDForUpdate(ctx context.Context, id model.PageID, spaceID model.SpaceID) (*model.Page, error) {
	row, err := r.q.FindPageByIDForUpdate(ctx, query.FindPageByIDForUpdateParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByIDForShareはページを取得し、呼び出し元のトランザクションが終わるまでその行を共有ロックする
// (廃棄・ゴミ箱のページも含む)。ページが存在しない場合はnilを返す
func (r *PageRepository) FindByIDForShare(ctx context.Context, id model.PageID, spaceID model.SpaceID) (*model.Page, error) {
	row, err := r.q.FindPageByIDForShare(ctx, query.FindPageByIDForShareParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindPinnedByTopicはトピック内のピン留めページを取得する (公開済み・未廃棄・未ゴミ箱のページのみ、pinned_at DESCでソート)
func (r *PageRepository) FindPinnedByTopic(ctx context.Context, topicID model.TopicID, spaceID model.SpaceID) ([]*model.Page, error) {
	rows, err := r.q.FindPinnedPagesByTopic(ctx, query.FindPinnedPagesByTopicParams{
		TopicID: string(topicID),
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// FindRegularByTopicPaginatedはトピック内の通常ページをオフセットページネーションで取得する (ピン留めなし・公開済み・未廃棄・未ゴミ箱のページのみ)
func (r *PageRepository) FindRegularByTopicPaginated(ctx context.Context, topicID model.TopicID, spaceID model.SpaceID, page int32, limit int32) (*PaginatedPages, error) {
	totalCount, err := r.q.CountRegularPagesByTopic(ctx, query.CountRegularPagesByTopicParams{
		TopicID: string(topicID),
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	rows, err := r.q.FindRegularPagesByTopicPaginated(ctx, query.FindRegularPagesByTopicPaginatedParams{
		TopicID: string(topicID),
		SpaceID: string(spaceID),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	return &PaginatedPages{
		Pages:      r.toModels(rows),
		TotalCount: totalCount,
	}, nil
}

// FindPinnedBySpaceはスペース内のピン留めされたアクティブなページ (公開済み・未廃棄・
// 未ゴミ箱・トピック未廃棄) をpinned_at DESC, id DESCで取得する。publicOnlyがtrueのときは
// 公開トピックのページのみを返す (非メンバー閲覧者向け)。
func (r *PageRepository) FindPinnedBySpace(ctx context.Context, spaceID model.SpaceID, publicOnly bool) ([]*model.Page, error) {
	rows, err := r.q.FindPinnedPagesBySpace(ctx, query.FindPinnedPagesBySpaceParams{
		SpaceID:    string(spaceID),
		PublicOnly: publicOnly,
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// FindRegularBySpacePaginatedはスペース内の通常ページ (ピン留めなし) をオフセット
// ページネーションで取得する。並び順はmodified_at DESC, id DESC。publicOnlyがtrueのときは
// 公開トピックのページのみを返す (非メンバー閲覧者向け)。
func (r *PageRepository) FindRegularBySpacePaginated(ctx context.Context, spaceID model.SpaceID, publicOnly bool, page int32, limit int32) (*PaginatedPages, error) {
	totalCount, err := r.q.CountRegularPagesBySpace(ctx, query.CountRegularPagesBySpaceParams{
		SpaceID:    string(spaceID),
		PublicOnly: publicOnly,
	})
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	rows, err := r.q.FindRegularPagesBySpacePaginated(ctx, query.FindRegularPagesBySpacePaginatedParams{
		SpaceID:    string(spaceID),
		PublicOnly: publicOnly,
		RowLimit:   limit,
		RowOffset:  offset,
	})
	if err != nil {
		return nil, err
	}

	return &PaginatedPages{
		Pages:      r.toModels(rows),
		TotalCount: totalCount,
	}, nil
}

// FindLinkedPagesPaginatedはページからのリンク先ページをオフセットページネーションで
// 取得する。呼び出し元が解決済みのoffsetとlimitを渡す。ゴミ箱に入ったページと廃棄済み
// トピックのページは除外し、閲覧者が開けるトピックのページに絞る (TopicVisibilityを参照)。
func (r *PageRepository) FindLinkedPagesPaginated(ctx context.Context, pageIDs []model.PageID, spaceID model.SpaceID, visibility TopicVisibility, offset int32, limit int32) (*PaginatedPages, error) {
	totalCount, err := r.q.CountLinkedPages(ctx, query.CountLinkedPagesParams{
		PageIds:          model.PageIDsToStrings(pageIDs),
		SpaceID:          string(spaceID),
		AllTopicsVisible: visibility.AllVisible,
		VisibleTopicIds:  model.TopicIDsToStrings(visibility.TopicIDs),
	})
	if err != nil {
		return nil, err
	}

	rows, err := r.q.FindLinkedPagesPaginated(ctx, query.FindLinkedPagesPaginatedParams{
		PageIds:          model.PageIDsToStrings(pageIDs),
		SpaceID:          string(spaceID),
		AllTopicsVisible: visibility.AllVisible,
		VisibleTopicIds:  model.TopicIDsToStrings(visibility.TopicIDs),
		RowLimit:         limit,
		RowOffset:        offset,
	})
	if err != nil {
		return nil, err
	}

	return &PaginatedPages{
		Pages:      r.toModels(rows),
		TotalCount: totalCount,
	}, nil
}

// FindBacklinkedPagesPaginatedは指定ページへのバックリンクをオフセットページネーションで
// 取得する。呼び出し元が解決済みのoffsetとlimitを渡す。ゴミ箱・廃棄済みトピック・
// トピック可視性の扱いはFindLinkedPagesPaginatedと同じ。
func (r *PageRepository) FindBacklinkedPagesPaginated(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, visibility TopicVisibility, offset int32, limit int32, excludePageIDs []model.PageID) (*PaginatedPages, error) {
	totalCount, err := r.q.CountBacklinkedPages(ctx, query.CountBacklinkedPagesParams{
		PageID:           string(pageID),
		SpaceID:          string(spaceID),
		AllTopicsVisible: visibility.AllVisible,
		VisibleTopicIds:  model.TopicIDsToStrings(visibility.TopicIDs),
		ExcludePageIds:   model.PageIDsToStrings(excludePageIDs),
	})
	if err != nil {
		return nil, err
	}

	rows, err := r.q.FindBacklinkedPagesPaginated(ctx, query.FindBacklinkedPagesPaginatedParams{
		PageID:           string(pageID),
		SpaceID:          string(spaceID),
		AllTopicsVisible: visibility.AllVisible,
		VisibleTopicIds:  model.TopicIDsToStrings(visibility.TopicIDs),
		ExcludePageIds:   model.PageIDsToStrings(excludePageIDs),
		RowLimit:         limit,
		RowOffset:        offset,
	})
	if err != nil {
		return nil, err
	}

	return &PaginatedPages{
		Pages:      r.toModels(rows),
		TotalCount: totalCount,
	}, nil
}

// FindBacklinksForPagesは複数ページのバックリンクを一括取得する (N+1回避)。
// フィルタ条件はFindBacklinkedPagesPaginatedと同じ。
func (r *PageRepository) FindBacklinksForPages(ctx context.Context, targetPages []*model.Page, spaceID model.SpaceID, visibility TopicVisibility, limit int32, excludePageIDs []model.PageID) (map[model.PageID]*PaginatedPages, error) {
	if len(targetPages) == 0 {
		return nil, nil
	}

	targetIDs := make([]string, len(targetPages))
	for i, pg := range targetPages {
		targetIDs[i] = string(pg.ID)
	}

	excludeIDs := model.PageIDsToStrings(excludePageIDs)

	// バックリンクページを一括取得
	rows, err := r.q.FindBacklinkedPagesForTargets(ctx, query.FindBacklinkedPagesForTargetsParams{
		TargetIds:        targetIDs,
		SpaceID:          string(spaceID),
		AllTopicsVisible: visibility.AllVisible,
		VisibleTopicIds:  model.TopicIDsToStrings(visibility.TopicIDs),
		ExcludePageIds:   excludeIDs,
		RowLimit:         limit,
	})
	if err != nil {
		return nil, err
	}

	// バックリンク件数を一括取得
	countRows, err := r.q.CountBacklinkedPagesForTargets(ctx, query.CountBacklinkedPagesForTargetsParams{
		TargetIds:        targetIDs,
		SpaceID:          string(spaceID),
		AllTopicsVisible: visibility.AllVisible,
		VisibleTopicIds:  model.TopicIDsToStrings(visibility.TopicIDs),
		ExcludePageIds:   excludeIDs,
	})
	if err != nil {
		return nil, err
	}

	// 件数マップを構築
	countMap := make(map[model.PageID]int64, len(countRows))
	for _, row := range countRows {
		var targetID string
		switch v := row.TargetID.(type) {
		case string:
			targetID = v
		case []byte:
			targetID = string(v)
		}
		countMap[model.PageID(targetID)] = row.Count
	}

	// ターゲットIDごとにバックリンクページをグループ化
	result := make(map[model.PageID]*PaginatedPages, len(targetPages))
	for _, pg := range targetPages {
		result[pg.ID] = &PaginatedPages{
			Pages:      []*model.Page{},
			TotalCount: countMap[pg.ID],
		}
	}

	for _, row := range rows {
		var targetID string
		switch v := row.TargetID.(type) {
		case string:
			targetID = v
		case []byte:
			targetID = string(v)
		}
		pageID := model.PageID(targetID)
		page := r.toModelFromTargetRow(row)
		if entry, ok := result[pageID]; ok {
			entry.Pages = append(entry.Pages, page)
		}
	}

	return result, nil
}

// toModelFromTargetRowはFindBacklinkedPagesForTargetsRowをmodel.Pageに変換する
func (r *PageRepository) toModelFromTargetRow(row query.FindBacklinkedPagesForTargetsRow) *model.Page {
	var title *string
	if row.Title != nil {
		switch v := row.Title.(type) {
		case string:
			title = &v
		case []byte:
			s := string(v)
			title = &s
		}
	}

	var publishedAt *time.Time
	if row.PublishedAt.Valid {
		publishedAt = &row.PublishedAt.Time
	}

	var trashedAt *time.Time
	if row.TrashedAt.Valid {
		trashedAt = &row.TrashedAt.Time
	}

	var pinnedAt *time.Time
	if row.PinnedAt.Valid {
		pinnedAt = &row.PinnedAt.Time
	}

	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}

	var featuredImageAttachmentID *model.AttachmentID
	if row.FeaturedImageAttachmentID != nil {
		id := model.AttachmentID(*row.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &id
	}

	return &model.Page{
		ID:                        model.PageID(row.ID),
		SpaceID:                   model.SpaceID(row.SpaceID),
		TopicID:                   model.TopicID(row.TopicID),
		Number:                    model.PageNumber(row.Number),
		Title:                     title,
		Body:                      row.Body,
		LinkedPageIDs:             model.StringsToPageIDs(row.LinkedPageIds),
		ModifiedAt:                row.ModifiedAt,
		PublishedAt:               publishedAt,
		TrashedAt:                 trashedAt,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
		PinnedAt:                  pinnedAt,
		DiscardedAt:               discardedAt,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
	}
}

// FindByIDsはIDリストに含まれるページを取得する (同スペース・公開済み・未廃棄のページのみ)
func (r *PageRepository) FindByIDs(ctx context.Context, ids []model.PageID, spaceID model.SpaceID) ([]*model.Page, error) {
	rows, err := r.q.FindPagesByIDs(ctx, query.FindPagesByIDsParams{
		Column1: model.PageIDsToStrings(ids),
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// LockByIDsは、同じスペースの未廃棄ページをID順に一括でロックする。
// 公開対象への並行する書き込みを待ってからリンクの記録を読むため、トランザクション内で使う
func (r *PageRepository) LockByIDs(ctx context.Context, ids []model.PageID, spaceID model.SpaceID) error {
	return r.q.LockPagesByIDs(ctx, query.LockPagesByIDsParams{
		Ids:     model.PageIDsToStrings(ids),
		SpaceID: string(spaceID),
	})
}

// FindMetadataByIDsは同スペースの未廃棄ページを、Wikiリンクのプレビューに必要な列だけで一括取得する。
// 未公開・ゴミ箱のページも含む。返すPageにはID・スペースID・トピックID・番号・タイトルと
// 公開・ゴミ箱・廃棄の日時だけを設定し、本文などの取得しないフィールドはゼロ値になる
func (r *PageRepository) FindMetadataByIDs(ctx context.Context, ids []model.PageID, spaceID model.SpaceID) ([]*model.Page, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.q.FindPageMetadataByIDs(ctx, query.FindPageMetadataByIDsParams{
		Ids:     model.PageIDsToStrings(ids),
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}

	pages := make([]*model.Page, 0, len(rows))
	for _, row := range rows {
		pages = append(pages, r.toModel(query.Page{
			ID:          row.ID,
			SpaceID:     row.SpaceID,
			TopicID:     row.TopicID,
			Number:      row.Number,
			Title:       row.Title,
			PublishedAt: row.PublishedAt,
			TrashedAt:   row.TrashedAt,
			DiscardedAt: row.DiscardedAt,
		}))
	}
	return pages, nil
}

// FindBacklinkedByPageIDはlinked_page_idsに指定ページIDが含まれるページを取得する (同スペース・公開済み・未廃棄のページのみ)
func (r *PageRepository) FindBacklinkedByPageID(ctx context.Context, pageID model.PageID, spaceID model.SpaceID) ([]*model.Page, error) {
	rows, err := r.q.FindBacklinkedPagesByPageID(ctx, query.FindBacklinkedPagesByPageIDParams{
		Column1: string(pageID),
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// UpdatePageInputはページ更新の入力パラメータ
type UpdatePageInput struct {
	ID                        model.PageID
	SpaceID                   model.SpaceID
	TopicID                   model.TopicID
	Title                     *string
	Body                      string
	LinkedPageIDs             []model.PageID
	ModifiedAt                time.Time
	PublishedAt               *time.Time
	FeaturedImageAttachmentID *model.AttachmentID
}

// Updateはページを更新する
func (r *PageRepository) Update(ctx context.Context, input UpdatePageInput) (*model.Page, error) {
	var publishedAt sql.NullTime
	if input.PublishedAt != nil {
		publishedAt = sql.NullTime{Time: *input.PublishedAt, Valid: true}
	}

	var featuredImageAttachmentID *string
	if input.FeaturedImageAttachmentID != nil {
		s := string(*input.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &s
	}

	row, err := r.q.UpdatePage(ctx, query.UpdatePageParams{
		ID:                        string(input.ID),
		TopicID:                   string(input.TopicID),
		Title:                     input.Title,
		Body:                      input.Body,
		LinkedPageIds:             model.PageIDsToStrings(input.LinkedPageIDs),
		ModifiedAt:                input.ModifiedAt,
		PublishedAt:               publishedAt,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		UpdatedAt:                 time.Now(),
		SpaceID:                   string(input.SpaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// MoveTopicInputはページ移動の入力パラメータ
type MoveTopicInput struct {
	ID      model.PageID
	SpaceID model.SpaceID
	TopicID model.TopicID
}

// MoveTopicはページのトピックを変更する
func (r *PageRepository) MoveTopic(ctx context.Context, input MoveTopicInput) (*model.Page, error) {
	row, err := r.q.MovePageToTopic(ctx, query.MovePageToTopicParams{
		ID:      string(input.ID),
		TopicID: string(input.TopicID),
		SpaceID: string(input.SpaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// TrashByIDは渡された時刻をtrashed_atに打刻してページをゴミ箱へ入れる。
// 時刻を引数で受け取るのは、DiscardByIDがdiscardedAtを受け取るのと同じく、時刻の決定を
// 永続化層の外 (UseCase) に置くためである。
func (r *PageRepository) TrashByID(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, trashedAt time.Time) error {
	return r.q.TrashPageByID(ctx, query.TrashPageByIDParams{
		ID:        string(pageID),
		SpaceID:   string(spaceID),
		TrashedAt: sql.NullTime{Time: trashedAt, Valid: true},
		UpdatedAt: trashedAt,
	})
}

// DiscardByIDは指定ページを論理削除する (タイトルをIDに変更し、discarded_atを設定する)
func (r *PageRepository) DiscardByID(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, discardedAt time.Time) error {
	return r.q.DiscardPageByID(ctx, query.DiscardPageByIDParams{
		ID:          string(pageID),
		SpaceID:     string(spaceID),
		DiscardedAt: sql.NullTime{Time: discardedAt, Valid: true},
		UpdatedAt:   discardedAt,
	})
}

// DiscardEmptyUnpublishedByIDは同じタイトルの空の未公開ページが残っている場合だけ論理削除する。
// 更新自体がページ行をロックするため、並行する公開が終わった後に条件を確かめ直せる。
func (r *PageRepository) DiscardEmptyUnpublishedByID(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, topicID model.TopicID, title string, discardedAt time.Time) (bool, error) {
	_, err := r.q.DiscardEmptyUnpublishedPageByID(ctx, query.DiscardEmptyUnpublishedPageByIDParams{
		ID:          string(pageID),
		SpaceID:     string(spaceID),
		TopicID:     string(topicID),
		Title:       title,
		DiscardedAt: sql.NullTime{Time: discardedAt, Valid: true},
		UpdatedAt:   discardedAt,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// FindByTopicAndTitleは指定トピック内で指定タイトルのページを取得する (廃棄済みを含む、スペースIDでスコープ)
func (r *PageRepository) FindByTopicAndTitle(ctx context.Context, topicID model.TopicID, title string, spaceID model.SpaceID) (*model.Page, error) {
	row, err := r.q.FindPageByTopicAndTitle(ctx, query.FindPageByTopicAndTitleParams{
		TopicID: string(topicID),
		Title:   title,
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// TopicPageTitleはトピックIDとページタイトルの組。Wikiリンクの解決で、本文中のリンクが指す
// ページをまとめて引くために使う。
type TopicPageTitle struct {
	TopicID model.TopicID
	Title   string
}

// LinkedPageはWikiリンクの解決に必要な値だけを持つページの射影。リンク数に比例して
// 行が増えるため、解決に使わない本文 (body) を転送しないようmodel.Pageでは返さない。
type LinkedPage struct {
	ID     model.PageID
	Number model.PageNumber
	Title  *string
}

// FindByTopicAndTitlePairsはトピックIDとタイトルの組に一致するページを1クエリで取得する
// (廃棄済みを含む、スペースIDでスコープ)。タイトルの比較はFindByTopicAndTitleと同じく
// 大文字小文字を区別しない。返すmapのキーは入力の表記を保持する。
// 組に一致するページが無い場合、その組は結果に現れない。
func (r *PageRepository) FindByTopicAndTitlePairs(ctx context.Context, pairs []TopicPageTitle, spaceID model.SpaceID) (map[TopicPageTitle]*LinkedPage, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	topicIDs := make([]string, 0, len(pairs))
	titles := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		topicIDs = append(topicIDs, string(pair.TopicID))
		titles = append(titles, pair.Title)
	}

	rows, err := r.q.FindPagesByTopicAndTitlePairs(ctx, query.FindPagesByTopicAndTitlePairsParams{
		TopicIds: topicIDs,
		Titles:   titles,
		SpaceID:  string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	pages := make(map[TopicPageTitle]*LinkedPage, len(rows))
	for _, row := range rows {
		key := TopicPageTitle{TopicID: model.TopicID(row.TopicID), Title: row.RequestedTitle}
		pages[key] = r.toLinkedPage(row)
	}
	return pages, nil
}

// toLinkedPageはFindPagesByTopicAndTitlePairsRowをLinkedPageに変換する
func (r *PageRepository) toLinkedPage(row query.FindPagesByTopicAndTitlePairsRow) *LinkedPage {
	var title *string
	if row.Title != nil {
		switch v := row.Title.(type) {
		case string:
			title = &v
		case []byte:
			s := string(v)
			title = &s
		}
	}

	return &LinkedPage{
		ID:     model.PageID(row.ID),
		Number: model.PageNumber(row.Number),
		Title:  title,
	}
}

// NextPageNumberはスペース内の次のページ番号を取得する
func (r *PageRepository) NextPageNumber(ctx context.Context, spaceID model.SpaceID) (model.PageNumber, error) {
	n, err := r.q.GetNextPageNumber(ctx, string(spaceID))
	if err != nil {
		return 0, err
	}
	return model.PageNumber(n), nil
}

// CreateLinkedPageInputはWikiリンクから参照されるページ作成の入力パラメータ
type CreateLinkedPageInput struct {
	SpaceID model.SpaceID
	TopicID model.TopicID
	Number  model.PageNumber
	Title   string
}

// CreateLinkedPageはWikiリンクから参照されるページを作成する
func (r *PageRepository) CreateLinkedPage(ctx context.Context, input CreateLinkedPageInput) (*model.Page, error) {
	now := time.Now()
	row, err := r.q.CreateUnpublishedPage(ctx, query.CreateUnpublishedPageParams{
		SpaceID:    string(input.SpaceID),
		TopicID:    string(input.TopicID),
		Number:     int32(input.Number),
		Title:      input.Title,
		ModifiedAt: now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// CreateBlankPageInputは、未公開の空ページの作成に必要な値を保持します。
type CreateBlankPageInput struct {
	SpaceID model.SpaceID
	TopicID model.TopicID
	Number  model.PageNumber
}

// CreateBlankPageはページ新規作成の入口向けに、タイトルの無い空ページを作成する。
func (r *PageRepository) CreateBlankPage(ctx context.Context, input CreateBlankPageInput) (*model.Page, error) {
	now := time.Now()
	row, err := r.q.CreateUnpublishedPage(ctx, query.CreateUnpublishedPageParams{
		SpaceID:    string(input.SpaceID),
		TopicID:    string(input.TopicID),
		Number:     int32(input.Number),
		Title:      nil,
		ModifiedAt: now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// PageLocationはページロケーション検索の結果
type PageLocation struct {
	TopicName string
	PageTitle string
}

// escapeLikePatternはPostgreSQLのLIKE特殊文字 (\, %, _) をエスケープする
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// SearchPageLocationsはスペース内のページをタイトルで検索する (Wikiリンク補完用)。
// 閲覧者が開けるトピックのページに絞る (TopicVisibilityを参照)。
// 未公開ページは、開けるトピックの公開ページかspaceMemberIDのメンバー自身の下書きからリンクされているものだけを含める。
func (r *PageRepository) SearchPageLocations(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, visibility TopicVisibility, q string) ([]PageLocation, error) {
	// 検索キーワードをスペースで分割し、各ワードをILIKEパターンに変換
	words := strings.Fields(q)
	patterns := make([]string, len(words))
	for i, word := range words {
		patterns[i] = fmt.Sprintf("%%%s%%", escapeLikePattern(word))
	}

	rows, err := r.q.SearchPageLocations(ctx, query.SearchPageLocationsParams{
		SpaceID:          string(spaceID),
		TitlePatterns:    patterns,
		AllTopicsVisible: visibility.AllVisible,
		VisibleTopicIds:  model.TopicIDsToStrings(visibility.TopicIDs),
		SpaceMemberID:    string(spaceMemberID),
	})
	if err != nil {
		return nil, err
	}

	locations := make([]PageLocation, 0, len(rows))
	for _, row := range rows {
		var title string
		if row.Title != nil {
			switch v := row.Title.(type) {
			case string:
				title = v
			case []byte:
				title = string(v)
			}
		}
		if title == "" {
			continue
		}
		locations = append(locations, PageLocation{
			TopicName: row.TopicName,
			PageTitle: title,
		})
	}

	return locations, nil
}

// toModelはquery.Pageをmodel.Pageに変換する
func (r *PageRepository) toModel(row query.Page) *model.Page {
	var title *string
	if row.Title != nil {
		switch v := row.Title.(type) {
		case string:
			title = &v
		case []byte:
			s := string(v)
			title = &s
		}
	}

	var publishedAt *time.Time
	if row.PublishedAt.Valid {
		publishedAt = &row.PublishedAt.Time
	}

	var trashedAt *time.Time
	if row.TrashedAt.Valid {
		trashedAt = &row.TrashedAt.Time
	}

	var pinnedAt *time.Time
	if row.PinnedAt.Valid {
		pinnedAt = &row.PinnedAt.Time
	}

	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}

	var featuredImageAttachmentID *model.AttachmentID
	if row.FeaturedImageAttachmentID != nil {
		id := model.AttachmentID(*row.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &id
	}

	return &model.Page{
		ID:                        model.PageID(row.ID),
		SpaceID:                   model.SpaceID(row.SpaceID),
		TopicID:                   model.TopicID(row.TopicID),
		Number:                    model.PageNumber(row.Number),
		Title:                     title,
		Body:                      row.Body,
		LinkedPageIDs:             model.StringsToPageIDs(row.LinkedPageIds),
		ModifiedAt:                row.ModifiedAt,
		PublishedAt:               publishedAt,
		TrashedAt:                 trashedAt,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
		PinnedAt:                  pinnedAt,
		DiscardedAt:               discardedAt,
		FeaturedImageAttachmentID: featuredImageAttachmentID,
	}
}

// toModelsはquery.Pageのスライスをmodel.Pageのスライスに変換する
func (r *PageRepository) toModels(rows []query.Page) []*model.Page {
	pages := make([]*model.Page, len(rows))
	for i, row := range rows {
		pages[i] = r.toModel(row)
	}
	return pages
}

// ListActiveBySpaceはスペース内のアクティブなページ (公開済み・未廃棄・未ゴミ箱・トピック
// 未廃棄) をすべて、トピック順・ページ番号順で返す。エクスポートはスペース全体を対象にするため、
// ページを1ページずつではなく1回の読み取りで取得する。
func (r *PageRepository) ListActiveBySpace(ctx context.Context, spaceID model.SpaceID) ([]*model.Page, error) {
	rows, err := r.q.ListActivePagesBySpace(ctx, string(spaceID))
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListByModifiedAtInputはListByModifiedAtの入力
type ListByModifiedAtInput struct {
	SpaceID model.SpaceID
	// TopicIDsはこのトピックのページだけを返す。VisibleTopicIDsに含まれるトピックに限る
	TopicIDs []model.TopicID
	// VisibleTopicIDsは閲覧者が開けるトピック。未公開のページのリンク元を探す範囲になる
	VisibleTopicIDs []model.TopicID
	// SpaceMemberIDは閲覧者のメンバー。このメンバーの下書きからリンクされた未公開のページも返す
	SpaceMemberID model.SpaceMemberID
	// IncludePublishedとIncludeUnpublishedは、公開済みのページと未公開のページを含めるかどうか
	IncludePublished   bool
	IncludeUnpublished bool
	// Titleはこのタイトルのページに絞る (大文字小文字を区別しない)。nilなら絞らない
	Title *string
	// Keywordsはすべてのキーワードをタイトルか本文に含むページに絞る (大文字小文字を区別しない部分一致)。
	// LIKEの特殊文字も1つの文字として探す。空なら絞らない
	Keywords []string
	// ModifiedSinceはこの日時以降に更新されたページに絞る。nilなら絞らない
	ModifiedSince *time.Time
	// AfterModifiedAtとAfterIDは前のページの最後のページの位置。どちらもnilなら先頭から返す
	AfterModifiedAt *time.Time
	AfterID         *model.PageID
	Limit           int32
}

// ListByModifiedAtは、未廃棄・未ゴミ箱で閲覧者が開けるトピックのページを、
// 更新日時の新しい順 (同じ日時ならIDの大きい順) にLimit件まで返す。
// 未公開のページは、開けるトピックの公開ページかSpaceMemberIDのメンバー自身の下書きからリンクされているものだけを含める。
// pages.modified_atはタイムゾーンを持たずUTCで保存しているため、日時はUTCに直して渡す
func (r *PageRepository) ListByModifiedAt(ctx context.Context, input ListByModifiedAtInput) ([]*model.Page, error) {
	var title sql.NullString
	if input.Title != nil {
		title = sql.NullString{String: *input.Title, Valid: true}
	}
	keywordPatterns := make([]string, len(input.Keywords))
	for i, keyword := range input.Keywords {
		keywordPatterns[i] = fmt.Sprintf("%%%s%%", escapeLikePattern(keyword))
	}
	var modifiedSince sql.NullTime
	if input.ModifiedSince != nil {
		modifiedSince = sql.NullTime{Time: input.ModifiedSince.UTC(), Valid: true}
	}
	afterModifiedAt, afterID := pageCursorParams(input.AfterModifiedAt, input.AfterID)

	rows, err := r.q.ListPagesByModifiedAt(ctx, query.ListPagesByModifiedAtParams{
		SpaceID:            string(input.SpaceID),
		TopicIds:           model.TopicIDsToStrings(input.TopicIDs),
		IncludePublished:   input.IncludePublished,
		IncludeUnpublished: input.IncludeUnpublished,
		VisibleTopicIds:    model.TopicIDsToStrings(input.VisibleTopicIDs),
		SpaceMemberID:      string(input.SpaceMemberID),
		Title:              title,
		KeywordPatterns:    keywordPatterns,
		ModifiedSince:      modifiedSince,
		AfterModifiedAt:    afterModifiedAt,
		AfterID:            afterID,
		RowLimit:           input.Limit,
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListLinkedByModifiedAtInputはListLinkedByModifiedAtの入力
type ListLinkedByModifiedAtInput struct {
	SpaceID model.SpaceID
	// PageIDsはリンク元のページの保存済みのリンク先 (linked_page_ids)
	PageIDs []model.PageID
	// VisibleTopicIDsは閲覧者が開けるトピック。このトピックのページだけを返す
	VisibleTopicIDs []model.TopicID
	// AfterModifiedAtとAfterIDは前のページの最後のページの位置。どちらもnilなら先頭から返す
	AfterModifiedAt *time.Time
	AfterID         *model.PageID
	Limit           int32
}

// ListLinkedByModifiedAtは、PageIDsのページのうち未廃棄・未ゴミ箱で閲覧者が開けるトピックのものを、
// 更新日時の新しい順 (同じ日時ならIDの大きい順) にLimit件まで返す。未公開のページも含める。
// 絞り込みはWeb画面のリンク一覧 (FindLinkedPagesPaginated) と同じ
func (r *PageRepository) ListLinkedByModifiedAt(ctx context.Context, input ListLinkedByModifiedAtInput) ([]*model.Page, error) {
	afterModifiedAt, afterID := pageCursorParams(input.AfterModifiedAt, input.AfterID)
	rows, err := r.q.ListLinkedPagesByModifiedAt(ctx, query.ListLinkedPagesByModifiedAtParams{
		SpaceID:         string(input.SpaceID),
		PageIds:         model.PageIDsToStrings(input.PageIDs),
		VisibleTopicIds: model.TopicIDsToStrings(input.VisibleTopicIDs),
		AfterModifiedAt: afterModifiedAt,
		AfterID:         afterID,
		RowLimit:        input.Limit,
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListBacklinkedByModifiedAtInputはListBacklinkedByModifiedAtの入力
type ListBacklinkedByModifiedAtInput struct {
	SpaceID model.SpaceID
	// PageIDはリンク先のページ。保存済みの本文でこのページにリンクしているページを返す
	PageID model.PageID
	// VisibleTopicIDsは閲覧者が開けるトピック。このトピックのページだけを返す
	VisibleTopicIDs []model.TopicID
	// AfterModifiedAtとAfterIDは前のページの最後のページの位置。どちらもnilなら先頭から返す
	AfterModifiedAt *time.Time
	AfterID         *model.PageID
	Limit           int32
}

// ListBacklinkedByModifiedAtは、保存済みの本文でPageIDのページにリンクしているページのうち、
// 未廃棄・未ゴミ箱で閲覧者が開けるトピックのものを、更新日時の新しい順 (同じ日時ならIDの大きい順) に
// Limit件まで返す。絞り込みはWeb画面のバックリンク一覧 (FindBacklinkedPagesPaginated) と同じ
func (r *PageRepository) ListBacklinkedByModifiedAt(ctx context.Context, input ListBacklinkedByModifiedAtInput) ([]*model.Page, error) {
	afterModifiedAt, afterID := pageCursorParams(input.AfterModifiedAt, input.AfterID)
	rows, err := r.q.ListBacklinkedPagesByModifiedAt(ctx, query.ListBacklinkedPagesByModifiedAtParams{
		SpaceID:         string(input.SpaceID),
		PageID:          string(input.PageID),
		VisibleTopicIds: model.TopicIDsToStrings(input.VisibleTopicIDs),
		AfterModifiedAt: afterModifiedAt,
		AfterID:         afterID,
		RowLimit:        input.Limit,
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// pageCursorParamsは更新日時の新しい順の一覧のカーソルを、クエリのパラメーターに変換する。
// pages.modified_atはタイムゾーンを持たずUTCで保存しているため、日時はUTCに直す。
// どちらかがnilなら先頭から返すよう、どちらも空にする
func pageCursorParams(modifiedAt *time.Time, id *model.PageID) (sql.NullTime, *string) {
	if modifiedAt == nil || id == nil {
		return sql.NullTime{}, nil
	}
	s := string(*id)
	return sql.NullTime{Time: modifiedAt.UTC(), Valid: true}, &s
}
