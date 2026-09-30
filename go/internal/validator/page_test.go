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
		name  string
		title string
	}{
		{name: "タイトルが空の場合はエラー", title: ""},
		{name: "タイトルが200文字を超える場合はエラー", title: strings.Repeat("あ", 201)},
		{name: "タイトルにスラッシュを含む場合はエラー", title: "foo/bar"},
		{name: "タイトルにバックスラッシュを含む場合はエラー", title: "foo\\bar"},
		{name: "タイトルにコロンを含む場合はエラー", title: "foo:bar"},
		{name: "タイトルにタブを含む場合はエラー", title: "foo\tbar"},
		{name: "タイトルにヌル文字を含む場合はエラー", title: "foo\x00bar"},
		{name: "タイトルに改行を含む場合はエラー", title: "foo\nbar"},
		{name: "タイトルが先頭スペースの場合はエラー", title: " foo"},
		{name: "タイトルが末尾スペースの場合はエラー", title: "foo "},
	}

	// 形式バリデーションのみテストするためnilのpageRepoを使用
	v := validator.NewPageUpdateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Validate(ctx, validator.PageUpdateValidatorInput{
				Title:           tt.title,
				PageID:          "test-page-id",
				TopicID:         "test-topic-id",
				SpaceID:         "test-space-id",
				SpaceIdentifier: "test-space",
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("ValidationErrorを期待したが、nilだった")
			}
			if !ve.HasFieldError("title") {
				t.Error("titleのフィールドエラーが無い")
			}
		})
	}
}

// TestPageUpdateValidator_AllowedTitleCharactersはバリデーターが通すタイトルを扱う。
// これらはDBを読む一意性チェックまで進むため、本テストは実際のDBに対して実行する。
func TestPageUpdateValidator_AllowedTitleCharacters(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("title-chars").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		Build()

	pageRepo := repository.NewPageRepository(queries)
	v := validator.NewPageUpdateValidator(pageRepo)

	tests := []struct {
		name  string
		title string
	}{
		{name: "アスタリスクを含むタイトル", title: "foo*bar"},
		{name: "クエスチョンマークを含むタイトル", title: "foo?bar"},
		{name: "ダブルクオートを含むタイトル", title: `foo"bar`},
		{name: "山括弧を含むタイトル", title: "foo<bar>"},
		{name: "パイプを含む記事のタイトル", title: "Annict | 見たアニメを記録して、共有しよう - annict.com"},
		{name: "Obsidianが記法として読む文字を含むタイトル", title: "foo #bar ^baz [qux]"},
		{name: "先頭ドットのタイトル", title: ".foo"},
		{name: "末尾ドットのタイトル", title: "foo."},
		{name: "Windows予約デバイス名CON", title: "CON"},
		{name: "Windows予約デバイス名con (小文字)", title: "con"},
		{name: "Windows予約デバイス名NUL", title: "NUL"},
		{name: "Windows予約デバイス名COM1", title: "COM1"},
		{name: "Windows予約デバイス名LPT1", title: "LPT1"},
		{name: "通常のタイトル", title: "テストページ"},
		{name: "中間にスペースがあるタイトル", title: "foo bar"},
		{name: "中間にドットがあるタイトル", title: "foo.bar"},
		{name: "200文字ちょうどのタイトル", title: strings.Repeat("あ", 200)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflictingPageID, err := v.Validate(ctx, validator.PageUpdateValidatorInput{
				Title:           tt.title,
				PageID:          model.PageID("any-page-id"),
				TopicID:         topicID,
				SpaceID:         spaceID,
				SpaceIdentifier: "title-chars",
			})

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
			}
			if conflictingPageID != nil {
				t.Errorf("conflictingPageID = %v、期待値 = nil", *conflictingPageID)
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

	// 別のページを作成 (このページのタイトルを変更するテスト)
	anotherPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Another Page").
		Build()

	pageRepo := repository.NewPageRepository(queries)
	v := validator.NewPageUpdateValidator(pageRepo)

	t.Run("同じタイトルの別ページが存在する場合はエラー", func(t *testing.T) {
		_, err := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Existing Page",
			PageID:          anotherPageID,
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "validator-space",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("ValidationErrorを期待したが、nilだった")
		}
		if !ve.HasFieldError("title") {
			t.Error("titleのフィールドエラーが無い")
		}
	})

	t.Run("自分自身のタイトルは重複にならない", func(t *testing.T) {
		_, err := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Existing Page",
			PageID:          existingPageID,
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "validator-space",
		})

		if err != nil {
			t.Errorf("予期しないエラー: %v", err)
		}
	})

	t.Run("重複しないタイトルの場合は正常", func(t *testing.T) {
		_, err := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Unique Title",
			PageID:          model.PageID("any-page-id"),
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "validator-space",
		})

		if err != nil {
			t.Errorf("予期しないエラー: %v", err)
		}
	})
}

