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

	// Member holding page:trash space-wide (the permission the operation is gated on).
	//
	// [Ja] スペース単位で page:trash を持つメンバー (本操作の判定軸となる権限)。
	trashMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("tp-trash@example.com").
		WithAtname("tptrash").
		Build()
	// Member holding page:write but not page:trash (an editor must not be able to trash).
	//
	// [Ja] page:write は持つが page:trash を持たないメンバー (編集者がゴミ箱に入れられないこと)。
	writerMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("tp-writer@example.com").
		WithAtname("tpwriter").
		Build()
	// Member whose page:trash comes only from a topic membership.
	//
	// [Ja] page:trash をトピックメンバーからだけ得るメンバー。
	topicScopedMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("tp-topic-scoped@example.com").
		WithAtname("tptopicscoped").
		Build()
	// Signed-in user who has not joined the space.
	//
	// [Ja] スペースに参加していないログイン済みユーザー。
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
		WithScopes([]model.Scope{model.ScopePageTrash}).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(writerMemberID).
		WithScopes([]model.Scope{model.ScopePageWrite}).
		Build()
	topicScopedSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(topicScopedMemberID).
		WithScopes([]model.Scope{}).
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
		WithScopes([]model.Scope{model.ScopeTopicRead, model.ScopePageTrash}).
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
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if page == nil {
			t.Fatal("FindBySpaceAndNumber() returned nil, want page")
		}
		return page
	}

	newPage(t, publicTopicID, 1, "Trashable Page")
	newPage(t, publicTopicID, 2, "Writer Page")
	newPage(t, publicTopicID, 3, "Non Member Page")
	newPage(t, privateTopicID, 4, "Private Page")
	newPage(t, privateTopicID, 5, "Topic Scoped Page")

	t.Run("正常系: page:trash を持つメンバーはページをゴミ箱に入れられる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      1,
			UserID:          trashMemberID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}
		if output == nil {
			t.Fatal("Execute() returned nil output")
		}
		// The redirect target is built from these, so both must come back resolved.
		//
		// [Ja] 遷移先をここから組み立てるため、両方とも解決済みで返る必要がある。
		if output.Space == nil || output.Space.Identifier != "tp-space" {
			t.Errorf("output.Space = %v, want the space tp-space", output.Space)
		}
		if output.Topic == nil || output.Topic.Number != 1 {
			t.Errorf("output.Topic = %v, want the public topic (number 1)", output.Topic)
		}

		page := findPage(t, 1)
		if page.TrashedAt == nil {
			t.Error("page.TrashedAt = nil, want a stamped time")
		}
		// The trash is not the logical deletion, so the page keeps its title and stays discardable
		// only by the batch jobs.
		//
		// [Ja] ゴミ箱は論理削除ではないため、ページはタイトルを保持し、discarded_at はバッチ処理の
		// 担当のまま残る。
		if page.Title == nil || *page.Title != "Trashable Page" {
			t.Errorf("page.Title = %v, want 'Trashable Page'", page.Title)
		}
		if page.DiscardedAt != nil {
			t.Errorf("page.DiscardedAt = %v, want nil", page.DiscardedAt)
		}
	})

	t.Run("異常系: page:write だけのメンバーはゴミ箱に入れられない", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      2,
			UserID:          writerMemberID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
		if page := findPage(t, 2); page.TrashedAt != nil {
			t.Errorf("page.TrashedAt = %v, want nil (権限が無いので更新されないべき)", page.TrashedAt)
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
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeForbidden)
		}
		if page := findPage(t, 3); page.TrashedAt != nil {
			t.Errorf("page.TrashedAt = %v, want nil (非メンバーでは更新されないべき)", page.TrashedAt)
		}
	})

	t.Run("異常系: 開けない非公開トピックのページはゴミ箱に入れられない", func(t *testing.T) {
		// The space-wide page:trash is not enough on its own: without topic:read the page must stay
		// indistinguishable from one that does not exist.
		//
		// [Ja] スペース単位の page:trash だけでは足りない。topic:read が無ければ、そのページは
		// 存在しないページと区別が付かないままであるべき。
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      4,
			UserID:          trashMemberID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
		if page := findPage(t, 4); page.TrashedAt != nil {
			t.Errorf("page.TrashedAt = %v, want nil (開けないトピックのページは更新されないべき)", page.TrashedAt)
		}
	})

	t.Run("正常系: トピックスコープの page:trash でもゴミ箱に入れられる", func(t *testing.T) {
		_, err := uc.Execute(context.Background(), TrashPageInput{
			SpaceIdentifier: "tp-space",
			PageNumber:      5,
			UserID:          topicScopedMemberID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v, want nil", err)
		}
		if page := findPage(t, 5); page.TrashedAt == nil {
			t.Error("page.TrashedAt = nil, want a stamped time")
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
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
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
			t.Fatal("expected AppError but got nil")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})
}
