package validator_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSettingsAccountDeletionCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	v := validator.NewSettingsAccountDeletionCreateValidator(repository.NewSpaceRepository(queries))

	loneID := testutil.NewUserBuilder(t, tx).
		WithEmail("validator-account-deletion-lone@example.com").
		WithAtname("validator_deletion_lone").
		Build()
	memberID := testutil.NewUserBuilder(t, tx).
		WithEmail("validator-account-deletion-member@example.com").
		WithAtname("validator_deletion_member").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("validator-account-deletion").Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(memberID).Build()

	// 削除済みのスペースだけに残る行は、参加しているスペースに数えない
	leftoverID := testutil.NewUserBuilder(t, tx).
		WithEmail("validator-account-deletion-leftover@example.com").
		WithAtname("validator_deletion_leftover").
		Build()
	discardedSpaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("validator-account-deletion-gone").WithDiscarded().Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(discardedSpaceID).WithUserID(leftoverID).Build()

	lone := &model.User{ID: loneID, Atname: "validator_deletion_lone"}
	member := &model.User{ID: memberID, Atname: "validator_deletion_member"}
	leftover := &model.User{ID: leftoverID, Atname: "validator_deletion_leftover"}

	testCases := []struct {
		name       string
		input      validator.SettingsAccountDeletionCreateValidatorInput
		wantErr    bool
		wantField  string
		wantGlobal bool
	}{
		{
			name:  "正常系: アットネームが一致し、参加しているスペースが無い",
			input: validator.SettingsAccountDeletionCreateValidatorInput{User: lone, Atname: "validator_deletion_lone"},
		},
		{
			name:  "正常系: 削除済みのスペースの行だけが残っている",
			input: validator.SettingsAccountDeletionCreateValidatorInput{User: leftover, Atname: "validator_deletion_leftover"},
		},
		{
			name:      "異常系: アットネームが空",
			input:     validator.SettingsAccountDeletionCreateValidatorInput{User: lone, Atname: ""},
			wantErr:   true,
			wantField: "atname",
		},
		{
			name:      "異常系: アットネームが一致しない",
			input:     validator.SettingsAccountDeletionCreateValidatorInput{User: lone, Atname: "someone_else"},
			wantErr:   true,
			wantField: "atname",
		},
		{
			name:      "異常系: 前後に空白のあるアットネームは一致とみなさない",
			input:     validator.SettingsAccountDeletionCreateValidatorInput{User: lone, Atname: " validator_deletion_lone"},
			wantErr:   true,
			wantField: "atname",
		},
		{
			name:       "異常系: 参加しているスペースがある",
			input:      validator.SettingsAccountDeletionCreateValidatorInput{User: member, Atname: "validator_deletion_member"},
			wantErr:    true,
			wantGlobal: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			err := v.Validate(ctx, tc.input)

			if !tc.wantErr {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、ValidationErrorを期待", err)
			}
			if tc.wantField != "" && !ve.HasFieldError(tc.wantField) {
				t.Errorf("%qのフィールドエラーが無い", tc.wantField)
			}
			if tc.wantGlobal && len(ve.Global) == 0 {
				t.Error("フォーム全体のエラーが無い")
			}
		})
	}
}
