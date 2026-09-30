package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestTrashPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	uc := NewTrashPageUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		pageRepo,
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	// スペースの編集者 (page_trash:writeを持つ)。
	trashMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("tp-trash@example.com").
		WithAtname("tptrash").
		Build()
	// スペースの閲覧者 (page_trash:writeを持たない)。
	writerMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("tp-writer@example.com").
		WithAtname("tpwriter").
		Build()
	// スペースの閲覧者で、非公開トピックのトピック編集者 (page_trash:writeをトピックのロールからだけ得る)。
	topicScopedMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("tp-topic-scoped@example.com").
		WithAtname("tptopicscoped").
		Build()
	// スペースに参加していないログイン済みユーザー。
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("tp-nonmember@example.com").
		WithAtname("tpnonmember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("tp-space").
		WithName("TP Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(trashMemberID).
		WithRole(model.SpaceRoleEditor).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(writerMemberID).
		WithRole(model.SpaceRoleViewer).
		Build()
	topicScopedSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(topicScopedMemberID).
		WithRole(model.SpaceRoleViewer).
		Build()

	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Private").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithSpaceMemberID(topicScopedSpaceMemberID).
		WithRole(model.TopicRoleEditor).
		Build()

	newPage := func(t *testing.T, topicID model.TopicID, number model.PageNumber, title string) {
		t.Helper()

		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(number).
			WithTitle(title).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
	}

	findPage := func(t *testing.T, number model.PageNumber) *model.Page {
		t.Helper()

		page, err := pageRepo.FindBySpaceAndNumber(context.Background(), spaceID, number)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber()のエラー = %v", err)
		}
		if page == nil {
			t.Fatal("FindBySpaceAndNumber()がnilを返した、期待値 = ページ")
		}
		return page
	}

	newPage(t, publicTopicID, 1, "Trashable Page")
	newPage(t, publicTopicID, 2, "Writer Page")
	newPage(t, publicTopicID, 3, "Non Member Page")
	newPage(t, publicTopicID, 4, "Other Topic Page")
	newPage(t, privateTopicID, 5, "Topic Scoped Page")

	t.Run("正常系: 編集者はページをゴミ箱に入れられる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      1,
			UserID:          trashMemberID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
		}
		if output == nil {
			t.Fatal("Execute()がnilの出力を返した")
		}
		// 遷移先をここから組み立てるため、解決済みで返る必要がある。
		if output.Space == nil || output.Space.Identifier != "tp-space" {
			t.Errorf("output.Space = %v、期待値 = スペースtp-space", output.Space)
		}

		page := findPage(t, 1)
		if page.TrashedAt == nil {
			t.Error("page.TrashedAt = nil、期待値 = 打刻された時刻")
		}
		// ゴミ箱は論理削除ではないため、ページはタイトルを保持し、discarded_atはバッチ処理の
		// 担当のまま残る。
		if page.Title == nil || *page.Title != "Trashable Page" {
			t.Errorf("page.Title = %v、期待値 = 'Trashable Page'", page.Title)
		}
		if page.DiscardedAt != nil {
			t.Errorf("page.DiscardedAt = %v、期待値 = nil", page.DiscardedAt)
		}
	})

	t.Run("異常系: 閲覧者はゴミ箱に入れられない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      2,
			UserID:          writerMemberID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("AppErrorを期待したが、nilだった")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeForbidden)
		}
		if page := findPage(t, 2); page.TrashedAt != nil {
			t.Errorf("page.TrashedAt = %v、期待値 = nil (権限が無いので更新されない)", page.TrashedAt)
		}
	})

	t.Run("異常系: スペースメンバーでないユーザーはゴミ箱に入れられない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      3,
			UserID:          nonMemberID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("AppErrorを期待したが、nilだった")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeForbidden)
		}
		if page := findPage(t, 3); page.TrashedAt != nil {
			t.Errorf("page.TrashedAt = %v、期待値 = nil (非メンバーでは更新されない)", page.TrashedAt)
		}
	})

	t.Run("異常系: トピック編集者のロールは他のトピックのページには及ばない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      4,
			UserID:          topicScopedMemberID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("AppErrorを期待したが、nilだった")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeForbidden)
		}
		if page := findPage(t, 4); page.TrashedAt != nil {
			t.Errorf("page.TrashedAt = %v、期待値 = nil (トピック編集者でないトピックのページは更新されない)", page.TrashedAt)
		}
	})

	t.Run("正常系: トピック編集者のロールでもゴミ箱に入れられる", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      5,
			UserID:          topicScopedMemberID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
		}
		if page := findPage(t, 5); page.TrashedAt == nil {
			t.Error("page.TrashedAt = nil、期待値 = 打刻された時刻")
		}
	})

	t.Run("異常系: 存在しないページは見つからない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      999,
			UserID:          trashMemberID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("AppErrorを期待したが、nilだった")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("異常系: 存在しないスペースは見つからない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-nonexistent-space",
			PageNumber:      1,
			UserID:          trashMemberID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("AppErrorを期待したが、nilだった")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})
}
