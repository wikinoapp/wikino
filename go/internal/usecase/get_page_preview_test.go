package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPagePreviewUsecase_Execute_RendersMarkdown(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPagePreviewUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
	)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pp-render@example.com").
		WithAtname("pprender").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("pp-render-space").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		Build()

	output, err := uc.Execute(context.Background(), GetPagePreviewInput{
		SpaceIdentifier: "pp-render-space",
		PageNumber:      1,
		UserID:          userID,
		Title:           "My Draft Title",
		Body:            "# Heading\n\nsome **bold** text",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output == nil {
		t.Fatal("出力がnil")
	}
	if output.Title != "My Draft Title" {
		t.Errorf("Title = %q、期待値 = %q", output.Title, "My Draft Title")
	}
	if !strings.Contains(output.BodyHTML, "<h1") {
		t.Errorf("BodyHTMLに描画された見出しが含まれていない: %q", output.BodyHTML)
	}
	if !strings.Contains(output.BodyHTML, "<strong>bold</strong>") {
		t.Errorf("BodyHTMLに描画された太字が含まれていない: %q", output.BodyHTML)
	}
}

func TestGetPagePreviewUsecase_Execute_ResolvesExistingWikilink(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPagePreviewUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
	)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pp-wiki@example.com").
		WithAtname("ppwiki").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("pp-wiki-space").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	// 編集対象のページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		Build()
	// Wikiリンクのリンク先となる既存ページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Existing Target").
		Build()

	output, err := uc.Execute(context.Background(), GetPagePreviewInput{
		SpaceIdentifier: "pp-wiki-space",
		PageNumber:      1,
		UserID:          userID,
		Body:            "see [[General/Existing Target]]",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	// 既存ページへのWikiリンクは <a> タグに変換され、ページ番号2のURLを含む。
	if !strings.Contains(output.BodyHTML, "/s/pp-wiki-space/pages/2") {
		t.Errorf("BodyHTMLに既存ページへのリンクが含まれていない: %q", output.BodyHTML)
	}
}

func TestGetPagePreviewUsecase_Execute_NoPersistence(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	uc := NewGetPagePreviewUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		pageRepo,
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
	)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pp-nopersist@example.com").
		WithAtname("ppnopersist").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("pp-nopersist-space").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		Build()

	// 存在しないページへのWikiリンクを含む本文でプレビューを生成する。
	output, err := uc.Execute(context.Background(), GetPagePreviewInput{
		SpaceIdentifier: "pp-nopersist-space",
		PageNumber:      1,
		UserID:          userID,
		Body:            "link to [[General/Brand New Page]]",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	// 未解決のWikiリンクは <a> タグに変換されず、プレーンテキストのまま残る。
	if strings.Contains(output.BodyHTML, "<a") {
		t.Errorf("未解決のWikiリンクがリンクになっている: %q", output.BodyHTML)
	}

	// リンク先ページが自動作成されていないこと。
	created, err := pageRepo.FindByTopicAndTitle(context.Background(), topicID, "Brand New Page", spaceID)
	if err != nil {
		t.Fatalf("FindByTopicAndTitle()のエラー = %v", err)
	}
	if created != nil {
		t.Error("プレビューでリンク先ページが作成された")
	}

	// 下書きが作成されていないこと。
	draft, err := draftPageRepo.FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
	if err != nil {
		t.Fatalf("FindByPageAndMember()のエラー = %v", err)
	}
	if draft != nil {
		t.Error("プレビューで下書きが作成された")
	}
}

func TestGetPagePreviewUsecase_Execute_NonMemberForbidden(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPagePreviewUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
	)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("pp-owner@example.com").
		WithAtname("ppowner").
		Build()
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("pp-outsider@example.com").
		WithAtname("ppoutsider").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("pp-forbidden-space").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithVisibility(0).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		Build()

	output, err := uc.Execute(context.Background(), GetPagePreviewInput{
		SpaceIdentifier: "pp-forbidden-space",
		PageNumber:      1,
		UserID:          outsiderID,
		Body:            "secret content",
	})
	if output != nil {
		t.Error("非メンバーなのに出力がnilではない")
	}
	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("*model.AppErrorを期待したが、%vだった", err)
	}
	if ae.Code != model.AppErrCodeForbidden {
		t.Errorf("エラーコード = %v、期待値 = %v", ae.Code, model.AppErrCodeForbidden)
	}
}

func TestGetPagePreviewUsecase_Execute_PageNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPagePreviewUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
		repository.NewAttachmentRepository(q),
	)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("pp-notfound@example.com").
		WithAtname("ppnotfound").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("pp-notfound-space").Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	output, err := uc.Execute(context.Background(), GetPagePreviewInput{
		SpaceIdentifier: "pp-notfound-space",
		PageNumber:      999,
		UserID:          userID,
		Body:            "content",
	})
	if output != nil {
		t.Error("ページが存在しないのに出力がnilではない")
	}
	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("*model.AppErrorを期待したが、%vだった", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("エラーコード = %v、期待値 = %v", ae.Code, model.AppErrCodeResourceNotFound)
	}
}
