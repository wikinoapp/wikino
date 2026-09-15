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
		{name: "名前にタブを含む場合はエラー", topicName: "foo\tbar", visibility: "public", field: "name"},
		{name: "名前にヌル文字を含む場合はエラー", topicName: "foo\x00bar", visibility: "public", field: "name"},
		{name: "名前に改行を含む場合はエラー", topicName: "foo\nbar", visibility: "public", field: "name"},
		{name: "名前が先頭スペースの場合はエラー", topicName: " foo", visibility: "public", field: "name"},
		{name: "名前が末尾スペースの場合はエラー", topicName: "foo ", visibility: "public", field: "name"},
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
				t.Fatal("ValidationErrorを期待したが、nilだった")
			}
			if !ve.HasFieldError(tt.field) {
				t.Errorf("%sのフィールドエラーが無い", tt.field)
			}
		})
	}
}

// TestTopicCreateValidator_AllowedNamesはバリデーターが通す名前を扱う。これらはDBを読む
// 一意性チェックまで進むため、本テストは実際のDBに対して実行する。
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
		{name: "アスタリスクを含む名前", topicName: "foo*bar", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "クエスチョンマークを含む名前", topicName: "foo?bar", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "ダブルクオートを含む名前", topicName: `foo"bar`, visibility: "public", want: model.TopicVisibilityPublic},
		{name: "山括弧を含む名前", topicName: "foo<bar>", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "パイプを含む名前", topicName: "foo|bar", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "Obsidianが記法として読む文字を含む名前", topicName: "foo #bar ^baz [qux]", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "先頭ドットの名前", topicName: ".foo", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "末尾ドットの名前", topicName: "foo.", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "Windows予約デバイス名CON", topicName: "CON", visibility: "public", want: model.TopicVisibilityPublic},
		{name: "Windows予約デバイス名com1 (小文字)", topicName: "com1", visibility: "public", want: model.TopicVisibilityPublic},
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
				t.Errorf("予期しないエラー: %v", err)
			}
			if visibility != tt.want {
				t.Errorf("visibility = %v、期待値 = %v", visibility, tt.want)
			}
		})
	}
}

// TestTopicCreateValidator_Uniquenessはスペースが既に持っている名前を扱う。削除済みの
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
					t.Errorf("予期しないエラー: %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("ValidationErrorを期待したが、nilだった")
			}
			if !ve.HasFieldError("name") {
				t.Error("nameのフィールドエラーが無い")
			}
		})
	}
}

// TestTopicUpdateValidator_FormatValidationは一般設定が作成フォームと共有する形式チェックを
// 扱う。チェック自体はTestTopicCreateValidator_FormatValidationで扱っているため、ここでは更新も
// そこへ到達することだけを確かめる。
func TestTopicUpdateValidator_FormatValidation(t *testing.T) {
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
		{name: "名前にスラッシュを含む場合はエラー", topicName: "foo/bar", visibility: "public", field: "name"},
		{name: "説明が150文字を超える場合はエラー", topicName: "テスト", description: strings.Repeat("あ", 151), visibility: "public", field: "description"},
		{name: "公開設定が未知の値の場合はエラー", topicName: "テスト", visibility: "internal", field: "visibility"},
	}

	// 形式バリデーションのみテストするためnilのtopicRepoを使用
	v := validator.NewTopicUpdateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Validate(ctx, validator.TopicUpdateValidatorInput{
				Name:        tt.topicName,
				Description: tt.description,
				Visibility:  tt.visibility,
				TopicID:     "test-topic-id",
				SpaceID:     "test-space-id",
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("ValidationErrorを期待したが、nilだった")
			}
			if !ve.HasFieldError(tt.field) {
				t.Errorf("%sのフィールドエラーが無い", tt.field)
			}
		})
	}
}

// TestTopicUpdateValidator_Uniquenessはスペースが既に持っている名前を扱う。更新するトピックは
// 自身の名前をそのままにでき、別のトピックが持つ名前は削除済みのものも含めて拒否される。
func TestTopicUpdateValidator_Uniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-update-uniqueness").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("日報").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("週報").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("議事録").
		WithDiscarded().
		Build()

	v := validator.NewTopicUpdateValidator(repository.NewTopicRepository(queries))

	tests := []struct {
		name      string
		topicName string
		wantError bool
	}{
		{name: "自身の名前をそのままにする場合は通る", topicName: "日報", wantError: false},
		{name: "別のトピックが持つ名前はエラー", topicName: "週報", wantError: true},
		{name: "削除済みトピックが名前を持っている場合もエラー", topicName: "議事録", wantError: true},
		{name: "使われていない名前は通る", topicName: "月報", wantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Validate(ctx, validator.TopicUpdateValidatorInput{
				Name:       tt.topicName,
				Visibility: "public",
				TopicID:    topicID,
				SpaceID:    spaceID,
			})

			if !tt.wantError {
				if err != nil {
					t.Errorf("予期しないエラー: %v", err)
				}
				return
			}

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("ValidationErrorを期待したが、nilだった")
			}
			if !ve.HasFieldError("name") {
				t.Error("nameのフィールドエラーが無い")
			}
		})
	}
}
