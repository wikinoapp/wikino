package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// SuggestionRepository は編集提案リポジトリ
type SuggestionRepository struct {
	q *query.Queries
}

// NewSuggestionRepository は SuggestionRepository を生成する
func NewSuggestionRepository(q *query.Queries) *SuggestionRepository {
	return &SuggestionRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *SuggestionRepository) WithTx(tx *sql.Tx) *SuggestionRepository {
	return &SuggestionRepository{q: r.q.WithTx(tx)}
}

// CreateSuggestionInput は編集提案作成の入力パラメータ
type CreateSuggestionInput struct {
	SpaceID              model.SpaceID
	TopicID              model.TopicID
	CreatedSpaceMemberID model.SpaceMemberID
	Number               model.SuggestionNumber
	Title                string
	Body                 string
	BodyHTML             string
	Status               model.SuggestionStatus
}

// Create は編集提案を作成する
func (r *SuggestionRepository) Create(ctx context.Context, input CreateSuggestionInput) (*model.Suggestion, error) {
	now := time.Now()
	row, err := r.q.CreateSuggestion(ctx, query.CreateSuggestionParams{
		SpaceID:              string(input.SpaceID),
		TopicID:              string(input.TopicID),
		CreatedSpaceMemberID: string(input.CreatedSpaceMemberID),
		Number:               int32(input.Number),
		Title:                input.Title,
		Body:                 input.Body,
		BodyHtml:             input.BodyHTML,
		Status:               int32(input.Status),
		CreatedAt:            now,
		UpdatedAt:            now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByID はIDで編集提案を取得する（スペースIDでスコープ）
func (r *SuggestionRepository) FindByID(ctx context.Context, id model.SuggestionID, spaceID model.SpaceID) (*model.Suggestion, error) {
	row, err := r.q.FindSuggestionByID(ctx, query.FindSuggestionByIDParams{
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

// FindByTopicAndNumber はトピックIDと番号で編集提案を取得する（スペースIDでスコープ）
func (r *SuggestionRepository) FindByTopicAndNumber(ctx context.Context, topicID model.TopicID, number model.SuggestionNumber, spaceID model.SpaceID) (*model.Suggestion, error) {
	row, err := r.q.FindSuggestionByTopicAndNumber(ctx, query.FindSuggestionByTopicAndNumberParams{
		TopicID: string(topicID),
		Number:  int32(number),
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

// FindBySpaceAndNumber はスペースIDと番号で編集提案を取得する
func (r *SuggestionRepository) FindBySpaceAndNumber(ctx context.Context, spaceID model.SpaceID, number model.SuggestionNumber) (*model.Suggestion, error) {
	row, err := r.q.FindSuggestionBySpaceAndNumber(ctx, query.FindSuggestionBySpaceAndNumberParams{
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

// ListByTopicAndStatuses はトピックIDとステータスリストで編集提案一覧を取得する
func (r *SuggestionRepository) ListByTopicAndStatuses(ctx context.Context, topicID model.TopicID, spaceID model.SpaceID, statuses []model.SuggestionStatus) ([]*model.Suggestion, error) {
	statusInts := make([]int32, len(statuses))
	for i, s := range statuses {
		statusInts[i] = int32(s)
	}
	rows, err := r.q.ListSuggestionsByTopicAndStatuses(ctx, query.ListSuggestionsByTopicAndStatusesParams{
		TopicID: string(topicID),
		SpaceID: string(spaceID),
		Column3: statusInts,
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// UpdateStatusInput は編集提案ステータス更新の入力パラメータ
type UpdateStatusInput struct {
	ID        model.SuggestionID
	SpaceID   model.SpaceID
	Status    model.SuggestionStatus
	AppliedAt *time.Time
}

// UpdateStatus は編集提案のステータスを更新する
func (r *SuggestionRepository) UpdateStatus(ctx context.Context, input UpdateStatusInput) (*model.Suggestion, error) {
	var appliedAt sql.NullTime
	if input.AppliedAt != nil {
		appliedAt = sql.NullTime{Time: *input.AppliedAt, Valid: true}
	}

	row, err := r.q.UpdateSuggestionStatus(ctx, query.UpdateSuggestionStatusParams{
		ID:        string(input.ID),
		Status:    int32(input.Status),
		AppliedAt: appliedAt,
		UpdatedAt: time.Now(),
		SpaceID:   string(input.SpaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateSuggestionInput は編集提案更新の入力パラメータ
type UpdateSuggestionInput struct {
	ID       model.SuggestionID
	SpaceID  model.SpaceID
	Title    string
	Body     string
	BodyHTML string
}

// Update は編集提案のタイトルと本文を更新する
func (r *SuggestionRepository) Update(ctx context.Context, input UpdateSuggestionInput) (*model.Suggestion, error) {
	row, err := r.q.UpdateSuggestion(ctx, query.UpdateSuggestionParams{
		ID:        string(input.ID),
		Title:     input.Title,
		Body:      input.Body,
		BodyHtml:  input.BodyHTML,
		UpdatedAt: time.Now(),
		SpaceID:   string(input.SpaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// CountByTopicAndStatuses はトピックIDとステータスリストで編集提案の件数を取得する
func (r *SuggestionRepository) CountByTopicAndStatuses(ctx context.Context, topicID model.TopicID, spaceID model.SpaceID, statuses []model.SuggestionStatus) (int64, error) {
	statusInts := make([]int32, len(statuses))
	for i, s := range statuses {
		statusInts[i] = int32(s)
	}
	return r.q.CountSuggestionsByTopicAndStatuses(ctx, query.CountSuggestionsByTopicAndStatusesParams{
		TopicID: string(topicID),
		SpaceID: string(spaceID),
		Column3: statusInts,
	})
}

// GetNextNumber はスペース内の次の編集提案番号を取得する
func (r *SuggestionRepository) GetNextNumber(ctx context.Context, spaceID model.SpaceID) (model.SuggestionNumber, error) {
	n, err := r.q.GetNextSuggestionNumber(ctx, string(spaceID))
	if err != nil {
		return 0, err
	}
	return model.SuggestionNumber(n), nil
}

// toModel は query.Suggestion を model.Suggestion に変換する
func (r *SuggestionRepository) toModel(row query.Suggestion) *model.Suggestion {
	var appliedAt *time.Time
	if row.AppliedAt.Valid {
		appliedAt = &row.AppliedAt.Time
	}

	return &model.Suggestion{
		ID:                   model.SuggestionID(row.ID),
		SpaceID:              model.SpaceID(row.SpaceID),
		TopicID:              model.TopicID(row.TopicID),
		CreatedSpaceMemberID: model.SpaceMemberID(row.CreatedSpaceMemberID),
		Number:               model.SuggestionNumber(row.Number),
		Title:                row.Title,
		Body:                 row.Body,
		BodyHTML:             row.BodyHtml,
		Status:               model.SuggestionStatus(row.Status),
		AppliedAt:            appliedAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

// toModels は query.Suggestion のスライスを model.Suggestion のスライスに変換する
func (r *SuggestionRepository) toModels(rows []query.Suggestion) []*model.Suggestion {
	suggestions := make([]*model.Suggestion, len(rows))
	for i, row := range rows {
		suggestions[i] = r.toModel(row)
	}
	return suggestions
}
