package validator_test

import (
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestProfileUpdateValidator_Validate(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("profile-update-self@example.com").
		WithAtname("profile_update_self").
		Build()
	testutil.NewUserBuilder(t, tx).
		WithEmail("profile-update-other@example.com").
		WithAtname("profile_update_other").
		Build()
	discardedID := testutil.NewUserBuilder(t, tx).
		WithEmail("profile-update-discarded@example.com").
		WithAtname("profile_update_gone").
		Build()
	if _, err := tx.ExecContext(t.Context(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(discardedID)); err != nil {
		t.Fatalf("ユーザーの削除に失敗: %v", err)
	}

	v := validator.NewProfileUpdateValidator(repository.NewUserRepository(testutil.QueriesWithTx(tx)))

	tests := []struct {
		name        string
		atname      string
		userName    string
		description string
		wantField   string
		wantMessage string
	}{
		{name: "自分の今のアットネームのまま保存できる", atname: "profile_update_self", userName: "名前", description: "自己紹介"},
		{name: "使われていないアットネームに変えられる", atname: "profile_update_new"},
		{name: "名前と自己紹介はちょうど上限まで入力できる", atname: "profile_update_self", userName: strings.Repeat("あ", validator.UserNameMaxLength), description: strings.Repeat("あ", validator.UserDescriptionMaxLength)},
		{name: "アットネームが空", atname: "", wantField: "atname", wantMessage: "アットネームを入力してください"},
		{name: "アットネームの形式", atname: "profile-update", wantField: "atname", wantMessage: "アットネームは英数字とアンダースコアのみ使用できます"},
		{name: "アットネームの長さ", atname: strings.Repeat("a", validator.AtnameMaxLength+1), wantField: "atname", wantMessage: "アットネームは20文字以内で入力してください"},
		{name: "アットネームの予約語", atname: "me", wantField: "atname", wantMessage: "このアットネームは使用できません"},
		{name: "ほかのユーザーのアットネーム", atname: "profile_update_other", wantField: "atname", wantMessage: "このアットネームは既に使用されています"},
		{name: "削除済みのユーザーのアットネーム", atname: "profile_update_gone", wantField: "atname", wantMessage: "このアットネームは既に使用されています"},
		{name: "名前の長さ", atname: "profile_update_self", userName: strings.Repeat("あ", validator.UserNameMaxLength+1), wantField: "name", wantMessage: "名前は30文字以内で入力してください"},
		{name: "自己紹介の長さ", atname: "profile_update_self", description: strings.Repeat("あ", validator.UserDescriptionMaxLength+1), wantField: "description", wantMessage: "説明は150文字以内で入力してください"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
			err := v.Validate(ctx, validator.ProfileUpdateValidatorInput{
				UserID:      userID,
				Atname:      tt.atname,
				Name:        tt.userName,
				Description: tt.description,
			})

			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("ValidationErrorを期待したが、%vだった", err)
			}
			got := ve.GetFieldErrors(tt.wantField)
			if len(got) != 1 || got[0] != tt.wantMessage {
				t.Errorf("%sのエラー = %q、期待値 = %q", tt.wantField, got, tt.wantMessage)
			}
		})
	}
}
