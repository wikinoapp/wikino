package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestSuggestionApplyPageError_DisplayPageTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		pageTitle string
		want      string
	}{
		{name: "タイトルがあるときはそれを返す", locale: i18n.LangJa, pageTitle: "ページのタイトル", want: "ページのタイトル"},
		{name: "タイトルが空のときは無題を返す", locale: i18n.LangJa, pageTitle: "", want: "無題"},
		{name: "英語ではタイトルが空のときはUntitledを返す", locale: i18n.LangEn, pageTitle: "", want: "Untitled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(t.Context(), tt.locale)
			pageErr := viewmodel.SuggestionApplyPageError{PageTitle: tt.pageTitle, Message: "エラー"}

			if got := pageErr.DisplayPageTitle(ctx); got != tt.want {
				t.Errorf("DisplayPageTitle() = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}
