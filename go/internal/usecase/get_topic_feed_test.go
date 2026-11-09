package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetTopicFeedUsecaseForTest(q *query.Queries) *GetTopicFeedUsecase {
	return NewGetTopicFeedUsecase(
		&config.Config{Domain: "wikino.example.com"},
		repository.NewSpaceRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewAttachmentRepository(q),
	)
}

func TestGetTopicFeedUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("トピックのページを新しく公開された順に返し、本文を絶対URLで描画する", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		uc := newGetTopicFeedUsecaseForTest(testutil.QueriesWithTx(tx))

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "gtf-space")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("公開トピック").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		siblingTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("別のトピック").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithFilename("photo.png").
			Build()

		baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		linkedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("リンク先").
			WithPublishedAt(baseTime.Add(1 * time.Hour)).
			Build()
		newestPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("新しいページ").
			WithBody("[[リンク先]]\n\n![写真](/attachments/" + string(attachmentID) + ")").
			WithPublishedAt(baseTime.Add(2 * time.Hour)).
			Build()
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(siblingTopicID).
			WithNumber(3).
			WithTitle("別のトピックのページ").
			WithPublishedAt(baseTime.Add(3 * time.Hour)).
			Build()

		output, err := uc.Execute(context.Background(), GetTopicFeedInput{SpaceIdentifier: "gtf-space", TopicNumber: 1})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.Space.ID != spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, spaceID)
		}
		if output.Topic.ID != topicID {
			t.Errorf("Topic.ID = %v、期待値 = %v", output.Topic.ID, topicID)
		}
		if len(output.Pages) != 2 {
			t.Fatalf("len(Pages) = %d、期待値 = 2", len(output.Pages))
		}
		if output.Pages[0].ID != newestPageID || output.Pages[1].ID != linkedPageID {
			t.Errorf("Pages = [%v, %v]、期待値 = [%v, %v]", output.Pages[0].ID, output.Pages[1].ID, newestPageID, linkedPageID)
		}

		content := output.ContentHTMLByPageID[newestPageID]
		wantContains := []string{
			`href="https://wikino.example.com/s/gtf-space/pages/1"`,
			`src="https://wikino.example.com/attachments/` + string(attachmentID) + `"`,
		}
		for _, want := range wantContains {
			if !strings.Contains(content, want) {
				t.Errorf("本文に %q が含まれていない: %s", want, content)
			}
		}
	})

	t.Run("スペースやトピックが無いとき・トピックが非公開か論理削除されているときはAppErrCodeResourceNotFoundを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		uc := newGetTopicFeedUsecaseForTest(testutil.QueriesWithTx(tx))

		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("gtf-notfound").Build()
		testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("非公開トピック").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("削除したトピック").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			WithDiscarded().
			Build()

		tests := []struct {
			name  string
			input GetTopicFeedInput
		}{
			{name: "スペースが無い", input: GetTopicFeedInput{SpaceIdentifier: "gtf-missing", TopicNumber: 1}},
			{name: "トピックが無い", input: GetTopicFeedInput{SpaceIdentifier: "gtf-notfound", TopicNumber: 99}},
			{name: "トピックが非公開", input: GetTopicFeedInput{SpaceIdentifier: "gtf-notfound", TopicNumber: 1}},
			{name: "トピックが論理削除されている", input: GetTopicFeedInput{SpaceIdentifier: "gtf-notfound", TopicNumber: 2}},
		}

		for _, tt := range tests {
			output, err := uc.Execute(context.Background(), tt.input)
			if output != nil {
				t.Errorf("%s: output = %v、nilを期待", tt.name, output)
			}
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
				t.Errorf("%s: エラー = %v、AppErrCodeResourceNotFoundを期待", tt.name, err)
			}
		}
	})
}
