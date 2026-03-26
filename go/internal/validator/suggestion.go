package validator

import (
	"context"
	"unicode/utf8"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

const (
	suggestionTitleMaxLength = 200
	suggestionBodyMaxLength  = 10000
)

// SuggestionCreateValidator は編集提案作成のバリデーションを行う
type SuggestionCreateValidator struct {
	draftPageRepo *repository.DraftPageRepository
}

// NewSuggestionCreateValidator は SuggestionCreateValidator を生成する
func NewSuggestionCreateValidator(draftPageRepo *repository.DraftPageRepository) *SuggestionCreateValidator {
	return &SuggestionCreateValidator{
		draftPageRepo: draftPageRepo,
	}
}

// SuggestionCreateValidatorInput はバリデーションの入力パラメータ
type SuggestionCreateValidatorInput struct {
	Title         string
	Body          string
	DraftPageIDs  []model.DraftPageID
	SpaceMemberID model.SpaceMemberID
	TopicID       model.TopicID
	SpaceID       model.SpaceID
}

// SuggestionCreateValidatorResult はバリデーションの結果
type SuggestionCreateValidatorResult struct {
	FormErrors *session.FormErrors
	DraftPages []*model.DraftPage
	Err        error
}

// Validate はバリデーションを行う
func (v *SuggestionCreateValidator) Validate(ctx context.Context, input SuggestionCreateValidatorInput) *SuggestionCreateValidatorResult {
	formErrors := session.NewFormErrors()

	// タイトル必須チェック
	if input.Title == "" {
		formErrors.AddField("title", i18n.T(ctx, "validation_suggestion_title_required"))
	}

	// タイトル文字数チェック
	if input.Title != "" && utf8.RuneCountInString(input.Title) > suggestionTitleMaxLength {
		formErrors.AddField("title", i18n.T(ctx, "validation_suggestion_title_too_long"))
	}

	// 本文文字数チェック
	if input.Body != "" && utf8.RuneCountInString(input.Body) > suggestionBodyMaxLength {
		formErrors.AddField("body", i18n.T(ctx, "validation_suggestion_body_too_long"))
	}

	// 下書きページ選択チェック
	if len(input.DraftPageIDs) == 0 {
		formErrors.AddField("draft_page_ids", i18n.T(ctx, "validation_suggestion_draft_pages_required"))
	}

	if formErrors.HasErrors() {
		return &SuggestionCreateValidatorResult{FormErrors: formErrors}
	}

	// 下書きページの存在確認（状態バリデーション）
	draftPages := make([]*model.DraftPage, 0, len(input.DraftPageIDs))
	for _, draftPageID := range input.DraftPageIDs {
		draftPage, err := v.draftPageRepo.FindByID(ctx, draftPageID, input.SpaceID)
		if err != nil {
			return &SuggestionCreateValidatorResult{Err: err}
		}

		if draftPage == nil || draftPage.SpaceMemberID != input.SpaceMemberID || draftPage.TopicID != input.TopicID {
			formErrors.AddField("draft_page_ids", i18n.T(ctx, "validation_suggestion_draft_page_not_found"))
			return &SuggestionCreateValidatorResult{FormErrors: formErrors}
		}

		draftPages = append(draftPages, draftPage)
	}

	return &SuggestionCreateValidatorResult{
		FormErrors: formErrors,
		DraftPages: draftPages,
	}
}

// SuggestionUpdateValidator は編集提案更新のバリデーションを行う
type SuggestionUpdateValidator struct{}

// NewSuggestionUpdateValidator は SuggestionUpdateValidator を生成する
func NewSuggestionUpdateValidator() *SuggestionUpdateValidator {
	return &SuggestionUpdateValidator{}
}

// SuggestionUpdateValidatorInput はバリデーションの入力パラメータ
type SuggestionUpdateValidatorInput struct {
	Title string
	Body  string
}

// SuggestionUpdateValidatorResult はバリデーションの結果
type SuggestionUpdateValidatorResult struct {
	FormErrors *session.FormErrors
}

// Validate はバリデーションを行う
func (v *SuggestionUpdateValidator) Validate(ctx context.Context, input SuggestionUpdateValidatorInput) *SuggestionUpdateValidatorResult {
	formErrors := session.NewFormErrors()

	// タイトル必須チェック
	if input.Title == "" {
		formErrors.AddField("title", i18n.T(ctx, "validation_suggestion_title_required"))
	}

	// タイトル文字数チェック
	if input.Title != "" && utf8.RuneCountInString(input.Title) > suggestionTitleMaxLength {
		formErrors.AddField("title", i18n.T(ctx, "validation_suggestion_title_too_long"))
	}

	// 本文文字数チェック
	if input.Body != "" && utf8.RuneCountInString(input.Body) > suggestionBodyMaxLength {
		formErrors.AddField("body", i18n.T(ctx, "validation_suggestion_body_too_long"))
	}

	return &SuggestionUpdateValidatorResult{FormErrors: formErrors}
}
