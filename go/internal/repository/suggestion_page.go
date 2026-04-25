package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// SuggestionPageRepository は編集提案ページリポジトリ
type SuggestionPageRepository struct {
	q *query.Queries
}

// NewSuggestionPageRepository は SuggestionPageRepository を生成する
func NewSuggestionPageRepository(q *query.Queries) *SuggestionPageRepository {
	return &SuggestionPageRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *SuggestionPageRepository) WithTx(tx *sql.Tx) *SuggestionPageRepository {
	return &SuggestionPageRepository{q: r.q.WithTx(tx)}
}

// CreateSuggestionPageInput は編集提案ページ作成の入力パラメータ
type CreateSuggestionPageInput struct {
	SpaceID                   model.SpaceID
	SuggestionID              model.SuggestionID
	PageID                    model.PageID
	PageRevisionID            *model.PageRevisionID
	Title                     *string
	Body                      string
	BodyHTML                  string
	LinkedPageIDs             []model.PageID
	FeaturedImageAttachmentID *model.AttachmentID
}

// Create は編集提案ページを作成する
func (r *SuggestionPageRepository) Create(ctx context.Context, input CreateSuggestionPageInput) (*model.SuggestionPage, error) {
	now := time.Now()

	var title sql.NullString
	if input.Title != nil {
		title = sql.NullString{String: *input.Title, Valid: true}
	}

	var pageRevisionID *string
	if input.PageRevisionID != nil {
		s := string(*input.PageRevisionID)
		pageRevisionID = &s
	}

	var featuredImageAttachmentID *string
	if input.FeaturedImageAttachmentID != nil {
		s := string(*input.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &s
	}

	row, err := r.q.CreateSuggestionPage(ctx, query.CreateSuggestionPageParams{
		SpaceID:                   string(input.SpaceID),
		SuggestionID:              string(input.SuggestionID),
		PageID:                    string(input.PageID),
		PageRevisionID:            pageRevisionID,
		Title:                     title,
		Body:                      input.Body,
		BodyHtml:                  input.BodyHTML,
		LinkedPageIds:             model.PageIDsToStrings(input.LinkedPageIDs),
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByID はIDで編集提案ページを取得する（スペースIDでスコープ）
func (r *SuggestionPageRepository) FindByID(ctx context.Context, id model.SuggestionPageID, spaceID model.SpaceID) (*model.SuggestionPage, error) {
	row, err := r.q.FindSuggestionPageByID(ctx, query.FindSuggestionPageByIDParams{
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

// ListBySuggestionID は編集提案IDで編集提案ページ一覧を取得する
func (r *SuggestionPageRepository) ListBySuggestionID(ctx context.Context, suggestionID model.SuggestionID, spaceID model.SpaceID) ([]*model.SuggestionPage, error) {
	rows, err := r.q.ListSuggestionPagesBySuggestionID(ctx, query.ListSuggestionPagesBySuggestionIDParams{
		SuggestionID: string(suggestionID),
		SpaceID:      string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// UpdateSuggestionPageContentInput は編集提案ページコンテンツ更新の入力パラメータ
type UpdateSuggestionPageContentInput struct {
	ID                        model.SuggestionPageID
	SpaceID                   model.SpaceID
	Title                     *string
	Body                      string
	BodyHTML                  string
	LinkedPageIDs             []model.PageID
	FeaturedImageAttachmentID *model.AttachmentID
}

// UpdateContent は編集提案ページのコンテンツを更新する
func (r *SuggestionPageRepository) UpdateContent(ctx context.Context, input UpdateSuggestionPageContentInput) (*model.SuggestionPage, error) {
	var title sql.NullString
	if input.Title != nil {
		title = sql.NullString{String: *input.Title, Valid: true}
	}

	var featuredImageAttachmentID *string
	if input.FeaturedImageAttachmentID != nil {
		s := string(*input.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &s
	}

	row, err := r.q.UpdateSuggestionPageContent(ctx, query.UpdateSuggestionPageContentParams{
		ID:                        string(input.ID),
		Title:                     title,
		Body:                      input.Body,
		BodyHtml:                  input.BodyHTML,
		LinkedPageIds:             model.PageIDsToStrings(input.LinkedPageIDs),
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		UpdatedAt:                 time.Now(),
		SpaceID:                   string(input.SpaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// Delete は編集提案ページを削除する
func (r *SuggestionPageRepository) Delete(ctx context.Context, id model.SuggestionPageID, spaceID model.SpaceID) error {
	return r.q.DeleteSuggestionPage(ctx, query.DeleteSuggestionPageParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
}

// ExistsByPageIDAndOpenStatus は指定ページを参照しているオープンな編集提案が存在するかを確認する
func (r *SuggestionPageRepository) ExistsByPageIDAndOpenStatus(ctx context.Context, pageID model.PageID, spaceID model.SpaceID) (bool, error) {
	return r.q.ExistsOpenSuggestionByPageID(ctx, query.ExistsOpenSuggestionByPageIDParams{
		PageID:  string(pageID),
		SpaceID: string(spaceID),
	})
}

// toModel は query.SuggestionPage を model.SuggestionPage に変換する
func (r *SuggestionPageRepository) toModel(row query.SuggestionPage) *model.SuggestionPage {
	var title *string
	if row.Title.Valid {
		title = &row.Title.String
	}

	var featuredImageAttachmentID *model.AttachmentID
	if row.FeaturedImageAttachmentID != nil {
		id := model.AttachmentID(*row.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &id
	}

	var pageRevisionID *model.PageRevisionID
	if row.PageRevisionID != nil {
		id := model.PageRevisionID(*row.PageRevisionID)
		pageRevisionID = &id
	}

	return &model.SuggestionPage{
		ID:                        model.SuggestionPageID(row.ID),
		SpaceID:                   model.SpaceID(row.SpaceID),
		SuggestionID:              model.SuggestionID(row.SuggestionID),
		PageID:                    model.PageID(row.PageID),
		PageRevisionID:            pageRevisionID,
		Title:                     title,
		Body:                      row.Body,
		BodyHTML:                  row.BodyHtml,
		LinkedPageIDs:             model.StringsToPageIDs(row.LinkedPageIds),
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}
}

// toModels は query.SuggestionPage のスライスを model.SuggestionPage のスライスに変換する
func (r *SuggestionPageRepository) toModels(rows []query.SuggestionPage) []*model.SuggestionPage {
	pages := make([]*model.SuggestionPage, len(rows))
	for i, row := range rows {
		pages[i] = r.toModel(row)
	}
	return pages
}
