package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/components"
)

// TestFormErrorSummaryはフィールドを指定しない呼び出しでグローバルエラーだけを表示し、
// nil・空・フィールドだけのエラーでは何も描画しないことを確かめる。
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
				t.Errorf("何も描画しないことを期待したが%qだった", got)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("出力に%qを期待したが%qだった", want, got)
				}
			}
			for _, notWant := range tt.notWantContains {
				if strings.Contains(got, notWant) {
					t.Errorf("出力に%qを期待しなかったが%qだった", notWant, got)
				}
			}
		})
	}
}

func TestFormErrorSummary_指定したフィールドを順に表示する(t *testing.T) {
	t.Parallel()

	formErrors := model.NewValidationError()
	formErrors.AddGlobal("保存できませんでした")
	formErrors.AddField("name", "名前を入力してください")
	formErrors.AddField("identifier", "識別子を入力してください")
	formErrors.AddField("identifier", "識別子の形式が不正です")
	formErrors.AddField("email", "表示対象ではないエラー")

	var body bytes.Buffer
	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)
	if err := components.FormErrorSummary(formErrors, "identifier", "name").Render(ctx, &body); err != nil {
		t.Fatalf("FormErrorSummaryをレンダリングできなかった: %v", err)
	}

	got := body.String()
	for _, want := range []string{
		"保存できませんでした",
		"入力内容を確認してください",
		`href="#identifier">識別子を入力してください</a>`,
		`href="#identifier">識別子の形式が不正です</a>`,
		`href="#name">名前を入力してください</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("出力に%qを期待したが%qだった", want, got)
		}
	}
	if strings.Index(got, "識別子の形式が不正です") > strings.Index(got, "名前を入力してください") {
		t.Error("識別子のエラーが名前のエラーより後ろに表示されている")
	}
	if strings.Contains(got, "表示対象ではないエラー") {
		t.Error("指定していないフィールドのエラーが表示されている")
	}
}

func TestFormErrorSummary_指定したフィールドにエラーが無ければ表示しない(t *testing.T) {
	t.Parallel()

	otherFieldErrors := model.NewValidationError()
	otherFieldErrors.AddField("email", "表示対象ではないエラー")

	for _, formErrors := range []*model.ValidationError{nil, model.NewValidationError(), otherFieldErrors} {
		var body bytes.Buffer
		if err := components.FormErrorSummary(formErrors, "identifier", "name").Render(t.Context(), &body); err != nil {
			t.Fatalf("FormErrorSummaryをレンダリングできなかった: %v", err)
		}
		if got := body.String(); got != "" {
			t.Errorf("何も描画しないことを期待したが%qだった", got)
		}
	}
}
