package validator

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// SettingsAccountDeletionCreateValidatorは、アカウントを削除するときの確認の入力と、
// 削除できる状態かを検証する
type SettingsAccountDeletionCreateValidator struct {
	spaceRepo *repository.SpaceRepository
}

// NewSettingsAccountDeletionCreateValidatorはSettingsAccountDeletionCreateValidatorを生成する
func NewSettingsAccountDeletionCreateValidator(spaceRepo *repository.SpaceRepository) *SettingsAccountDeletionCreateValidator {
	return &SettingsAccountDeletionCreateValidator{spaceRepo: spaceRepo}
}

// SettingsAccountDeletionCreateValidatorInputはバリデーションの入力パラメータ
type SettingsAccountDeletionCreateValidatorInput struct {
	// Userは削除しようとしているユーザー (閲覧者本人)
	User *model.User
	// Atnameは確認のために閲覧者が入力したアットネーム
	Atname string
}

// Validateは、入力したアットネームが本人のものと一致し、参加しているスペースが無いかを確かめる。
// アットネームは前後の空白も含めた完全一致で判定する (Rails版の `Accounts::DestroyConfirmationForm` と同じ)。
// 参加しているスペースがあれば、入力によらず削除できないため、フォーム全体の誤りとして示す。
func (v *SettingsAccountDeletionCreateValidator) Validate(ctx context.Context, input SettingsAccountDeletionCreateValidatorInput) error {
	ve := model.NewValidationError()

	if input.Atname == "" {
		ve.AddField("atname", i18n.T(ctx, "validation_required"))
	} else if input.Atname != input.User.Atname {
		ve.AddField("atname", i18n.T(ctx, "validation_account_deletion_atname_mismatch"))
	}

	spaces, err := v.spaceRepo.ListActiveByUser(ctx, input.User.ID)
	if err != nil {
		return err
	}
	if len(spaces) > 0 {
		ve.AddGlobal(i18n.T(ctx, "validation_account_deletion_has_active_spaces"))
	}

	if ve.HasErrors() {
		return ve
	}

	return nil
}
