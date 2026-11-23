package validator

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// SpaceDeletionCreateValidatorはスペースの削除の確認のバリデーションを行う。
// 削除するスペースの名前を入力させ、取り違えたスペースを消さないようにする。
type SpaceDeletionCreateValidator struct{}

// NewSpaceDeletionCreateValidatorはSpaceDeletionCreateValidatorを生成する
func NewSpaceDeletionCreateValidator() *SpaceDeletionCreateValidator {
	return &SpaceDeletionCreateValidator{}
}

// SpaceDeletionCreateValidatorInputはバリデーションの入力パラメータ
type SpaceDeletionCreateValidatorInput struct {
	// SpaceNameは閲覧者が入力したスペースの名前
	SpaceName string
	// Spaceは削除しようとしているスペース
	Space *model.Space
}

// Validateは、入力したスペースの名前が削除するスペースの名前と一致するかを確かめる。
// 一致は前後の空白も含めた完全一致で判定する。
func (v *SpaceDeletionCreateValidator) Validate(ctx context.Context, input SpaceDeletionCreateValidatorInput) error {
	ve := model.NewValidationError()

	if input.SpaceName == "" {
		ve.AddField("space_name", i18n.T(ctx, "validation_required"))
		return ve
	}
	if input.SpaceName != input.Space.Name {
		ve.AddField("space_name", i18n.T(ctx, "validation_space_deletion_name_mismatch"))
		return ve
	}

	return nil
}
