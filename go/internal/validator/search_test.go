package validator_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSearchShowValidator_Validate(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewSearchShowValidator()

	tests := []struct {
		name             string
		query            string
		wantError        string
		wantKeywords     []string
		wantSpaceFilters []model.SpaceIdentifier
	}{
		{name: "空なら検索しない", query: ""},
		{name: "空白で分けたキーワードをAND検索の条件にする", query: "日報  2025", wantKeywords: []string{"日報", "2025"}},
		{name: "space:の指定をキーワードから分ける", query: "space:wiki 日報 space:docs", wantKeywords: []string{"日報"}, wantSpaceFilters: []model.SpaceIdentifier{"wiki", "docs"}},
		{name: "space:の指定だけなら検索しない", query: "space:wiki", wantSpaceFilters: []model.SpaceIdentifier{"wiki"}},
		{name: "1文字はエラー", query: "a", wantError: "検索キーワードは2文字以上で入力してください"},
		{name: "2文字なら検索する", query: "日報", wantKeywords: []string{"日報"}},
		{name: "100文字なら検索する", query: strings.Repeat("あ", 100), wantKeywords: []string{strings.Repeat("あ", 100)}},
		{name: "101文字はエラー", query: strings.Repeat("あ", 101), wantError: "検索キーワードは100文字以内で入力してください"},
		{name: "<を含むとエラー", query: "<script>", wantError: "検索キーワードに不正な文字が含まれています"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := v.Validate(ctx, validator.SearchShowValidatorInput{Query: tt.query})

			if tt.wantError != "" {
				ve := model.AsValidationError(err)
				if ve == nil || !slices.Contains(ve.GetFieldErrors("q"), tt.wantError) {
					t.Fatalf("qのエラー %q を期待したが、%vだった", tt.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate()のエラー = %v", err)
			}
			if !slices.Equal(got.Keywords, tt.wantKeywords) {
				t.Errorf("Keywords = %q、期待値 = %q", got.Keywords, tt.wantKeywords)
			}
			if !slices.Equal(got.SpaceIdentifiers, tt.wantSpaceFilters) {
				t.Errorf("SpaceIdentifiers = %q、期待値 = %q", got.SpaceIdentifiers, tt.wantSpaceFilters)
			}
		})
	}
}
