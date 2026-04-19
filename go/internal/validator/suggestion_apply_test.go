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

func TestSuggestionApplyValidator_FormatValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// 形式バリデーションは PageUpdateValidator の形式チェック部分で処理される。
	// 形式違反は DB を使わないため、nil pageRepo で動作する。
	pageUpdateValidator := validator.NewPageUpdateValidator(nil)
	v := validator.NewSuggestionApplyValidator(pageUpdateValidator)

	tests := []struct {
		name  string
		title string
	}{
		{name: "タイトル空文字", title: ""},
		{name: "タイトル200文字超", title: strings.Repeat("あ", 201)},
		{name: "禁止文字スラッシュ", title: "foo/bar"},
		{name: "禁止文字バックスラッシュ", title: "foo\\bar"},
		{name: "禁止文字コロン", title: "foo:bar"},
		{name: "先頭スペース", title: " foo"},
		{name: "末尾ドット", title: "foo."},
		{name: "Windows予約語 CON", title: "CON"},
		{name: "Windows予約語 NUL", title: "NUL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := tt.title
			_, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
				SpaceID:         model.SpaceID("test-space-id"),
				SpaceIdentifier: model.SpaceIdentifier("test-space"),
				Entries: []validator.SuggestionApplyValidatorEntry{
					{
						PageID:  model.PageID("test-page-id"),
						TopicID: model.TopicID("test-topic-id"),
						Title:   &title,
					},
				},
			})

			ae := model.AsSuggestionApplyError(err)
			if ae == nil {
				t.Fatalf("expected SuggestionApplyError but got %T: %v", err, err)
			}
			if len(ae.PageErrors) != 1 {
				t.Errorf("expected 1 page error but got %d: %v", len(ae.PageErrors), ae.PageErrors)
			}
		})
	}
}

func TestSuggestionApplyValidator_Uniqueness(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sa-validator-unique").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		Build()

	// 反映対象の既存ページ
	targetPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Target Page").
		Build()

	pageRepo := repository.NewPageRepository(queries)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	v := validator.NewSuggestionApplyValidator(pageUpdateValidator)

	t.Run("競合なし", func(t *testing.T) {
		title := "Unique Title"
		out, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
			SpaceID:         spaceID,
			SpaceIdentifier: "sa-validator-unique",
			Entries: []validator.SuggestionApplyValidatorEntry{
				{PageID: targetPageID, TopicID: topicID, Title: &title},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out == nil {
			t.Fatal("output should not be nil")
		}
		if len(out.ConflictingPageIDs) != 0 {
			t.Errorf("ConflictingPageIDs should be empty, got %v", out.ConflictingPageIDs)
		}
	})

	t.Run("自ページと同タイトルは衝突とみなさない", func(t *testing.T) {
		title := "Target Page"
		out, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
			SpaceID:         spaceID,
			SpaceIdentifier: "sa-validator-unique",
			Entries: []validator.SuggestionApplyValidatorEntry{
				{PageID: targetPageID, TopicID: topicID, Title: &title},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out == nil {
			t.Fatal("output should not be nil")
		}
		if len(out.ConflictingPageIDs) != 0 {
			t.Errorf("ConflictingPageIDs should be empty, got %v", out.ConflictingPageIDs)
		}
	})

	t.Run("未公開かつ本文が空のページと競合 → 論理削除リストに含まれる", func(t *testing.T) {
		unpublishedID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Empty Unpublished").
			WithBody("").
			WithBodyHTML("").
			WithUnpublished().
			Build()

		title := "Empty Unpublished"
		out, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
			SpaceID:         spaceID,
			SpaceIdentifier: "sa-validator-unique",
			Entries: []validator.SuggestionApplyValidatorEntry{
				{PageID: targetPageID, TopicID: topicID, Title: &title},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out == nil {
			t.Fatal("output should not be nil")
		}
		if len(out.ConflictingPageIDs) != 1 || out.ConflictingPageIDs[0] != unpublishedID {
			t.Errorf("ConflictingPageIDs = %v, want [%v]", out.ConflictingPageIDs, unpublishedID)
		}
	})

	t.Run("公開済みページと競合 → SuggestionApplyError", func(t *testing.T) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(3).
			WithTitle("Published Page").
			Build()

		title := "Published Page"
		_, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
			SpaceID:         spaceID,
			SpaceIdentifier: "sa-validator-unique",
			Entries: []validator.SuggestionApplyValidatorEntry{
				{PageID: targetPageID, TopicID: topicID, Title: &title},
			},
		})

		ae := model.AsSuggestionApplyError(err)
		if ae == nil {
			t.Fatalf("expected SuggestionApplyError but got %T: %v", err, err)
		}
		if len(ae.PageErrors) != 1 {
			t.Fatalf("expected 1 page error, got %d: %v", len(ae.PageErrors), ae.PageErrors)
		}
		// PageTitle に生タイトルが入っている
		if ae.PageErrors[0].PageTitle != "Published Page" {
			t.Errorf("PageTitle = %q, want %q", ae.PageErrors[0].PageTitle, "Published Page")
		}
		// HTML リンクがメッセージに含まれる（@templ.Raw で展開される前提）
		if !strings.Contains(ae.PageErrors[0].Message, "<a") {
			t.Errorf("Message should contain HTML link, got %q", ae.PageErrors[0].Message)
		}
	})

	t.Run("未公開だが本文ありのページと競合 → SuggestionApplyError", func(t *testing.T) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(4).
			WithTitle("Unpublished With Body").
			WithBody("has content").
			WithUnpublished().
			Build()

		title := "Unpublished With Body"
		_, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
			SpaceID:         spaceID,
			SpaceIdentifier: "sa-validator-unique",
			Entries: []validator.SuggestionApplyValidatorEntry{
				{PageID: targetPageID, TopicID: topicID, Title: &title},
			},
		})

		ae := model.AsSuggestionApplyError(err)
		if ae == nil {
			t.Fatalf("expected SuggestionApplyError but got %T: %v", err, err)
		}
		if len(ae.PageErrors) != 1 {
			t.Fatalf("expected 1 page error, got %d: %v", len(ae.PageErrors), ae.PageErrors)
		}
	})
}

