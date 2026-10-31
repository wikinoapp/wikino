package validator

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SuggestionApplyValidatorは編集提案反映時のバリデーションを行う。
// 内部でPageUpdateValidatorを各SuggestionPageに対してループ呼び出しし、
// 全違反を *model.SuggestionApplyError.PageErrorsに集約する。
type SuggestionApplyValidator struct {
	pageUpdateValidator *PageUpdateValidator
}

// NewSuggestionApplyValidatorはSuggestionApplyValidatorを生成する
func NewSuggestionApplyValidator(pageUpdateValidator *PageUpdateValidator) *SuggestionApplyValidator {
	return &SuggestionApplyValidator{pageUpdateValidator: pageUpdateValidator}
}

// SuggestionApplyValidatorInputはバリデーションの入力パラメータ
type SuggestionApplyValidatorInput struct {
	SpaceID         model.SpaceID
	SpaceIdentifier model.SpaceIdentifier
	Entries         []SuggestionApplyValidatorEntry
}

// SuggestionApplyValidatorEntryは反映対象のページ1件分の入力
type SuggestionApplyValidatorEntry struct {
	PageID  model.PageID
	TopicID model.TopicID
	// TitleはSuggestionPage.Title。NULLの場合はチェック対象外
	Title *string
}

// SuggestionApplyValidateOutputはバリデーション成功時の出力
type SuggestionApplyValidateOutput struct {
	// PageReplacementsは反映前に論理削除すべき競合ページと、そのタイトルを取るページの組の一覧
	PageReplacements []SuggestionApplyPageReplacement
}

// SuggestionApplyPageReplacementは、編集提案の反映で論理削除する競合ページと、
// そのタイトルを取る (競合ページを置き換える) ページの組
type SuggestionApplyPageReplacement struct {
	ConflictingPageID model.PageID
	PageID            model.PageID
}

// Validateは各エントリに対してPageUpdateValidatorを呼び出し、
// 成功時は論理削除する競合ページと置き換えるページの組を返し、違反がある場合は *model.SuggestionApplyErrorを返す。
func (v *SuggestionApplyValidator) Validate(
	ctx context.Context,
	input SuggestionApplyValidatorInput,
) (*SuggestionApplyValidateOutput, error) {
	applyErr := &model.SuggestionApplyError{PageErrors: make([]model.SuggestionApplyPageError, 0)}
	pageReplacements := make([]SuggestionApplyPageReplacement, 0)

	for _, entry := range input.Entries {
		if entry.Title == nil {
			continue
		}

		conflictID, err := v.pageUpdateValidator.Validate(ctx, PageUpdateValidatorInput{
			Title:           *entry.Title,
			PageID:          entry.PageID,
			TopicID:         entry.TopicID,
			SpaceID:         input.SpaceID,
			SpaceIdentifier: input.SpaceIdentifier,
		})
		if err != nil {
			var inner *model.ValidationError
			if errors.As(err, &inner) {
				v.collectPageErrors(applyErr, inner, *entry.Title)
				continue
			}
			return nil, err
		}

		if conflictID != nil {
			pageReplacements = append(pageReplacements, SuggestionApplyPageReplacement{
				ConflictingPageID: *conflictID,
				PageID:            entry.PageID,
			})
		}
	}

	if len(applyErr.PageErrors) > 0 {
		return nil, applyErr
	}

	return &SuggestionApplyValidateOutput{
		PageReplacements: pageReplacements,
	}, nil
}

// collectPageErrorsはPageUpdateValidatorが返したValidationErrorのフィールドエラーを、
// 構造化されたPageErrorsに変換してapplyErrに追加する。
func (v *SuggestionApplyValidator) collectPageErrors(applyErr *model.SuggestionApplyError, inner *model.ValidationError, pageTitle string) {
	for _, msgs := range inner.Fields {
		for _, msg := range msgs {
			applyErr.PageErrors = append(applyErr.PageErrors, model.SuggestionApplyPageError{
				PageTitle: pageTitle,
				Message:   msg,
			})
		}
	}
}
