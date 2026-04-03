package validator

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

var (
	ErrDraftPageNotFound  = errors.New("下書きページが見つかりません")
	ErrDraftPageNotLinked = errors.New("下書きページが編集提案ページにリンクされていません")
)

// SuggestionPageCreateValidator は編集提案ページ追加のバリデーションを行う
type SuggestionPageCreateValidator struct {
	draftPageRepo      *repository.DraftPageRepository
	suggestionPageRepo *repository.SuggestionPageRepository
}

// NewSuggestionPageCreateValidator は SuggestionPageCreateValidator を生成する
func NewSuggestionPageCreateValidator(
	draftPageRepo *repository.DraftPageRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
) *SuggestionPageCreateValidator {
	return &SuggestionPageCreateValidator{
		draftPageRepo:      draftPageRepo,
		suggestionPageRepo: suggestionPageRepo,
	}
}

// SuggestionPageCreateValidatorInput はバリデーションの入力パラメータ
type SuggestionPageCreateValidatorInput struct {
	DraftPageIDs  []model.DraftPageID
	SpaceMemberID model.SpaceMemberID
	TopicID       model.TopicID
	SpaceID       model.SpaceID
	SuggestionID  model.SuggestionID
}

// Validate はバリデーションを行う。
// 成功時は検証済みの DraftPage スライスを返す。
func (v *SuggestionPageCreateValidator) Validate(ctx context.Context, input SuggestionPageCreateValidatorInput) ([]*model.DraftPage, error) {
	ve := model.NewValidationError()

	// 下書きページ選択チェック
	if len(input.DraftPageIDs) == 0 {
		ve.AddField("draft_page_ids", i18n.T(ctx, "validation_suggestion_draft_pages_required"))
		return nil, ve
	}

	// 既存の編集提案ページを取得（重複チェック用）
	existingPages, err := v.suggestionPageRepo.ListBySuggestionID(ctx, input.SuggestionID, input.SpaceID)
	if err != nil {
		return nil, err
	}
	existingPageIDs := make(map[model.PageID]bool, len(existingPages))
	for _, sp := range existingPages {
		existingPageIDs[sp.PageID] = true
	}

	// 下書きページの検証
	draftPages := make([]*model.DraftPage, 0, len(input.DraftPageIDs))
	for _, draftPageID := range input.DraftPageIDs {
		draftPage, err := v.draftPageRepo.FindByID(ctx, draftPageID, input.SpaceID)
		if err != nil {
			return nil, err
		}

		// 存在チェック・所有者チェック・トピックチェック
		if draftPage == nil || draftPage.SpaceMemberID != input.SpaceMemberID || draftPage.TopicID != input.TopicID {
			ve.AddField("draft_page_ids", i18n.T(ctx, "validation_suggestion_page_draft_page_not_found"))
			return nil, ve
		}

		// 重複ページチェック
		if existingPageIDs[draftPage.PageID] {
			ve.AddField("draft_page_ids", i18n.T(ctx, "validation_suggestion_page_already_exists"))
			return nil, ve
		}

		draftPages = append(draftPages, draftPage)
	}

	return draftPages, nil
}

// SuggestionPageUpdateValidator は編集提案ページ更新のバリデーションを行う
type SuggestionPageUpdateValidator struct {
	draftPageRepo *repository.DraftPageRepository
}

// NewSuggestionPageUpdateValidator は SuggestionPageUpdateValidator を生成する
func NewSuggestionPageUpdateValidator(draftPageRepo *repository.DraftPageRepository) *SuggestionPageUpdateValidator {
	return &SuggestionPageUpdateValidator{
		draftPageRepo: draftPageRepo,
	}
}

// SuggestionPageUpdateValidatorInput はバリデーションの入力パラメータ
type SuggestionPageUpdateValidatorInput struct {
	SuggestionPageID model.SuggestionPageID
	PageID           model.PageID
	SpaceMemberID    model.SpaceMemberID
	SpaceID          model.SpaceID
}

// Validate はバリデーションを行う。
// 成功時はDraftPageを返す。エラー時はErrDraftPageNotFound、ErrDraftPageNotLinked、またはシステムエラーを返す。
func (v *SuggestionPageUpdateValidator) Validate(ctx context.Context, input SuggestionPageUpdateValidatorInput) (*model.DraftPage, error) {
	// DraftPageの存在確認
	dp, err := v.draftPageRepo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return nil, err
	}
	if dp == nil {
		return nil, ErrDraftPageNotFound
	}

	// DraftPageが対象のSuggestionPageにリンクされていることを検証
	if dp.SuggestionPageID == nil || *dp.SuggestionPageID != input.SuggestionPageID {
		return nil, ErrDraftPageNotLinked
	}

	return dp, nil
}
