package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetSpaceShowUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	uc := NewGetSpaceShowUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo)

	// Space owner (holds the space:admin scope by default).
	// [Ja] スペースオーナー (デフォルトで space:admin スコープを持つ)。
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gss-owner@example.com").
		WithAtname("gssowner").
		Build()
	// Member with a limited scope that lacks topic:write (used to verify CanCreateTopic=false).
	// [Ja] topic:write を持たない限定スコープのメンバー (CanCreateTopic=false の検証用)。
	limitedID := testutil.NewUserBuilder(t, tx).
		WithEmail("gss-limited@example.com").
		WithAtname("gsslimited").
		Build()
	// Logged-in user who has not joined the space (used to verify guest-equivalent behavior).
	// [Ja] スペースに参加していないログイン済みユーザー (ゲスト相当の挙動の検証用)。
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gss-nonmember@example.com").
		WithAtname("gssnonmember").
		Build()
	// Member who has joined a topic, used to verify FirstJoinedTopic is fetched even when pages exist.
	// [Ja] トピックに参加済みのメンバー。ページが存在しても FirstJoinedTopic が取得されることの検証用。
	joinedMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gss-joined@example.com").
		WithAtname("gssjoined").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gss-space").
		WithName("GSS Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(limitedID).
		WithScopes([]model.Scope{model.ScopePageRead}).
		Build()
	joinedMemberSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(joinedMemberID).
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

	// Make the joined member a member of the public topic so it has a smallest-id joined topic.
	// [Ja] 参加済みメンバーを公開トピックに参加させ、id が最小の参加トピックを持たせる。
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithSpaceMemberID(joinedMemberSpaceMemberID).
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Pinned and regular pages in the public topic.
	// [Ja] 公開トピックのピン留めページと通常ページ。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Public Pinned").
		WithLinkedPageIDs([]model.PageID{}).
		WithPinnedAt(baseTime).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(2).
		WithTitle("Public Regular").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// Pinned and regular pages in the private topic (visible to members only).
	// [Ja] 非公開トピックのピン留めページと通常ページ (メンバーのみ閲覧可)。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(3).
		WithTitle("Private Pinned").
		WithLinkedPageIDs([]model.PageID{}).
		WithPinnedAt(baseTime).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(4).
		WithTitle("Private Regular").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	t.Run("存在しないスペースでnilが返る", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSpaceShowInput{
			SpaceIdentifier: "nonexistent",
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != nil {
			t.Error("output should be nil for non-existent space")
		}
	})

	t.Run("非メンバーは公開トピックのページのみ取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSpaceShowInput{
			SpaceIdentifier: "gss-space",
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SpaceMember != nil {
			t.Error("SpaceMember should be nil for unauthenticated user")
		}
		if output.JoinedSpace {
			t.Error("JoinedSpace should be false for unauthenticated user")
		}
		if len(output.PinnedPages) != 1 {
			t.Errorf("len(PinnedPages) = %d, want 1 (public-topic pinned only)", len(output.PinnedPages))
		}
		if len(output.Pages) != 1 {
			t.Errorf("len(Pages) = %d, want 1 (public-topic regular only)", len(output.Pages))
		}
		if output.TotalCount != 1 {
			t.Errorf("TotalCount = %d, want 1", output.TotalCount)
		}
		if output.CanCreateTopic {
			t.Error("CanCreateTopic should be false for guest")
		}
		if output.FirstJoinedTopic != nil {
			t.Error("FirstJoinedTopic should be nil for guest")
		}
	})

	t.Run("ログイン済みでも非メンバーは公開トピックのページのみ取得できる", func(t *testing.T) {
		userID := nonMemberID
		output, err := uc.Execute(context.Background(), GetSpaceShowInput{
			SpaceIdentifier: "gss-space",
			UserID:          &userID,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SpaceMember != nil {
			t.Error("SpaceMember should be nil for a logged-in non-member")
		}
		if output.JoinedSpace {
			t.Error("JoinedSpace should be false for a logged-in non-member")
		}
		if len(output.PinnedPages) != 1 {
			t.Errorf("len(PinnedPages) = %d, want 1 (public-topic pinned only)", len(output.PinnedPages))
		}
		if len(output.Pages) != 1 {
			t.Errorf("len(Pages) = %d, want 1 (public-topic regular only)", len(output.Pages))
		}
		if output.TotalCount != 1 {
			t.Errorf("TotalCount = %d, want 1", output.TotalCount)
		}
		if output.CanCreateTopic {
			t.Error("CanCreateTopic should be false for a logged-in non-member")
		}
		if output.FirstJoinedTopic != nil {
			t.Error("FirstJoinedTopic should be nil for a logged-in non-member")
		}
	})

	t.Run("メンバーは非公開トピックを含む全ページを取得できる", func(t *testing.T) {
		userID := ownerID
		output, err := uc.Execute(context.Background(), GetSpaceShowInput{
			SpaceIdentifier: "gss-space",
			UserID:          &userID,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMember should not be nil for member")
		}
		if !output.JoinedSpace {
			t.Error("JoinedSpace should be true for member")
		}
		if len(output.PinnedPages) != 2 {
			t.Errorf("len(PinnedPages) = %d, want 2 (public + private pinned)", len(output.PinnedPages))
		}
		if len(output.Pages) != 2 {
			t.Errorf("len(Pages) = %d, want 2 (public + private regular)", len(output.Pages))
		}
		if output.TotalCount != 2 {
			t.Errorf("TotalCount = %d, want 2", output.TotalCount)
		}
		// space:admin implies topic:write, so CanCreateTopic is true.
		// [Ja] space:admin は topic:write を含意するため CanCreateTopic は true。
		if !output.CanCreateTopic {
			t.Error("CanCreateTopic should be true for space:admin member")
		}
		// The owner has not joined any topic (no topic_member), so FirstJoinedTopic is nil.
		// [Ja] オーナーはどのトピックにも参加していない (topic_member なし) ため FirstJoinedTopic は nil。
		if output.FirstJoinedTopic != nil {
			t.Error("FirstJoinedTopic should be nil for a member who has not joined any topic")
		}
	})

	t.Run("topic_writeを持たないメンバーはCanCreateTopicがfalse", func(t *testing.T) {
		userID := limitedID
		output, err := uc.Execute(context.Background(), GetSpaceShowInput{
			SpaceIdentifier: "gss-space",
			UserID:          &userID,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if !output.JoinedSpace {
			t.Error("JoinedSpace should be true for member")
		}
		if output.CanCreateTopic {
			t.Error("CanCreateTopic should be false for a member without topic:write scope")
		}
	})

	t.Run("ページがあるメンバーでも参加トピックがあればFirstJoinedTopicが取得できる", func(t *testing.T) {
		userID := joinedMemberID
		output, err := uc.Execute(context.Background(), GetSpaceShowInput{
			SpaceIdentifier: "gss-space",
			UserID:          &userID,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if !output.JoinedSpace {
			t.Error("JoinedSpace should be true for member")
		}
		// Verify FirstJoinedTopic is fetched even when pages exist, locking in the
		// "not gated on the empty state" decision against regression.
		// [Ja] ページが存在する状態でも FirstJoinedTopic が取得されることを検証し、
		// 「空状態判定でゲートしない」という実装判断の退行を防ぐ。
		if len(output.Pages) == 0 {
			t.Error("Pages should not be empty for a member of this space")
		}
		if output.FirstJoinedTopic == nil {
			t.Fatal("FirstJoinedTopic should not be nil for a member who has joined a topic")
		}
		if output.FirstJoinedTopic.ID != publicTopicID {
			t.Errorf("FirstJoinedTopic.ID = %v, want %v", output.FirstJoinedTopic.ID, publicTopicID)
		}
	})
}

func TestGetSpaceShowUsecase_Execute_空状態(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	uc := NewGetSpaceShowUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("gss-empty@example.com").
		WithAtname("gssempty").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gss-empty").
		WithName("GSS Empty").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	// A topic the member has joined (it has no pages).
	// [Ja] メンバーが参加するトピック (ページは無い)。
	joinedTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Joined Topic").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(joinedTopicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	t.Run("ページが0件のメンバーはFirstJoinedTopicが取得できる", func(t *testing.T) {
		uid := userID
		output, err := uc.Execute(context.Background(), GetSpaceShowInput{
			SpaceIdentifier: "gss-empty",
			UserID:          &uid,
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("output should not be nil")
		}
		if len(output.PinnedPages) != 0 {
			t.Errorf("len(PinnedPages) = %d, want 0", len(output.PinnedPages))
		}
		if len(output.Pages) != 0 {
			t.Errorf("len(Pages) = %d, want 0", len(output.Pages))
		}
		if output.TotalCount != 0 {
			t.Errorf("TotalCount = %d, want 0", output.TotalCount)
		}
		if output.FirstJoinedTopic == nil {
			t.Fatal("FirstJoinedTopic should not be nil for a member with a joined topic")
		}
		if output.FirstJoinedTopic.ID != joinedTopicID {
			t.Errorf("FirstJoinedTopic.ID = %v, want %v", output.FirstJoinedTopic.ID, joinedTopicID)
		}
	})
}
