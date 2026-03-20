package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSuggestionBodyHTMLUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	topicRepo := repository.NewTopicRepository(q)
	pageRepo := repository.NewPageRepository(q)

	uc := NewGetSuggestionBodyHTMLUsecase(topicRepo, pageRepo)

	t.Run("正常系: Markdownが正しくHTMLに変換される", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("sug-body-html-1").
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionBodyHTMLInput{
			Body:             "**太字**のテスト",
			CurrentTopicName: "General",
			SpaceID:          spaceID,
			SpaceIdentifier:  "sug-body-html-1",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if !strings.Contains(output.BodyHTML, "<strong>太字</strong>") {
			t.Errorf("BodyHTML = %q, want containing <strong>太字</strong>", output.BodyHTML)
		}
	})

	t.Run("正常系: Wikiリンクが解決される", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("sug-body-html-wl").
			Build()
		topicID := testutil.NewTopicBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithName("General").
			Build()
		testutil.NewPageBuilderDB(t, db).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("リンク先ページ").
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionBodyHTMLInput{
			Body:             "[[リンク先ページ]]を参照",
			CurrentTopicName: "General",
			SpaceID:          spaceID,
			SpaceIdentifier:  "sug-body-html-wl",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if !strings.Contains(output.BodyHTML, "/s/sug-body-html-wl/") {
			t.Errorf("BodyHTML should contain resolved wikilink URL, got %q", output.BodyHTML)
		}
	})

	t.Run("正常系: 空の本文でも正しく処理される", func(t *testing.T) {
		t.Parallel()

		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier("sug-body-html-empty").
			Build()

		output, err := uc.Execute(context.Background(), GetSuggestionBodyHTMLInput{
			Body:             "",
			CurrentTopicName: "General",
			SpaceID:          spaceID,
			SpaceIdentifier:  "sug-body-html-empty",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
	})
}
