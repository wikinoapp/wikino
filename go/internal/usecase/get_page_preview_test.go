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
		t.Fatalf("Execute() error = %v", err)
	}
	if output == nil {
		t.Fatal("output should not be nil")
	}
	if output.Title != "My Draft Title" {
		t.Errorf("Title = %q, want %q", output.Title, "My Draft Title")
	}
	if !strings.Contains(output.BodyHTML, "<h1") {
		t.Errorf("BodyHTML should contain rendered heading, got %q", output.BodyHTML)
	}
	if !strings.Contains(output.BodyHTML, "<strong>bold</strong>") {
		t.Errorf("BodyHTML should contain rendered bold text, got %q", output.BodyHTML)
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
	// The page being edited.
	// [Ja] 編集対象のページ
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Editing Page").
		Build()
	// An existing target page for the wiki link.
	// [Ja] Wiki リンクのリンク先となる既存ページ
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
		t.Fatalf("Execute() error = %v", err)
	}
	// 既存ページへの Wiki リンクは <a> タグに変換され、ページ番号 2 の URL を含む。
	if !strings.Contains(output.BodyHTML, "/s/pp-wiki-space/pages/2") {
		t.Errorf("BodyHTML should contain link to existing page, got %q", output.BodyHTML)
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

	// 存在しないページへの Wiki リンクを含む本文でプレビューを生成する。
	output, err := uc.Execute(context.Background(), GetPagePreviewInput{
		SpaceIdentifier: "pp-nopersist-space",
		PageNumber:      1,
		UserID:          userID,
		Body:            "link to [[General/Brand New Page]]",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// 未解決の Wiki リンクは <a> タグに変換されず、プレーンテキストのまま残る。
	if strings.Contains(output.BodyHTML, "<a") {
		t.Errorf("unresolved wiki link should not become a link, got %q", output.BodyHTML)
	}

	// リンク先ページが自動作成されていないこと。
	created, err := pageRepo.FindByTopicAndTitle(context.Background(), topicID, "Brand New Page", spaceID)
	if err != nil {
		t.Fatalf("FindByTopicAndTitle() error = %v", err)
	}
	if created != nil {
		t.Error("preview must not create the linked page")
	}

	// 下書きが作成されていないこと。
	draft, err := draftPageRepo.FindByPageAndMember(context.Background(), pageID, spaceMemberID, spaceID)
	if err != nil {
		t.Fatalf("FindByPageAndMember() error = %v", err)
	}
	if draft != nil {
		t.Error("preview must not create a draft page")
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
		t.Error("output should be nil for a non-member")
	}
	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("expected *model.AppError, got %v", err)
	}
	if ae.Code != model.AppErrCodeForbidden {
		t.Errorf("error code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
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
		t.Error("output should be nil when the page does not exist")
	}
	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatalf("expected *model.AppError, got %v", err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("error code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
	}
}
