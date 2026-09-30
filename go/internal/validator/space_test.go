package validator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSpaceCreateValidator_FormatValidation(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name       string
		identifier string
		spaceName  string
		field      string
		wantMsg    string
	}{
		{name: "識別子が空の場合はエラー", identifier: "", spaceName: "テスト", field: "identifier", wantMsg: "識別子を入力してください"},
		{name: "識別子が20文字を超える場合はエラー", identifier: strings.Repeat("a", 21), spaceName: "テスト", field: "identifier", wantMsg: "識別子は20文字以内で入力してください"},
		{name: "識別子に記号を含む場合はエラー", identifier: "a@b", spaceName: "テスト", field: "identifier", wantMsg: "識別子には半角英数字とハイフンのみ使用できます"},
		{name: "識別子にアンダースコアを含む場合はエラー", identifier: "a_b", spaceName: "テスト", field: "identifier", wantMsg: "識別子には半角英数字とハイフンのみ使用できます"},
		{name: "識別子に空白を含む場合はエラー", identifier: "a b", spaceName: "テスト", field: "identifier", wantMsg: "識別子には半角英数字とハイフンのみ使用できます"},
		{name: "識別子に全角文字を含む場合はエラー", identifier: "スペース", spaceName: "テスト", field: "identifier", wantMsg: "識別子には半角英数字とハイフンのみ使用できます"},
		{name: "識別子が予約語の場合はエラー", identifier: "www", spaceName: "テスト", field: "identifier", wantMsg: "この識別子は使用できません"},
		{name: "識別子が大文字の予約語の場合もエラー", identifier: "WWW", spaceName: "テスト", field: "identifier", wantMsg: "この識別子は使用できません"},
		{name: "名前が空の場合はエラー", identifier: "valid", spaceName: "", field: "name", wantMsg: "名前を入力してください"},
		{name: "名前が空白だけの場合はエラー", identifier: "valid", spaceName: "  　", field: "name", wantMsg: "名前を入力してください"},
		{name: "名前が30文字を超える場合はエラー", identifier: "valid", spaceName: strings.Repeat("あ", 31), field: "name", wantMsg: "名前は30文字以内で入力してください"},
	}

	// 形式バリデーションのみテストするためnilのspaceRepoを使用
	v := validator.NewSpaceCreateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(ctx, validator.SpaceCreateValidatorInput{
				Identifier: tt.identifier,
				Name:       tt.spaceName,
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("ValidationErrorを期待したが、%vだった", err)
			}
			msgs := ve.GetFieldErrors(tt.field)
			found := false
			for _, msg := range msgs {
				if msg == tt.wantMsg {
					found = true
				}
			}
			if !found {
				t.Errorf("%sのフィールドエラー = %v、%qが含まれていない", tt.field, msgs, tt.wantMsg)
			}
		})
	}
}

// TestSpaceCreateValidator_Uniquenessは識別子の一意性を扱う。形式を満たす入力はDBを読む
// 一意性チェックまで進むため、本テストは実際のDBに対して実行する。
func TestSpaceCreateValidator_Uniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	v := validator.NewSpaceCreateValidator(repository.NewSpaceRepository(testutil.QueriesWithTx(tx)))

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	testutil.NewSpaceBuilder(t, tx).WithIdentifier("space-uniq-taken").Build()
	testutil.NewSpaceBuilder(t, tx).WithIdentifier("space-uniq-discarded").WithDiscarded().Build()

	tests := []struct {
		name       string
		identifier string
		wantErr    bool
	}{
		{name: "使われていない識別子は通る", identifier: "space-uniq-free", wantErr: false},
		{name: "20文字ちょうどの識別子は通る", identifier: strings.Repeat("b", 20), wantErr: false},
		{name: "使われている識別子はエラー", identifier: "space-uniq-taken", wantErr: true},
		{name: "大文字と小文字だけが違う識別子もエラー", identifier: "Space-Uniq-Taken", wantErr: true},
		{name: "削除済みのスペースの識別子もエラー", identifier: "space-uniq-discarded", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(ctx, validator.SpaceCreateValidatorInput{
				Identifier: tt.identifier,
				Name:       "テスト",
			})

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("予期しないエラー: %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("ValidationErrorを期待したが、%vだった", err)
			}
			if msgs := ve.GetFieldErrors("identifier"); len(msgs) != 1 || msgs[0] != "この識別子は既に使用されています" {
				t.Errorf("identifierのフィールドエラー = %v、期待値 = [この識別子は既に使用されています]", msgs)
			}
		})
	}
}
