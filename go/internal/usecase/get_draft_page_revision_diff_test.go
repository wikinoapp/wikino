package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetDraftPageRevisionDiffUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	uc := NewGetDraftPageRevisionDiffUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo, draftPageRepo, draftPageRevisionRepo)

	ctx := context.Background()

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gdprd-owner@example.com").
		WithAtname("gdprdowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gdprd-space").
		WithName("GDPRD Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
		WithVisibility(0). // public. [Ja] 公開
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
		WithTitle("テストページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	// Two revisions, oldest first: v1 then v2. repo.Create stamps time.Now(), and the
	// (created_at, id) total order keeps them stable even within the same microsecond.
	//
	// [Ja] リビジョン 2 件を古い順に作成: v1 → v2。repo.Create は time.Now() を打つが、
	// (created_at, id) の全順序により同一マイクロ秒でも順序は安定する。
	rev1, err := draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
		DraftPageID:   draftPageID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         "Old Title",
		Body:          "line one\n",
		BodyHTML:      "<p>line one</p>",
	})
	if err != nil {
		t.Fatalf("Create() rev1 error = %v", err)
	}
	rev2, err := draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
		DraftPageID:   draftPageID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         "New Title",
		Body:          "line one\nline two\n",
		BodyHTML:      "<p>line one</p><p>line two</p>",
	})
	if err != nil {
		t.Fatalf("Create() rev2 error = %v", err)
	}

	t.Run("対象リビジョンと直前リビジョンを返す", func(t *testing.T) {
		output, err := uc.Execute(ctx, GetDraftPageRevisionDiffInput{
			SpaceIdentifier: "gdprd-space",
			PageNumber:      1,
			RevisionID:      rev2.ID,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Revision == nil || output.Revision.ID != rev2.ID {
			t.Fatalf("Revision = %+v, want ID %s", output.Revision, rev2.ID)
		}
		if output.PreviousRevision == nil || output.PreviousRevision.ID != rev1.ID {
			t.Fatalf("PreviousRevision = %+v, want ID %s", output.PreviousRevision, rev1.ID)
		}
		// rev2 is the newest revision, so it is the current one.
		// [Ja] rev2 は最新リビジョンなので現在のものとして扱われる。
		if !output.IsCurrent {
			t.Error("IsCurrent = false, want true (最新リビジョン)")
		}
	})

	t.Run("最古のリビジョンでは直前リビジョンがnilになる", func(t *testing.T) {
		output, err := uc.Execute(ctx, GetDraftPageRevisionDiffInput{
			SpaceIdentifier: "gdprd-space",
			PageNumber:      1,
			RevisionID:      rev1.ID,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Revision == nil || output.Revision.ID != rev1.ID {
			t.Fatalf("Revision = %+v, want ID %s", output.Revision, rev1.ID)
		}
		if output.PreviousRevision != nil {
			t.Errorf("PreviousRevision = %+v, want nil", output.PreviousRevision)
		}
		// rev1 is older than rev2, so it is not the current revision.
		// [Ja] rev1 は rev2 より古いため現在のリビジョンではない。
		if output.IsCurrent {
			t.Error("IsCurrent = true, want false (最新でないリビジョン)")
		}
	})

	t.Run("存在しないリビジョンIDではResourceNotFoundを返す", func(t *testing.T) {
		_, err := uc.Execute(ctx, GetDraftPageRevisionDiffInput{
			SpaceIdentifier: "gdprd-space",
			PageNumber:      1,
			RevisionID:      model.DraftPageRevisionID("00000000-0000-0000-0000-000000000000"),
			UserID:          ownerID,
		})
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %v", err)
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("スペース非メンバーにはForbiddenを返す", func(t *testing.T) {
		strangerID := testutil.NewUserBuilder(t, tx).
			WithEmail("gdprd-stranger@example.com").
			WithAtname("gdprdstranger").
			Build()

		_, err := uc.Execute(ctx, GetDraftPageRevisionDiffInput{
			SpaceIdentifier: "gdprd-space",
			PageNumber:      1,
			RevisionID:      rev2.ID,
			UserID:          strangerID,
		})
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %v", err)
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("他メンバーの下書きのリビジョンにはResourceNotFoundを返す", func(t *testing.T) {
		// Another member of the same space with their own draft and revision for the same
		// page: the owner must not be able to view it.
		//
		// [Ja] 同じスペースの別メンバーが同じページに自分の下書きとリビジョンを持つ場合、
		// オーナーからは閲覧できないこと。
		otherUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("gdprd-other@example.com").
			WithAtname("gdprdother").
			Build()
		otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(otherUserID).
			Build()
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(otherMemberID).
			Build()
		otherDraftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithPageID(pageID).
			WithSpaceMemberID(otherMemberID).
			Build()
		otherRev, err := draftPageRevisionRepo.Create(ctx, repository.CreateDraftPageRevisionInput{
			DraftPageID:   otherDraftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: otherMemberID,
			Title:         "Other Member Title",
			Body:          "other member body\n",
			BodyHTML:      "<p>other member body</p>",
		})
		if err != nil {
			t.Fatalf("Create() otherRev error = %v", err)
		}

		_, err = uc.Execute(ctx, GetDraftPageRevisionDiffInput{
			SpaceIdentifier: "gdprd-space",
			PageNumber:      1,
			RevisionID:      otherRev.ID,
			UserID:          ownerID,
		})
		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatalf("expected AppError, got %v", err)
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})
}
