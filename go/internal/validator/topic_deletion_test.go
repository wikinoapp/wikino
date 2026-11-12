package validator_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestTopicDeletionCreateValidator_Validate(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	topic := &model.Topic{Name: "日報"}

	tests := []struct {
		name      string
		topicName string
		wantMsg   string
	}{
		{name: "名前が一致すれば通す", topicName: "日報"},
		{name: "名前が空ならエラー", topicName: "", wantMsg: "入力してください"},
		{name: "名前が違えばエラー", topicName: "週報", wantMsg: "トピック名が間違っています"},
		{name: "前後に空白があれば一致しないものとする", topicName: " 日報", wantMsg: "トピック名が間違っています"},
	}

	v := validator.NewTopicDeletionCreateValidator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := v.Validate(ctx, validator.TopicDeletionCreateValidatorInput{
				TopicName: tt.topicName,
				Topic:     topic,
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
			errs := ve.GetFieldErrors("topic_name")
			if len(errs) != 1 || errs[0] != tt.wantMsg {
				t.Errorf("topic_nameのエラー = %q、期待値 = [%q]", errs, tt.wantMsg)
			}
		})
	}
}