func TestSuggestionApplyValidator_MultipleEntries(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("sa-validator-multi").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		Build()

	// 反映対象のページ2件
	pageAID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Source A").
		Build()
	pageBID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Source B").
		Build()

	// 衝突する公開済みページ2件
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Conflict A").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Conflict B").
		Build()

	pageRepo := repository.NewPageRepository(queries)
	pageUpdateValidator := validator.NewPageUpdateValidator(pageRepo)
	v := validator.NewSuggestionApplyValidator(pageUpdateValidator)

	t.Run("複数ページで違反が発生すると全メッセージが集約される", func(t *testing.T) {
		titleA := "Conflict A"
		titleB := "Conflict B"

		_, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
			SpaceID:         spaceID,
			SpaceIdentifier: "sa-validator-multi",
			Entries: []validator.SuggestionApplyValidatorEntry{
				{PageID: pageAID, TopicID: topicID, Title: &titleA},
				{PageID: pageBID, TopicID: topicID, Title: &titleB},
			},
		})

		ae := model.AsSuggestionApplyError(err)
		if ae == nil {
			t.Fatalf("expected SuggestionApplyError but got %T: %v", err, err)
		}
		if len(ae.PageErrors) != 2 {
			t.Errorf("expected 2 page errors, got %d: %v", len(ae.PageErrors), ae.PageErrors)
		}

		titles := make([]string, 0, len(ae.PageErrors))
		for _, pe := range ae.PageErrors {
			titles = append(titles, pe.PageTitle)
		}
		joined := strings.Join(titles, ",")
		if !strings.Contains(joined, "Conflict A") {
			t.Errorf("page errors should mention Conflict A, got %q", joined)
		}
		if !strings.Contains(joined, "Conflict B") {
			t.Errorf("page errors should mention Conflict B, got %q", joined)
		}
	})
}

func TestSuggestionApplyValidator_NilTitleSkipped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	pageUpdateValidator := validator.NewPageUpdateValidator(nil)
	v := validator.NewSuggestionApplyValidator(pageUpdateValidator)

	out, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
		SpaceID:         model.SpaceID("test-space-id"),
		SpaceIdentifier: model.SpaceIdentifier("test-space"),
		Entries: []validator.SuggestionApplyValidatorEntry{
			{
				PageID:  model.PageID("test-page-id"),
				TopicID: model.TopicID("test-topic-id"),
				Title:   nil,
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output should not be nil")
	}
	if len(out.ConflictingPageIDs) != 0 {
		t.Errorf("ConflictingPageIDs should be empty, got %v", out.ConflictingPageIDs)
	}
}

// TestSuggestionApplyValidator_XSSRegression は、タイトルに HTML/JS を含む
// SuggestionPage を反映しようとした場合、PageErrors[i].PageTitle に生文字列が
// そのまま保持されることを確認する。エスケープはテンプレート側の責務であり、
// Validator では行わない（構造化により保持 → テンプレートの自動エスケープに
// 委ねる方針）。
func TestSuggestionApplyValidator_XSSRegression(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	pageUpdateValidator := validator.NewPageUpdateValidator(nil)
	v := validator.NewSuggestionApplyValidator(pageUpdateValidator)

	maliciousTitle := `<script>alert("xss")</script>`
	_, err := v.Validate(ctx, validator.SuggestionApplyValidatorInput{
		SpaceID:         model.SpaceID("test-space-id"),
		SpaceIdentifier: model.SpaceIdentifier("test-space"),
		Entries: []validator.SuggestionApplyValidatorEntry{
			{
				PageID:  model.PageID("test-page-id"),
				TopicID: model.TopicID("test-topic-id"),
				Title:   &maliciousTitle,
			},
		},
	})

	ae := model.AsSuggestionApplyError(err)
	if ae == nil {
		t.Fatalf("expected SuggestionApplyError but got %T: %v", err, err)
	}
	if len(ae.PageErrors) == 0 {
		t.Fatal("expected at least 1 page error")
	}

	// PageTitle には生文字列がそのまま保持される（テンプレート側で自動エスケープされる）
	if ae.PageErrors[0].PageTitle != maliciousTitle {
		t.Errorf("PageTitle = %q, want %q", ae.PageErrors[0].PageTitle, maliciousTitle)
	}
}
