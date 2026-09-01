package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

// TestFormErrorSummary owns the contract that callers previously had to repeat: only global errors
// create the inset summary container. Nil, empty, and field-only validation errors render nothing.
//
// [Ja] TestFormErrorSummary は呼び出し側が以前繰り返していた契約を固定する。余白付きの概要領域を
// 作るのは global error だけで、nil・空・フィールドだけの validation error は何も描画しない。
func TestFormErrorSummary(t *testing.T) {
	t.Parallel()

	emptyErrors := model.NewValidationError()
	fieldErrors := model.NewValidationError()
	fieldErrors.AddField("email", "field-error")
	globalErrors := model.NewValidationError()
	globalErrors.AddGlobal("global-error")

	tests := []struct {
		name            string
		formErrors      *model.ValidationError
		wantEmpty       bool
		wantContains    []string
		notWantContains []string
	}{
		{
			name:      "nil",
			wantEmpty: true,
		},
		{
			name:       "エラーなし",
			formErrors: emptyErrors,
			wantEmpty:  true,
		},
		{
			name:            "フィールドエラーだけ",
			formErrors:      fieldErrors,
			wantEmpty:       true,
			notWantContains: []string{"field-error"},
		},
		{
			name:       "グローバルエラー",
			formErrors: globalErrors,
			wantContains: []string{
				`<div class="px-4">`,
				`<div class="alert" data-variant="destructive">`,
				"global-error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var body bytes.Buffer
			if err := components.FormErrorSummary(tt.formErrors).Render(t.Context(), &body); err != nil {
				t.Fatalf("FormErrorSummaryをレンダリングできなかった: %v", err)
			}

			got := body.String()
			if tt.wantEmpty && got != "" {
				t.Errorf("何も描画しないことを期待したが %q だった", got)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("出力に %q を期待したが %q だった", want, got)
				}
			}
			for _, notWant := range tt.notWantContains {
				if strings.Contains(got, notWant) {
					t.Errorf("出力に %q を期待しなかったが %q だった", notWant, got)
				}
			}
		})
	}
}
