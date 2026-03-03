package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// PageRepository はページリポジトリ
type PageRepository struct {
	q *query.Queries
}

// NewPageRepository は PageRepository を生成する
func NewPageRepository(q *query.Queries) *PageRepository {
	return &PageRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *PageRepository) WithTx(tx *sql.Tx) *PageRepository {
	return &PageRepository{q: r.q.WithTx(tx)}
}

// FindBySpaceAndNumber はスペースIDとページ番号でページを取得する（廃棄されていないページのみ）
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

// FindLinkedPagesPaginated はリンク先ページをオフセットページネーションで取得する（同スペース・公開済み・未廃棄のページのみ）
func (r *PageRepository) FindLinkedPagesPaginated(ctx context.Context, pageIDs []model.PageID, spaceID model.SpaceID, page int32, limit int32) (*PaginatedPages, error) {
	totalCount, err := r.q.CountLinkedPages(ctx, query.CountLinkedPagesParams{
		Column1: model.PageIDsToStrings(pageIDs),
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	rows, err := r.q.FindLinkedPagesPaginated(ctx, query.FindLinkedPagesPaginatedParams{
		Column1: model.PageIDsToStrings(pageIDs),
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

// FindBacklinkedPagesPaginated はバックリンクページをオフセットページネーションで取得する（同スペース・公開済み・未廃棄のページのみ）
func (r *PageRepository) FindBacklinkedPagesPaginated(ctx context.Context, pageID model.PageID, spaceID model.SpaceID, page int32, limit int32, excludePageIDs []model.PageID) (*PaginatedPages, error) {
	totalCount, err := r.q.CountBacklinkedPages(ctx, query.CountBacklinkedPagesParams{
		Column1: string(pageID),
		SpaceID: string(spaceID),
		Column3: model.PageIDsToStrings(excludePageIDs),
	})
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	rows, err := r.q.FindBacklinkedPagesPaginated(ctx, query.FindBacklinkedPagesPaginatedParams{
		Column1: string(pageID),
		SpaceID: string(spaceID),
		Limit:   limit,
		Offset:  offset,
		Column5: model.PageIDsToStrings(excludePageIDs),
	})
	if err != nil {
		return nil, err
	}

	return &PaginatedPages{
		Pages:      r.toModels(rows),
		TotalCount: totalCount,
	}, nil
}

// FindBacklinksForPages は複数ページのバックリンクを一括取得する（N+1回避）
func (r *PageRepository) FindBacklinksForPages(ctx context.Context, targetPages []*model.Page, spaceID model.SpaceID, limit int32, excludePageIDs []model.PageID) (map[model.PageID]*PaginatedPages, error) {
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
		Column1: targetIDs,
		SpaceID: string(spaceID),
		Limit:   limit,
		Column4: excludeIDs,
	})
	if err != nil {
		return nil, err
	}

	// バックリンク件数を一括取得
	countRows, err := r.q.CountBacklinkedPagesForTargets(ctx, query.CountBacklinkedPagesForTargetsParams{
		Column1: targetIDs,
		SpaceID: string(spaceID),
		Column3: excludeIDs,
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

// toModelFromTargetRow は FindBacklinkedPagesForTargetsRow を model.Page に変換する
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
	if row.FeaturedImageAttachmentID.Valid {
		id := model.AttachmentID(row.FeaturedImageAttachmentID.UUID.String())
		featuredImageAttachmentID = &id
	}

	return &model.Page{
		ID:                        model.PageID(row.ID),
		SpaceID:                   model.SpaceID(row.SpaceID),
		TopicID:                   model.TopicID(row.TopicID),
		Number:                    model.PageNumber(row.Number),
		Title:                     title,
		Body:                      row.Body,
		BodyHTML:                  row.BodyHtml,
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

// FindByIDs はIDリストに含まれるページを取得する（同スペース・公開済み・未廃棄のページのみ）
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

// FindBacklinkedByPageID はlinked_page_idsに指定ページIDが含まれるページを取得する（同スペース・公開済み・未廃棄のページのみ）
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

// UpdatePageInput はページ更新の入力パラメータ
type UpdatePageInput struct {
	ID                        model.PageID
	SpaceID                   model.SpaceID
	TopicID                   model.TopicID
	Title                     *string
	Body                      string
	BodyHTML                  string
	LinkedPageIDs             []model.PageID
	ModifiedAt                time.Time
	PublishedAt               *time.Time
	FeaturedImageAttachmentID *model.AttachmentID
}

// Update はページを更新する
func (r *PageRepository) Update(ctx context.Context, input UpdatePageInput) (*model.Page, error) {
	var publishedAt sql.NullTime
	if input.PublishedAt != nil {
		publishedAt = sql.NullTime{Time: *input.PublishedAt, Valid: true}
	}

	var featuredImageAttachmentID uuid.NullUUID
	if input.FeaturedImageAttachmentID != nil {
		parsed, err := uuid.Parse(string(*input.FeaturedImageAttachmentID))
		if err != nil {
			return nil, err
		}
		featuredImageAttachmentID = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	row, err := r.q.UpdatePage(ctx, query.UpdatePageParams{
		ID:                        string(input.ID),
		TopicID:                   string(input.TopicID),
		Title:                     input.Title,
		Body:                      input.Body,
		BodyHtml:                  input.BodyHTML,
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

// FindByTopicAndTitle は指定トピック内で指定タイトルのページを取得する（廃棄されていないページのみ、スペースIDでスコープ）
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

// NextPageNumber はスペース内の次のページ番号を取得する
func (r *PageRepository) NextPageNumber(ctx context.Context, spaceID model.SpaceID) (model.PageNumber, error) {
	n, err := r.q.GetNextPageNumber(ctx, string(spaceID))
	if err != nil {
		return 0, err
	}
	return model.PageNumber(n), nil
}

// CreateLinkedPageInput はWikiリンクから参照されるページ作成の入力パラメータ
type CreateLinkedPageInput struct {
	SpaceID model.SpaceID
	TopicID model.TopicID
	Number  model.PageNumber
	Title   string
}

// CreateLinkedPage はWikiリンクから参照されるページを作成する
func (r *PageRepository) CreateLinkedPage(ctx context.Context, input CreateLinkedPageInput) (*model.Page, error) {
	now := time.Now()
	row, err := r.q.CreateLinkedPage(ctx, query.CreateLinkedPageParams{
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

// PageLocation はページロケーション検索の結果
type PageLocation struct {
	TopicName string
	PageTitle string
}

// escapeLikePattern はPostgreSQLのLIKE特殊文字（\, %, _）をエスケープする
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// SearchPageLocations はスペース内のページをタイトルで検索する（Wikiリンク補完用）
func (r *PageRepository) SearchPageLocations(ctx context.Context, spaceID model.SpaceID, q string) ([]PageLocation, error) {
	// 検索キーワードをスペースで分割し、各ワードをILIKEパターンに変換
	words := strings.Fields(q)
	patterns := make([]string, len(words))
	for i, word := range words {
		patterns[i] = fmt.Sprintf("%%%s%%", escapeLikePattern(word))
	}

	rows, err := r.q.SearchPageLocations(ctx, query.SearchPageLocationsParams{
		SpaceID: string(spaceID),
		Column2: patterns,
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

// toModel は query.Page を model.Page に変換する
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
	if row.FeaturedImageAttachmentID.Valid {
		id := model.AttachmentID(row.FeaturedImageAttachmentID.UUID.String())
		featuredImageAttachmentID = &id
	}

	return &model.Page{
		ID:                        model.PageID(row.ID),
		SpaceID:                   model.SpaceID(row.SpaceID),
		TopicID:                   model.TopicID(row.TopicID),
		Number:                    model.PageNumber(row.Number),
		Title:                     title,
		Body:                      row.Body,
		BodyHTML:                  row.BodyHtml,
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

// toModels は query.Page のスライスを model.Page のスライスに変換する
func (r *PageRepository) toModels(rows []query.Page) []*model.Page {
	pages := make([]*model.Page, len(rows))
	for i, row := range rows {
		pages[i] = r.toModel(row)
	}
	return pages
}
