package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// DraftPageRepository は下書きページリポジトリ
type DraftPageRepository struct {
	q *query.Queries
}

// NewDraftPageRepository は DraftPageRepository を生成する
func NewDraftPageRepository(q *query.Queries) *DraftPageRepository {
	return &DraftPageRepository{q: q}
}

// WithTx はトランザクションを使用する新しいRepositoryを返す
func (r *DraftPageRepository) WithTx(tx *sql.Tx) *DraftPageRepository {
	return &DraftPageRepository{q: r.q.WithTx(tx)}
}

// FindByID はIDで下書きを取得する（スペースIDでスコープ）
func (r *DraftPageRepository) FindByID(ctx context.Context, id model.DraftPageID, spaceID model.SpaceID) (*model.DraftPage, error) {
	row, err := r.q.FindDraftPageByID(ctx, query.FindDraftPageByIDParams{
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

// FindByPageAndMember はページIDとスペースメンバーIDで下書きを取得する
func (r *DraftPageRepository) FindByPageAndMember(ctx context.Context, pageID model.PageID, spaceMemberID model.SpaceMemberID, spaceID model.SpaceID) (*model.DraftPage, error) {
	row, err := r.q.FindDraftPageByPageAndMember(ctx, query.FindDraftPageByPageAndMemberParams{
		PageID:        string(pageID),
		SpaceMemberID: string(spaceMemberID),
		SpaceID:       string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// CreateDraftPageInput は下書き作成の入力パラメータ
type CreateDraftPageInput struct {
	SpaceID                   model.SpaceID
	PageID                    model.PageID
	SpaceMemberID             model.SpaceMemberID
	TopicID                   model.TopicID
	SuggestionPageID          *model.SuggestionPageID
	Title                     *string
	Body                      string
	BodyHTML                  string
	LinkedPageIDs             []model.PageID
	FeaturedImageAttachmentID *model.AttachmentID
	ModifiedAt                time.Time
}

// Create は下書きを作成する
func (r *DraftPageRepository) Create(ctx context.Context, input CreateDraftPageInput) (*model.DraftPage, error) {
	now := time.Now()

	var suggestionPageID *string
	if input.SuggestionPageID != nil {
		s := string(*input.SuggestionPageID)
		suggestionPageID = &s
	}

	var featuredImageAttachmentID *string
	if input.FeaturedImageAttachmentID != nil {
		s := string(*input.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &s
	}

	row, err := r.q.CreateDraftPage(ctx, query.CreateDraftPageParams{
		SpaceID:                   string(input.SpaceID),
		PageID:                    string(input.PageID),
		SpaceMemberID:             string(input.SpaceMemberID),
		TopicID:                   string(input.TopicID),
		SuggestionPageID:          suggestionPageID,
		Title:                     input.Title,
		Body:                      input.Body,
		BodyHtml:                  input.BodyHTML,
		LinkedPageIds:             model.PageIDsToStrings(input.LinkedPageIDs),
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		ModifiedAt:                input.ModifiedAt,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateDraftPageInput は下書き更新の入力パラメータ
type UpdateDraftPageInput struct {
	ID                        model.DraftPageID
	SpaceID                   model.SpaceID
	TopicID                   model.TopicID
	Title                     *string
	Body                      string
	BodyHTML                  string
	LinkedPageIDs             []model.PageID
	FeaturedImageAttachmentID *model.AttachmentID
	ModifiedAt                time.Time
}

// Update は下書きを更新する
func (r *DraftPageRepository) Update(ctx context.Context, input UpdateDraftPageInput) (*model.DraftPage, error) {
	var featuredImageAttachmentID *string
	if input.FeaturedImageAttachmentID != nil {
		s := string(*input.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &s
	}

	row, err := r.q.UpdateDraftPage(ctx, query.UpdateDraftPageParams{
		ID:                        string(input.ID),
		TopicID:                   string(input.TopicID),
		Title:                     input.Title,
		Body:                      input.Body,
		BodyHtml:                  input.BodyHTML,
		LinkedPageIds:             model.PageIDsToStrings(input.LinkedPageIDs),
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		ModifiedAt:                input.ModifiedAt,
		UpdatedAt:                 time.Now(),
		SpaceID:                   string(input.SpaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// Delete は下書きを削除する
func (r *DraftPageRepository) Delete(ctx context.Context, id model.DraftPageID, spaceID model.SpaceID) error {
	return r.q.DeleteDraftPage(ctx, query.DeleteDraftPageParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
}

// FindBySuggestionPageID は編集提案ページIDで下書きを取得する
func (r *DraftPageRepository) FindBySuggestionPageID(ctx context.Context, suggestionPageID model.SuggestionPageID, spaceID model.SpaceID) (*model.DraftPage, error) {
	spID := string(suggestionPageID)

	row, err := r.q.FindDraftPageBySuggestionPageID(ctx, query.FindDraftPageBySuggestionPageIDParams{
		SuggestionPageID: &spID,
		SpaceID:          string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateSuggestionPageID は下書きの編集提案ページIDを更新する
func (r *DraftPageRepository) UpdateSuggestionPageID(ctx context.Context, id model.DraftPageID, spaceID model.SpaceID, suggestionPageID *model.SuggestionPageID) (*model.DraftPage, error) {
	var spID *string
	if suggestionPageID != nil {
		s := string(*suggestionPageID)
		spID = &s
	}

	row, err := r.q.UpdateDraftPageSuggestionPageID(ctx, query.UpdateDraftPageSuggestionPageIDParams{
		ID:               string(id),
		SuggestionPageID: spID,
		UpdatedAt:        time.Now(),
		SpaceID:          string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// ListByUser はユーザーの下書きページ一覧を取得する（サイドバー表示用）
func (r *DraftPageRepository) ListByUser(ctx context.Context, userID model.UserID, limit int32) ([]*model.DraftPage, error) {
	rows, err := r.q.ListDraftPagesByUser(ctx, query.ListDraftPagesByUserParams{
		UserID: string(userID),
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}
	return r.toDraftPagesFromJoinedRows(rows), nil
}

// ListByUserForIndex はユーザーの下書きページ一覧を取得する（下書き一覧画面用）
func (r *DraftPageRepository) ListByUserForIndex(ctx context.Context, userID model.UserID) ([]*model.DraftPage, error) {
	rows, err := r.q.ListDraftPagesByUserForIndex(ctx, string(userID))
	if err != nil {
		return nil, err
	}
	return r.toDraftPagesFromIndexRows(rows), nil
}

// toDraftPagesFromIndexRows は query.ListDraftPagesByUserForIndexRow のスライスを model.DraftPage のスライスに変換する
func (r *DraftPageRepository) toDraftPagesFromIndexRows(rows []query.ListDraftPagesByUserForIndexRow) []*model.DraftPage {
	drafts := make([]*model.DraftPage, len(rows))
	for i, row := range rows {
		var draftTitle *string
		if row.DraftPageTitle != nil {
			switch v := row.DraftPageTitle.(type) {
			case string:
				draftTitle = &v
			case []byte:
				s := string(v)
				draftTitle = &s
			}
		}

		var pageTitle *string
		if row.PageTitle != nil {
			switch v := row.PageTitle.(type) {
			case string:
				pageTitle = &v
			case []byte:
				s := string(v)
				pageTitle = &s
			}
		}

		drafts[i] = &model.DraftPage{
			ID:         model.DraftPageID(row.DraftPageID),
			Title:      draftTitle,
			ModifiedAt: row.DraftPageModifiedAt,
			Page: &model.Page{
				ID:     model.PageID(row.PageID),
				Title:  pageTitle,
				Number: model.PageNumber(row.PageNumber),
			},
			Topic: &model.Topic{
				ID:         model.TopicID(row.TopicID),
				Name:       row.TopicName,
				Number:     row.TopicNumber,
				Visibility: model.TopicVisibility(row.TopicVisibility),
				Space: &model.Space{
					ID:         model.SpaceID(row.SpaceID),
					Identifier: model.SpaceIdentifier(row.SpaceIdentifier),
					Name:       row.SpaceName,
				},
			},
		}
	}
	return drafts
}

// toDraftPagesFromJoinedRows は query.ListDraftPagesByUserRow のスライスを model.DraftPage のスライスに変換する
func (r *DraftPageRepository) toDraftPagesFromJoinedRows(rows []query.ListDraftPagesByUserRow) []*model.DraftPage {
	drafts := make([]*model.DraftPage, len(rows))
	for i, row := range rows {
		var draftTitle *string
		if row.DraftPageTitle != nil {
			switch v := row.DraftPageTitle.(type) {
			case string:
				draftTitle = &v
			case []byte:
				s := string(v)
				draftTitle = &s
			}
		}

		var pageTitle *string
		if row.PageTitle != nil {
			switch v := row.PageTitle.(type) {
			case string:
				pageTitle = &v
			case []byte:
				s := string(v)
				pageTitle = &s
			}
		}

		drafts[i] = &model.DraftPage{
			ID:         model.DraftPageID(row.DraftPageID),
			Title:      draftTitle,
			ModifiedAt: row.DraftPageModifiedAt,
			Page: &model.Page{
				ID:     model.PageID(row.PageID),
				Title:  pageTitle,
				Number: model.PageNumber(row.PageNumber),
			},
			Topic: &model.Topic{
				Name:       row.TopicName,
				Visibility: model.TopicVisibility(row.TopicVisibility),
				Space: &model.Space{
					Identifier: model.SpaceIdentifier(row.SpaceIdentifier),
				},
			},
		}
	}
	return drafts
}

// ListByMemberAndTopic はスペースメンバーとトピックに紐づく下書きページ一覧を取得する（編集提案作成画面用）
func (r *DraftPageRepository) ListByMemberAndTopic(ctx context.Context, spaceMemberID model.SpaceMemberID, topicID model.TopicID, spaceID model.SpaceID) ([]*model.DraftPage, error) {
	rows, err := r.q.ListDraftPagesByMemberAndTopic(ctx, query.ListDraftPagesByMemberAndTopicParams{
		SpaceMemberID: string(spaceMemberID),
		TopicID:       string(topicID),
		SpaceID:       string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	return r.toDraftPagesFromMemberTopicRows(rows), nil
}

// toDraftPagesFromMemberTopicRows は query.ListDraftPagesByMemberAndTopicRow のスライスを model.DraftPage のスライスに変換する
func (r *DraftPageRepository) toDraftPagesFromMemberTopicRows(rows []query.ListDraftPagesByMemberAndTopicRow) []*model.DraftPage {
	drafts := make([]*model.DraftPage, len(rows))
	for i, row := range rows {
		var draftTitle *string
		if row.Title != nil {
			switch v := row.Title.(type) {
			case string:
				draftTitle = &v
			case []byte:
				s := string(v)
				draftTitle = &s
			}
		}

		var pageTitle *string
		if row.PageTitle != nil {
			switch v := row.PageTitle.(type) {
			case string:
				pageTitle = &v
			case []byte:
				s := string(v)
				pageTitle = &s
			}
		}

		drafts[i] = &model.DraftPage{
			ID:            model.DraftPageID(row.ID),
			SpaceID:       model.SpaceID(row.SpaceID),
			PageID:        model.PageID(row.PageID),
			SpaceMemberID: model.SpaceMemberID(row.SpaceMemberID),
			TopicID:       model.TopicID(row.TopicID),
			Title:         draftTitle,
			Body:          row.Body,
			BodyHTML:      row.BodyHtml,
			LinkedPageIDs: model.StringsToPageIDs(row.LinkedPageIds),
			ModifiedAt:    row.ModifiedAt,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			Page: &model.Page{
				ID:     model.PageID(row.PageID),
				Title:  pageTitle,
				Number: model.PageNumber(row.PageNumber),
			},
		}
	}
	return drafts
}

// toModel は query.DraftPage を model.DraftPage に変換する
func (r *DraftPageRepository) toModel(row query.DraftPage) *model.DraftPage {
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

	var suggestionPageID *model.SuggestionPageID
	if row.SuggestionPageID != nil {
		id := model.SuggestionPageID(*row.SuggestionPageID)
		suggestionPageID = &id
	}

	var featuredImageAttachmentID *model.AttachmentID
	if row.FeaturedImageAttachmentID != nil {
		id := model.AttachmentID(*row.FeaturedImageAttachmentID)
		featuredImageAttachmentID = &id
	}

	return &model.DraftPage{
		ID:                        model.DraftPageID(row.ID),
		SpaceID:                   model.SpaceID(row.SpaceID),
		PageID:                    model.PageID(row.PageID),
		SpaceMemberID:             model.SpaceMemberID(row.SpaceMemberID),
		TopicID:                   model.TopicID(row.TopicID),
		SuggestionPageID:          suggestionPageID,
		Title:                     title,
		Body:                      row.Body,
		BodyHTML:                  row.BodyHtml,
		LinkedPageIDs:             model.StringsToPageIDs(row.LinkedPageIds),
		FeaturedImageAttachmentID: featuredImageAttachmentID,
		ModifiedAt:                row.ModifiedAt,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}
}
