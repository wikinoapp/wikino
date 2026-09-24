package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPageOgImageUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPageOgImageUsecase(
		repository.NewSpaceRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
	)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpoi-space").
		WithName("OGスペース").
		Build()
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
	publicPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("公開ページ").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("非公開ページ").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(3).
		WithTitle("ゴミ箱のページ").
		WithTrashed().
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(4).
		WithNilTitle().
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(5).
		WithTitle("").
		Build()
	unpublishedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(7).
		WithTitle("未公開のページ").
		WithUnpublished().
		Build()

	otherSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpoi-other").
		Build()
	otherTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithNumber(1).
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	otherPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithTopicID(otherTopicID).
		WithNumber(1).
		WithTitle("別スペースのページ").
		Build()
	// 番号6のページは別のスペースにだけある
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithTopicID(otherTopicID).
		WithNumber(6).
		WithTitle("別スペースだけのページ").
		Build()

	t.Run("正常系: 公開トピックのページはスペース・トピック・ページを返す", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageOgImageInput{
			SpaceIdentifier: "gpoi-space",
			PageNumber:      1,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space == nil || output.Space.Name != "OGスペース" {
			t.Errorf("Space = %v、期待値 = OGスペース", output.Space)
		}
		if output.Topic == nil || output.Topic.ID != publicTopicID {
			t.Errorf("Topic = %v、期待値 = 公開トピック", output.Topic)
		}
		if output.Page == nil || output.Page.ID != publicPageID {
			t.Errorf("Page = %v、期待値のID = %v", output.Page, publicPageID)
		}
	})

	t.Run("正常系: 番号が同じでもスペース識別子で指定したスペースのページを返す", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageOgImageInput{
			SpaceIdentifier: "gpoi-other",
			PageNumber:      1,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Page == nil || output.Page.ID != otherPageID {
			t.Errorf("Page = %v、期待値のID = %v", output.Page, otherPageID)
		}
	})

	t.Run("正常系: published_atがnullの公開トピックのページも返す", func(t *testing.T) {
		// ページ表示画面のIsPublicと同じく、published_atは判定に含めない
		output, err := uc.Execute(context.Background(), GetPageOgImageInput{
			SpaceIdentifier: "gpoi-space",
			PageNumber:      7,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Page == nil || output.Page.ID != unpublishedPageID {
			t.Errorf("Page = %v、期待値のID = %v", output.Page, unpublishedPageID)
		}
	})

	notFoundTests := []struct {
		name            string
		spaceIdentifier model.SpaceIdentifier
		pageNumber      int32
	}{
		{name: "非公開トピックのページ", spaceIdentifier: "gpoi-space", pageNumber: 2},
		{name: "ゴミ箱に入ったページ", spaceIdentifier: "gpoi-space", pageNumber: 3},
		{name: "タイトルがnilのページ", spaceIdentifier: "gpoi-space", pageNumber: 4},
		{name: "タイトルが空文字列のページ", spaceIdentifier: "gpoi-space", pageNumber: 5},
		{name: "存在しないページ番号", spaceIdentifier: "gpoi-space", pageNumber: 999},
		{name: "別のスペースにだけある番号のページ", spaceIdentifier: "gpoi-space", pageNumber: 6},
		{name: "存在しないスペース", spaceIdentifier: "gpoi-missing", pageNumber: 1},
	}
	for _, tt := range notFoundTests {
		t.Run("異常系: "+tt.name+"はAppErrCodeResourceNotFoundを返す", func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetPageOgImageInput{
				SpaceIdentifier: tt.spaceIdentifier,
				PageNumber:      tt.pageNumber,
			})
			if output != nil {
				t.Errorf("出力 = %v、期待値 = nil", output)
			}
			ae := model.AsAppError(err)
			if ae == nil {
				t.Fatalf("エラー = %v、AppErrorを期待", err)
			}
			if ae.Code != model.AppErrCodeResourceNotFound {
				t.Errorf("Code = %d、期待値 = %d", ae.Code, model.AppErrCodeResourceNotFound)
			}
		})
	}
}
