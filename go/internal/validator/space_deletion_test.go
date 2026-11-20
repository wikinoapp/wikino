package validator_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSpaceDeletionCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	space := &model.Space{Name: "開発チーム"}

	tests := []struct {
		name      string
		spaceName string
		wantMsg   string
	}{
		{name: "名前が一致すれば通す", spaceName: "開発チーム"},
		{name: "名前が空ならエラー", spaceName: "", wantMsg: "入力してください"},
		{name: "名前が違えばエラー", spaceName: "営業チーム", wantMsg: "スペース名が間違っています"},
		{name: "前後に空白があれば一致しないものとする", spaceName: "開発チーム ", wantMsg: "スペース名が間違っています"},
	}

	v := validator.NewSpaceDeletionCreateValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := v.Validate(ctx, validator.SpaceDeletionCreateValidatorInput{
				SpaceName: tt.spaceName,
				Space:     space,
			})
			if tt.wantMsg == "" {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("ValidationErrorを期待したが、%vだった", err)
			}
			errs := ve.GetFieldErrors("space_name")
			if len(errs) != 1 || errs[0] != tt.wantMsg {
				t.Errorf("space_nameのエラー = %q、期待値 = [%q]", errs, tt.wantMsg)
			}
		})
	}
}
