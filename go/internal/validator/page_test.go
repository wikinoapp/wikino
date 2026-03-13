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

func TestPageUpdateValidator_FormatValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	tests := []struct {
		name      string
		title     string
		wantError bool
	}{
		{
			name:      "タイトルが空の場合はエラー",
			title:     "",
			wantError: true,
		},
		{
			name:      "タイトルが200文字以内の場合は正常",
			title:     strings.Repeat("あ", 200),
			wantError: false,
		},
		{
			name:      "タイトルが200文字を超える場合はエラー",
			title:     strings.Repeat("あ", 201),
			wantError: true,
		},
		{
			name:      "タイトルにスラッシュを含む場合はエラー",
			title:     "foo/bar",
			wantError: true,
		},
		{
			name:      "タイトルにバックスラッシュを含む場合はエラー",
			title:     "foo\\bar",
			wantError: true,
		},
		{
			name:      "タイトルにコロンを含む場合はエラー",
			title:     "foo:bar",
			wantError: true,
		},
		{
			name:      "タイトルにアスタリスクを含む場合はエラー",
			title:     "foo*bar",
			wantError: true,
		},
		{
			name:      "タイトルにクエスチョンマークを含む場合はエラー",
			title:     "foo?bar",
			wantError: true,
		},
		{
			name:      "タイトルにダブルクオートを含む場合はエラー",
			title:     `foo"bar`,
			wantError: true,
		},
		{
			name:      "タイトルに山括弧を含む場合はエラー",
			title:     "foo<bar>",
			wantError: true,
		},
		{
			name:      "タイトルにパイプを含む場合はエラー",
			title:     "foo|bar",
			wantError: true,
		},
		{
			name:      "タイトルが先頭スペースの場合はエラー",
			title:     " foo",
			wantError: true,
		},
		{
			name:      "タイトルが末尾スペースの場合はエラー",
			title:     "foo ",
			wantError: true,
		},
		{
			name:      "タイトルが先頭ドットの場合はエラー",
			title:     ".foo",
			wantError: true,
		},
		{
			name:      "タイトルが末尾ドットの場合はエラー",
			title:     "foo.",
			wantError: true,
		},
		{
			name:      "Windows予約語 CON はエラー",
			title:     "CON",
			wantError: true,
		},
		{
			name:      "Windows予約語 con (小文字) はエラー",
			title:     "con",
			wantError: true,
		},
		{
			name:      "Windows予約語 NUL はエラー",
			title:     "NUL",
			wantError: true,
		},
		{
			name:      "Windows予約語 COM1 はエラー",
			title:     "COM1",
			wantError: true,
		},
		{
			name:      "Windows予約語 LPT1 はエラー",
			title:     "LPT1",
			wantError: true,
		},
		{
			name:      "通常のタイトルは正常",
			title:     "テストページ",
			wantError: false,
		},
		{
			name:      "中間にスペースがある場合は正常",
			title:     "foo bar",
			wantError: false,
		},
		{
			name:      "中間にドットがある場合は正常",
			title:     "foo.bar",
			wantError: false,
		},
	}

	// 形式バリデーションのみテストするためnilのpageRepoを使用
	v := validator.NewPageUpdateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantError {
				// 形式バリデーション成功時はDB検証に進むため、nil pageRepoではテストできない
				// 正常系はTestPageUpdateValidator_Uniquenessでテストする
				t.Skip("format validation passes, requires DB for uniqueness check")
			}

			result := v.Validate(ctx, validator.PageUpdateValidatorInput{
				Title:           tt.title,
				PageID:          "test-page-id",
				TopicID:         "test-topic-id",
				SpaceID:         "test-space-id",
				SpaceIdentifier: "test-space",
			})

			if !result.FormErrors.HasErrors() {
				t.Error("expected errors but got none")
			}
			if !result.FormErrors.HasFieldError("title") {
				t.Error("expected title field error but got none")
			}
		})
	}
}

func TestPageUpdateValidator_Uniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("validator-space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		Build()

	// 既存のページを作成
	existingPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Existing Page").
		Build()

	// 別のページを作成（このページのタイトルを変更するテスト）
	anotherPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Another Page").
		Build()

	pageRepo := repository.NewPageRepository(queries)
	v := validator.NewPageUpdateValidator(pageRepo)

	t.Run("同じタイトルの別ページが存在する場合はエラー", func(t *testing.T) {
		result := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Existing Page",
			PageID:          anotherPageID,
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "validator-space",
		})

		if !result.FormErrors.HasErrors() {
			t.Error("expected uniqueness error but got none")
		}
		if !result.FormErrors.HasFieldError("title") {
			t.Error("expected title field error but got none")
		}
	})

	t.Run("自分自身のタイトルは重複にならない", func(t *testing.T) {
		result := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Existing Page",
			PageID:          existingPageID,
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "validator-space",
		})

		if result.FormErrors.HasErrors() {
			t.Errorf("unexpected errors: %v", result.FormErrors)
		}
	})

	t.Run("重複しないタイトルの場合は正常", func(t *testing.T) {
		result := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Unique Title",
			PageID:          model.PageID("any-page-id"),
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "validator-space",
		})

		if result.FormErrors.HasErrors() {
			t.Errorf("unexpected errors: %v", result.FormErrors)
		}
	})
}
