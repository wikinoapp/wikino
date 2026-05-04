package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newDeleteDraftPageUsecaseForTest(t *testing.T) (*DeleteDraftPageUsecase, *repository.DraftPageRepository, *repository.DraftPageRevisionRepository) {
	t.Helper()

	db := testutil.GetTestDB()
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	uc := NewDeleteDraftPageUsecase(db, spaceRepo, spaceMemberRepo, draftPageRepo, draftPageRevisionRepo, pageRepo, topicRepo, topicMemberRepo)

	return uc, draftPageRepo, draftPageRevisionRepo
}

func TestDeleteDraftPageUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc, draftPageRepo, draftPageRevisionRepo := newDeleteDraftPageUsecaseForTest(t)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("delete-draft-success").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-draft-success@example.com").
		WithAtname("deletedraftsuccess").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("draft body").
		Build()
	// リビジョンも事前に作成し、削除されることを検証する
	testutil.NewDraftPageRevisionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithDraftPageID(draftPageID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewDraftPageRevisionBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithDraftPageID(draftPageID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	err := uc.Execute(context.Background(), DeleteDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("delete-draft-success"),
		PageNumber:      1,
		UserID:          userID,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// 下書きが削除されていることを確認
	got, err := draftPageRepo.FindByID(context.Background(), draftPageID, spaceID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if got != nil {
		t.Error("削除後は下書きが取得できないべき")
	}

	// リビジョンも削除されていることを確認 (FK 制約があるため、ここが 0 でないと
	// そもそも下書きの削除も失敗する想定)
	count, err := draftPageRevisionRepo.CountByDraftPageID(context.Background(), draftPageID, spaceID)
	if err != nil {
		t.Fatalf("CountByDraftPageID() error = %v", err)
	}
	if count != 0 {
		t.Errorf("削除後のリビジョン件数 = %d, want 0", count)
	}
}

func TestDeleteDraftPageUsecase_Execute_DraftNotFound(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc, _, _ := newDeleteDraftPageUsecaseForTest(t)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("delete-draft-notfound").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-draft-notfound@example.com").
		WithAtname("deletedraftnotfound").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	// DraftPage は作成しない

	err := uc.Execute(context.Background(), DeleteDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("delete-draft-notfound"),
		PageNumber:      1,
		UserID:          userID,
	})

	var ae *model.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *model.AppError, got %T (%v)", err, err)
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("AppErrCode = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
	}
}

func TestDeleteDraftPageUsecase_Execute_NotMember(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc, _, _ := newDeleteDraftPageUsecaseForTest(t)

	// 他人のスペース・他人の下書き
	otherUserID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-draft-other@example.com").
		WithAtname("deletedraftother").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("delete-draft-notmember").
		Build()
	otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(otherUserID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(otherMemberID).
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(otherMemberID).
		WithTopicID(topicID).
		WithBody("other's draft").
		Build()

	// 別ユーザー（スペースメンバーでない）
	intruderID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-draft-intruder@example.com").
		WithAtname("deletedraftintruder").
		Build()

	err := uc.Execute(context.Background(), DeleteDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("delete-draft-notmember"),
		PageNumber:      1,
		UserID:          intruderID,
	})

	var ae *model.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *model.AppError, got %T (%v)", err, err)
	}
	if ae.Code != model.AppErrCodeForbidden {
		t.Errorf("AppErrCode = %v, want %v", ae.Code, model.AppErrCodeForbidden)
	}
}

func TestDeleteDraftPageUsecase_Execute_MissingDeleteScope(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	uc, _, _ := newDeleteDraftPageUsecaseForTest(t)

	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("delete-draft-noscope").
		Build()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-draft-noscope@example.com").
		WithAtname("deletedraftnoscope").
		Build()
	// space:admin を付与せず draft_page:write のみのメンバー
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes([]model.Scope{model.ScopeDraftPageWrite}).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("draft body").
		Build()

	err := uc.Execute(context.Background(), DeleteDraftPageInput{
		SpaceIdentifier: model.SpaceIdentifier("delete-draft-noscope"),
		PageNumber:      1,
		UserID:          userID,
	})

	var ae *model.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *model.AppError, got %T (%v)", err, err)
	}
	if ae.Code != model.AppErrCodeForbidden {
		t.Errorf("AppErrCode = %v, want %v", ae.Code, model.AppErrCodeForbidden)
	}
}
