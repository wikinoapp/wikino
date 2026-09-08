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

func TestTopicCreateValidator_FormatValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	tests := []struct {
		name        string
		topicName   string
		description string
		visibility  string
		field       string
	}{
		{name: "名前が空の場合はエラー", topicName: "", visibility: "public", field: "name"},
		{name: "名前が30文字を超える場合はエラー", topicName: strings.Repeat("あ", 31), visibility: "public", field: "name"},
		{name: "名前にスラッシュを含む場合はエラー", topicName: "foo/bar", visibility: "public", field: "name"},
		{name: "名前にバックスラッシュを含む場合はエラー", topicName: "foo\\bar", visibility: "public", field: "name"},
		{name: "名前にコロンを含む場合はエラー", topicName: "foo:bar", visibility: "public", field: "name"},
		{name: "名前にアスタリスクを含む場合はエラー", topicName: "foo*bar", visibility: "public", field: "name"},
		{name: "名前にクエスチョンマークを含む場合はエラー", topicName: "foo?bar", visibility: "public", field: "name"},
		{name: "名前にダブルクオートを含む場合はエラー", topicName: `foo"bar`, visibility: "public", field: "name"},
		{name: "名前に山括弧を含む場合はエラー", topicName: "foo<bar>", visibility: "public", field: "name"},
		{name: "名前にパイプを含む場合はエラー", topicName: "foo|bar", visibility: "public", field: "name"},
		{name: "名前が先頭スペースの場合はエラー", topicName: " foo", visibility: "public", field: "name"},
		{name: "名前が末尾スペースの場合はエラー", topicName: "foo ", visibility: "public", field: "name"},
		{name: "名前が先頭ドットの場合はエラー", topicName: ".foo", visibility: "public", field: "name"},
		{name: "名前が末尾ドットの場合はエラー", topicName: "foo.", visibility: "public", field: "name"},
		{name: "名前がWindows予約デバイス名の場合はエラー", topicName: "CON", visibility: "public", field: "name"},
		{name: "名前がWindows予約デバイス名 (小文字) の場合はエラー", topicName: "com1", visibility: "public", field: "name"},
		{name: "説明が150文字を超える場合はエラー", topicName: "テスト", description: strings.Repeat("あ", 151), visibility: "public", field: "description"},
		{name: "公開設定が空の場合はエラー", topicName: "テスト", visibility: "", field: "visibility"},
		{name: "公開設定が未知の値の場合はエラー", topicName: "テスト", visibility: "internal", field: "visibility"},
	}

	// 形式バリデーションのみテストするためnilのtopicRepoを使用
	v := validator.NewTopicCreateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Validate(ctx, validator.TopicCreateValidatorInput{
				Name:        tt.topicName,
				Description: tt.description,
				Visibility:  tt.visibility,
				SpaceID:     "test-space-id",
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("expected ValidationError but got nil")
			}
			if !ve.HasFieldError(tt.field) {
				t.Errorf("expected %s field error but got none", tt.field)
			}
		})
	}
}

// TestTopicCreateValidator_AllowedNames covers the names the validator lets through. They reach the
// uniqueness check, which reads the database, so the test runs against a real one.
//
// [Ja] TestTopicCreateValidator_AllowedNames はバリデーターが通す名前を扱う。これらは DB を読む
// 一意性チェックまで進むため、本テストは実際の DB に対して実行する。
func TestTopicCreateValidator_AllowedNames(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-name-chars").
		Build()

	v := validator.NewTopicCreateValidator(repository.NewTopicRepository(queries))

	tests := []struct {
		name       string
		topicName  string
		visibility string
		want       model.TopicVisibility
	}{
		{name: "通常の名前", topicName: "日報", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "非公開トピック", topicName: "議事録", visibility: "private", want: model.TopicVisibilityPrivate},
		{name: "中間にスペースがある名前", topicName: "foo bar", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "中間にドットがある名前", topicName: "foo.bar", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "30文字ちょうどの名前", topicName: strings.Repeat("あ", 30), visibility: "public", want: model.TopicVisibilityPublic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visibility, err := v.Validate(ctx, validator.TopicCreateValidatorInput{
				Name:        tt.topicName,
				Description: strings.Repeat("あ", 150),
				Visibility:  tt.visibility,
				SpaceID:     spaceID,
			})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if visibility != tt.want {
				t.Errorf("visibility = %v, want %v", visibility, tt.want)
			}
		})
	}
}

// TestTopicCreateValidator_Uniqueness covers the name a space already carries. A discarded topic
// keeps holding its name, because the unique index the database enforces covers discarded rows
// too, so the name it holds cannot be given to a new topic either.
//
// [Ja] TestTopicCreateValidator_Uniqueness はスペースが既に持っている名前を扱う。削除済みの
// トピックも名前を持ち続ける。データベースの一意インデックスが削除済みの行も対象にするため、
// その名前を新しいトピックに付けることもできない。
func TestTopicCreateValidator_Uniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-uniqueness").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("日報").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("議事録").
		WithDiscarded().
		Build()

	v := validator.NewTopicCreateValidator(repository.NewTopicRepository(queries))

	tests := []struct {
		name      string
		topicName string
		wantError bool
	}{
		{name: "同じ名前のトピックが既にある場合はエラー", topicName: "日報", wantError: true},
		{name: "削除済みトピックが名前を持っている場合もエラー", topicName: "議事録", wantError: true},
		{name: "使われていない名前は通る", topicName: "週報", wantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Validate(ctx, validator.TopicCreateValidatorInput{
				Name:       tt.topicName,
				Visibility: "public",
				SpaceID:    spaceID,
			})

			if !tt.wantError {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("expected ValidationError but got nil")
			}
			if !ve.HasFieldError("name") {
				t.Error("expected name field error but got none")
			}
		})
	}
}
