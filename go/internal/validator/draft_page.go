package validator

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// APIDraftPageCreateValidatorは公開APIでの下書き作成の、下書きを作る先の指定のバリデーションを行う。
// 既存のページの番号と新しいページのトピックの番号のどちらか一方を求める。
// OpenAPI記述のスキーマで「どちらか一方」を表すと生成コードの型が分かれるため、ここで検証する
type APIDraftPageCreateValidator struct{}

// NewAPIDraftPageCreateValidatorはAPIDraftPageCreateValidatorを生成する
func NewAPIDraftPageCreateValidator() *APIDraftPageCreateValidator {
	return &APIDraftPageCreateValidator{}
}

// APIDraftPageCreateValidatorInputはバリデーションの入力パラメータ
type APIDraftPageCreateValidatorInput struct {
	PageNumber  *int32
	TopicNumber *int32
}

// Validateはバリデーションを行う。
// 両方を指定したときは両方のフィールドの問題にし、どちらも無いときは指し示すフィールドが無いため
// 全体の問題にする
func (v *APIDraftPageCreateValidator) Validate(ctx context.Context, input APIDraftPageCreateValidatorInput) error {
	ve := model.NewValidationError()

	switch {
	case input.PageNumber != nil && input.TopicNumber != nil:
		ve.AddField("page_number", i18n.T(ctx, "api_error_draft_page_target_required"))
		ve.AddField("topic_number", i18n.T(ctx, "api_error_draft_page_target_required"))
	case input.PageNumber == nil && input.TopicNumber == nil:
		ve.AddGlobal(i18n.T(ctx, "api_error_draft_page_target_required"))
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}
