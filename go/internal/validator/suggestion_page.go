package validator

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

var (
	ErrDraftPageNotFound  = errors.New("下書きページが見つかりません")
	ErrDraftPageNotLinked = errors.New("下書きページが編集提案ページにリンクされていません")
)

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

// SuggestionPageUpdateValidatorResult はバリデーションの結果
type SuggestionPageUpdateValidatorResult struct {
	DraftPage *model.DraftPage
	Err       error
}

// Validate はバリデーションを行う
func (v *SuggestionPageUpdateValidator) Validate(ctx context.Context, input SuggestionPageUpdateValidatorInput) *SuggestionPageUpdateValidatorResult {
	// DraftPageの存在確認
	dp, err := v.draftPageRepo.FindByPageAndMember(ctx, input.PageID, input.SpaceMemberID, input.SpaceID)
	if err != nil {
		return &SuggestionPageUpdateValidatorResult{Err: err}
	}
	if dp == nil {
		return &SuggestionPageUpdateValidatorResult{Err: ErrDraftPageNotFound}
	}

	// DraftPageが対象のSuggestionPageにリンクされていることを検証
	if dp.SuggestionPageID == nil || *dp.SuggestionPageID != input.SuggestionPageID {
		return &SuggestionPageUpdateValidatorResult{Err: ErrDraftPageNotLinked}
	}

	return &SuggestionPageUpdateValidatorResult{DraftPage: dp}
}