func TestPageUpdateValidator_UnpublishedConflict(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("unpub-conflict").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		Build()

	// リネーム対象のページ
	renamingPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Original Title").
		Build()

	pageRepo := repository.NewPageRepository(queries)
	v := validator.NewPageUpdateValidator(pageRepo)

	t.Run("未公開かつ本文が空のページと競合する場合はエラーにならず競合ページIDが返る", func(t *testing.T) {
		// 未公開かつ本文が空のページを作成
		unpublishedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(10).
			WithTitle("Target Title").
			WithBody("").
			WithUnpublished().
			Build()

		conflictingPageID, err := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Target Title",
			PageID:          renamingPageID,
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "unpub-conflict",
		})

		if err != nil {
			t.Errorf("予期しないエラー: %v", err)
		}
		if conflictingPageID == nil {
			t.Fatal("conflictingPageIDがnil")
		}
		if *conflictingPageID != unpublishedPageID {
			t.Errorf("conflictingPageID = %v、期待値 = %v", *conflictingPageID, unpublishedPageID)
		}
	})

	t.Run("未公開だが本文がある場合は従来どおりエラー", func(t *testing.T) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(11).
			WithTitle("Has Body").
			WithBody("some content").
			WithUnpublished().
			Build()

		conflictingPageID, err := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Has Body",
			PageID:          renamingPageID,
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "unpub-conflict",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("ValidationErrorを期待したが、nilだった")
		}
		if !ve.HasFieldError("title") {
			t.Error("titleのフィールドエラーが無い")
		}
		if conflictingPageID != nil {
			t.Error("本文が空でないのにconflictingPageIDがnilではない")
		}
	})

	t.Run("公開済みページとの競合は従来どおりエラー", func(t *testing.T) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(12).
			WithTitle("Published Page").
			Build()

		conflictingPageID, err := v.Validate(ctx, validator.PageUpdateValidatorInput{
			Title:           "Published Page",
			PageID:          renamingPageID,
			TopicID:         topicID,
			SpaceID:         spaceID,
			SpaceIdentifier: "unpub-conflict",
		})

		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("ValidationErrorを期待したが、nilだった")
		}
		if !ve.HasFieldError("title") {
			t.Error("titleのフィールドエラーが無い")
		}
		if conflictingPageID != nil {
			t.Error("公開済みページなのにconflictingPageIDがnilではない")
		}
	})
}

func TestPageCreateValidator(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("create-validator").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("公開済み").
		Build()
	unpublishedID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("中身の無い未公開").
		WithBody("").
		WithUnpublished().
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("本文のある未公開").
		WithBody("本文").
		WithUnpublished().
		Build()

	v := validator.NewPageCreateValidator(repository.NewPageRepository(queries))
	validate := func(title string) (*model.PageID, error) {
		return v.Validate(ctx, validator.PageCreateValidatorInput{
			Title:   title,
			TopicID: topicID,
			SpaceID: spaceID,
		})
	}

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("重複しないタイトルは通る", func(t *testing.T) {
		conflictingPageID, err := validate("新しいページ")
		if err != nil || conflictingPageID != nil {
			t.Errorf("Validate() = (%v, %v)、期待値 = (nil, nil)", conflictingPageID, err)
		}
	})

	t.Run("形式の問題はtitleのエラー", func(t *testing.T) {
		_, err := validate("foo/bar")
		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("title") {
			t.Errorf("error = %v、期待値 = titleのValidationError", err)
		}
	})

	for _, title := range []string{"公開済み", "本文のある未公開"} {
		t.Run("重複はHTMLを含まないtitleのエラー: "+title, func(t *testing.T) {
			_, err := validate(title)
			ve := model.AsValidationError(err)
			if ve == nil || !ve.HasFieldError("title") {
				t.Fatalf("error = %v、期待値 = titleのValidationError", err)
			}
			if got := ve.Fields["title"][0]; strings.Contains(got, "<") {
				t.Errorf("メッセージ = %q、期待値 = HTMLを含まない", got)
			}
		})
	}

	t.Run("中身の無い未公開のページとの重複は、そのページのIDを返す", func(t *testing.T) {
		conflictingPageID, err := validate("中身の無い未公開")
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if conflictingPageID == nil || *conflictingPageID != unpublishedID {
			t.Errorf("conflictingPageID = %v、期待値 = %s", conflictingPageID, unpublishedID)
		}
	})
}

func TestAPIPageUpdateValidator(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("api-update-validator").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("更新するページ").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("公開済み").
		Build()
	unpublishedID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("中身の無い未公開").
		WithBody("").
		WithUnpublished().
		Build()

	v := validator.NewAPIPageUpdateValidator(repository.NewPageRepository(queries))
	validate := func(title string) (*model.PageID, error) {
		return v.Validate(ctx, validator.APIPageUpdateValidatorInput{
			Title:   title,
			PageID:  pageID,
			TopicID: topicID,
			SpaceID: spaceID,
		})
	}

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("重複しないタイトルは通る", func(t *testing.T) {
		conflictingPageID, err := validate("新しいタイトル")
		if err != nil || conflictingPageID != nil {
			t.Errorf("Validate() = (%v, %v)、期待値 = (nil, nil)", conflictingPageID, err)
		}
	})

	t.Run("ページ自身のタイトルは重複として扱わない", func(t *testing.T) {
		conflictingPageID, err := validate("更新するページ")
		if err != nil || conflictingPageID != nil {
			t.Errorf("Validate() = (%v, %v)、期待値 = (nil, nil)", conflictingPageID, err)
		}
	})

	t.Run("形式の問題はtitleのエラー", func(t *testing.T) {
		_, err := validate("foo:bar")
		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("title") {
			t.Errorf("error = %v、期待値 = titleのValidationError", err)
		}
	})

	t.Run("他のページとの重複はHTMLを含まないtitleのエラー", func(t *testing.T) {
		_, err := validate("公開済み")
		ve := model.AsValidationError(err)
		if ve == nil || !ve.HasFieldError("title") {
			t.Fatalf("error = %v、期待値 = titleのValidationError", err)
		}
		if got := ve.Fields["title"][0]; strings.Contains(got, "<") {
			t.Errorf("メッセージ = %q、期待値 = HTMLを含まない", got)
		}
	})

	t.Run("中身の無い未公開のページとの重複は、そのページのIDを返す", func(t *testing.T) {
		conflictingPageID, err := validate("中身の無い未公開")
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if conflictingPageID == nil || *conflictingPageID != unpublishedID {
			t.Errorf("conflictingPageID = %v、期待値 = %s", conflictingPageID, unpublishedID)
		}
	})
}
