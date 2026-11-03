package validator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestAPIDraftPageCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	number := int32(1)

	tests := []struct {
		name        string
		input       validator.APIDraftPageCreateValidatorInput
		wantErr     bool
		wantFields  []string
		wantGlobals int
	}{
		{
			name:  "ページの番号だけなら通る",
			input: validator.APIDraftPageCreateValidatorInput{PageNumber: &number},
		},
		{
			name:  "トピックの番号だけなら通る",
			input: validator.APIDraftPageCreateValidatorInput{TopicNumber: &number},
		},
		{
			name:       "両方を指定すると両方のフィールドの問題になる",
			input:      validator.APIDraftPageCreateValidatorInput{PageNumber: &number, TopicNumber: &number},
			wantErr:    true,
			wantFields: []string{"page_number", "topic_number"},
		},
		{
			name:        "どちらも無ければ全体の問題になる",
			input:       validator.APIDraftPageCreateValidatorInput{},
			wantErr:     true,
			wantGlobals: 1,
		},
	}

	v := validator.NewAPIDraftPageCreateValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := v.Validate(ctx, tt.input)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、期待値 = ValidationError", err)
			}
			for _, field := range tt.wantFields {
				if !ve.HasFieldError(field) {
					t.Errorf("%qのフィールドエラーが無い", field)
				}
			}
			if len(ve.Global) != tt.wantGlobals {
				t.Errorf("全体のエラーの数 = %d、期待値 = %d", len(ve.Global), tt.wantGlobals)
			}
		})
	}
}

func TestDraftPageSaveValidator_Validate(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	v := validator.NewDraftPageSaveValidator()

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "本文が空なら通る", body: ""},
		{name: "本文が上限ちょうどなら通る", body: strings.Repeat("あ", model.PageBodyMaxLength)},
		{name: "本文が上限を1字超えるとエラー", body: strings.Repeat("あ", model.PageBodyMaxLength+1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(ctx, validator.DraftPageSaveValidatorInput{Body: tt.body})

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("ValidationErrorを期待したが、%vだった", err)
			}
			want := "本文は500000文字以内で入力してください (現在500001文字)"
			if got := ve.GetFieldErrors("body"); len(got) != 1 || got[0] != want {
				t.Errorf("bodyのフィールドエラー = %q、期待値 = [%q]", got, want)
			}
		})
	}
}
