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

// DraftPageSaveValidatorはWeb画面での下書きの保存 (自動保存と手動保存) のバリデーションを行う
type DraftPageSaveValidator struct{}

// NewDraftPageSaveValidatorはDraftPageSaveValidatorを生成する
func NewDraftPageSaveValidator() *DraftPageSaveValidator {
	return &DraftPageSaveValidator{}
}

// DraftPageSaveValidatorInputはバリデーションの入力パラメータ
type DraftPageSaveValidatorInput struct {
	Body string
}

// Validateはバリデーションを行う。
// 下書きは書きかけの状態を保存するため、タイトルの形式は確かめず、公開するときに確かめる。
// 本文の上限は、上限を超えた下書きを公開できないまま溜め込まないよう、保存するときに確かめる
func (v *DraftPageSaveValidator) Validate(ctx context.Context, input DraftPageSaveValidatorInput) error {
	ve := model.NewValidationError()

	validatePageBodyLength(ctx, ve, input.Body)

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// APIDraftPagePublicationCreateValidatorは公開APIでの下書きの公開で、公開する下書きの本文のバリデーションを行う。
// 公開する本文はリクエストに含まれず、保存済みの下書きから読むため、OpenAPI記述のスキーマでは確かめられない。
// 上限が決まる前に保存した下書きや、上限を確かめない経路で保存した下書きを、上限を超えたまま公開しないようにする
type APIDraftPagePublicationCreateValidator struct{}

// NewAPIDraftPagePublicationCreateValidatorはAPIDraftPagePublicationCreateValidatorを生成する
func NewAPIDraftPagePublicationCreateValidator() *APIDraftPagePublicationCreateValidator {
	return &APIDraftPagePublicationCreateValidator{}
}

// APIDraftPagePublicationCreateValidatorInputはバリデーションの入力パラメータ
type APIDraftPagePublicationCreateValidatorInput struct {
	Body string
}

// Validateはバリデーションを行う
func (v *APIDraftPagePublicationCreateValidator) Validate(ctx context.Context, input APIDraftPagePublicationCreateValidatorInput) error {
	ve := model.NewValidationError()

	validatePageBodyLength(ctx, ve, input.Body)

	if ve.HasErrors() {
		return ve
	}
	return nil
}
