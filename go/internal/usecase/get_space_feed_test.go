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

func newGetSpaceFeedUsecaseForTest(q *query.Queries) *GetSpaceFeedUsecase {
	return NewGetSpaceFeedUsecase(
		&config.Config{Domain: "wikino.example.com"},
		repository.NewSpaceRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewAttachmentRepository(q),
	)
}

func TestGetSpaceFeedUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックのページを新しく公開された順に返し、本文を絶対URLで描画する", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		uc := newGetSpaceFeedUsecaseForTest(testutil.QueriesWithTx(tx))

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "gsf-space")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("公開トピック").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		privateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("非公開トピック").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithFilename("photo.png").
			Build()

		baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		linkedPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("リンク先").
			WithBody("リンク先の本文").
			WithPublishedAt(baseTime.Add(1 * time.Hour)).
			Build()
		newestPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(publicTopicID).
			WithNumber(2).
			WithTitle("新しいページ").
			WithBody("[[リンク先]]\n\n![写真](/attachments/" + string(attachmentID) + ")").
			WithPublishedAt(baseTime.Add(2 * time.Hour)).
			Build()
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(privateTopicID).
			WithNumber(3).
			WithTitle("非公開のページ").
			WithPublishedAt(baseTime.Add(3 * time.Hour)).
			Build()

		output, err := uc.Execute(context.Background(), GetSpaceFeedInput{SpaceIdentifier: "gsf-space"})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.Space.ID != spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, spaceID)
		}
		if len(output.Pages) != 2 {
			t.Fatalf("len(Pages) = %d、期待値 = 2", len(output.Pages))
		}
		if output.Pages[0].ID != newestPageID || output.Pages[1].ID != linkedPageID {
			t.Errorf("Pages = [%v, %v]、期待値 = [%v, %v]", output.Pages[0].ID, output.Pages[1].ID, newestPageID, linkedPageID)
		}

		content := output.ContentHTMLByPageID[newestPageID]
		wantContains := []string{
			`href="https://wikino.example.com/s/gsf-space/pages/1"`,
			`src="https://wikino.example.com/attachments/` + string(attachmentID) + `"`,
		}
		for _, want := range wantContains {
			if !strings.Contains(content, want) {
				t.Errorf("本文に %q が含まれていない: %s", want, content)
			}
		}
		if strings.Contains(content, "data-attachment-id") {
			t.Errorf("本文に描画用の属性が残っている: %s", content)
		}
		if got := output.ContentHTMLByPageID[linkedPageID]; !strings.Contains(got, "リンク先の本文") {
			t.Errorf("リンク先のページの本文 = %q、「リンク先の本文」を含むことを期待", got)
		}
	})

	t.Run("載せるページが無いときは空の一覧を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		uc := newGetSpaceFeedUsecaseForTest(testutil.QueriesWithTx(tx))

		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("gsf-empty").Build()

		output, err := uc.Execute(context.Background(), GetSpaceFeedInput{SpaceIdentifier: "gsf-empty"})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space.ID != spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, spaceID)
		}
		if len(output.Pages) != 0 {
			t.Errorf("len(Pages) = %d、期待値 = 0", len(output.Pages))
		}
	})

	t.Run("スペースが無いときはAppErrCodeResourceNotFoundを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		uc := newGetSpaceFeedUsecaseForTest(testutil.QueriesWithTx(tx))

		testutil.NewSpaceBuilder(t, tx).WithIdentifier("gsf-discarded").WithDiscarded().Build()

		for _, identifier := range []model.SpaceIdentifier{"gsf-missing", "gsf-discarded"} {
			output, err := uc.Execute(context.Background(), GetSpaceFeedInput{SpaceIdentifier: identifier})
			if output != nil {
				t.Errorf("%s: output = %v、nilを期待", identifier, output)
			}
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
				t.Errorf("%s: エラー = %v、AppErrCodeResourceNotFoundを期待", identifier, err)
			}
		}
	})
}
