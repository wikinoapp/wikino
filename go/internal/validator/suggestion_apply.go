package validator

import (
	"context"
	"errors"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SuggestionApplyValidator は編集提案反映時のバリデーションを行う。
// 内部で PageUpdateValidator を各 SuggestionPage に対してループ呼び出しし、
// 全違反を *model.SuggestionApplyError.PageErrors に集約する。
type SuggestionApplyValidator struct {
	pageUpdateValidator *PageUpdateValidator
}

// NewSuggestionApplyValidator は SuggestionApplyValidator を生成する
func NewSuggestionApplyValidator(pageUpdateValidator *PageUpdateValidator) *SuggestionApplyValidator {
	return &SuggestionApplyValidator{pageUpdateValidator: pageUpdateValidator}
}

// SuggestionApplyValidatorInput はバリデーションの入力パラメータ
type SuggestionApplyValidatorInput struct {
	SpaceID         model.SpaceID
	SpaceIdentifier model.SpaceIdentifier
	Entries         []SuggestionApplyValidatorEntry
}

// SuggestionApplyValidatorEntry は反映対象のページ1件分の入力
type SuggestionApplyValidatorEntry struct {
	PageID  model.PageID
	TopicID model.TopicID
	// Title は SuggestionPage.Title。NULL の場合はチェック対象外
	Title *string
}

// SuggestionApplyValidateOutput はバリデーション成功時の出力
type SuggestionApplyValidateOutput struct {
	// ConflictingPageIDs は反映前に論理削除すべき競合ページのID一覧
	ConflictingPageIDs []model.PageID
}

// Validate は各エントリに対して PageUpdateValidator を呼び出し、
// 成功時は論理削除対象のIDを返し、違反がある場合は *model.SuggestionApplyError を返す。
func (v *SuggestionApplyValidator) Validate(
	ctx context.Context,
	input SuggestionApplyValidatorInput,
) (*SuggestionApplyValidateOutput, error) {
	applyErr := &model.SuggestionApplyError{PageErrors: make([]model.SuggestionApplyPageError, 0)}
	conflictingPageIDs := make([]model.PageID, 0)

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
			conflictingPageIDs = append(conflictingPageIDs, *conflictID)
		}
	}

	if len(applyErr.PageErrors) > 0 {
		return nil, applyErr
	}

	return &SuggestionApplyValidateOutput{
		ConflictingPageIDs: conflictingPageIDs,
	}, nil
}

// collectPageErrors は PageUpdateValidator が返した ValidationError のフィールドエラーを、
// 構造化された PageErrors に変換して applyErr に追加する。
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
