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
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	uc := NewGetSpaceShowUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

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
	limitedSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
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
	// Make the limited member join the public topic with no extra topic scopes, so the topic section
	// lists it but CanCreatePageByTopic stays false (the member lacks page:write).
	//
	// [Ja] 限定メンバーを追加のトピックスコープ無しで公開トピックに参加させ、トピックセクションには
	// 表示されるが CanCreatePageByTopic は false のままになる (page:write を持たない) ようにする。
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithSpaceMemberID(limitedSpaceMemberID).
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
		// The public topic label is still resolved for the guest's visible cards.
		// [Ja] ゲストに見えるカードのために公開トピックのラベルは解決される。
		if output.TopicMap[publicTopicID] == nil {
			t.Error("TopicMap should contain the public topic for a guest")
		}
		// A guest cannot edit any page.
		// [Ja] ゲストはどのページも編集できない。
		if output.CanEditPageByTopic[publicTopicID] {
			t.Error("CanEditPageByTopic should be false for a guest")
		}
		// The topic section shows only public topics to a guest, with no create action anywhere.
		// [Ja] トピックセクションはゲストには公開トピックのみを表示し、どこにも作成導線を出さない。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d, want 1 (public topic only)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != publicTopicID {
			t.Errorf("SectionTopics[0].ID = %v, want %v (public topic)", output.SectionTopics[0].ID, publicTopicID)
		}
		if output.CanCreatePageByTopic[publicTopicID] {
			t.Error("CanCreatePageByTopic should be false for a guest")
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
		// A logged-in non-member goes through the same nil-spaceMember path as a guest, so the
		// topic section shows only public topics with no create action anywhere.
		//
		// [Ja] ログイン済み非メンバーはゲストと同じ spaceMember == nil 経路を通るため、トピック
		// セクションには公開トピックのみが表示され、どこにも作成導線は出ない。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d, want 1 (public topic only)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != publicTopicID {
			t.Errorf("SectionTopics[0].ID = %v, want %v (public topic)", output.SectionTopics[0].ID, publicTopicID)
		}
		if output.CanCreatePageByTopic[publicTopicID] {
			t.Error("CanCreatePageByTopic should be false for a logged-in non-member")
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
		// TopicMap covers both topics that the listed pages belong to (for card labels).
		// [Ja] TopicMap は一覧ページが属する両トピックを含む (カードラベル用)。
		if output.TopicMap[publicTopicID] == nil {
			t.Error("TopicMap should contain the public topic")
		}
		if output.TopicMap[privateTopicID] == nil {
			t.Error("TopicMap should contain the private topic")
		}
		// space:admin implies page:write across every topic, so both topics are editable.
		// [Ja] space:admin は全トピックで page:write を含意するため、両トピックとも編集可能。
		if !output.CanEditPageByTopic[publicTopicID] {
			t.Error("CanEditPageByTopic should be true for the public topic for a space:admin member")
		}
		if !output.CanEditPageByTopic[privateTopicID] {
			t.Error("CanEditPageByTopic should be true for the private topic for a space:admin member")
		}
		// The owner has not joined any topic (no topic_member), so FirstJoinedTopic is nil.
		// [Ja] オーナーはどのトピックにも参加していない (topic_member なし) ため FirstJoinedTopic は nil。
		if output.FirstJoinedTopic != nil {
			t.Error("FirstJoinedTopic should be nil for a member who has not joined any topic")
		}
		// The topic section lists the member's joined topics, so it is empty for the owner who has
		// joined none, even though space:admin grants access to every topic's pages.
		//
		// [Ja] トピックセクションはメンバーの参加トピックを並べるため、space:admin が全トピックの
		// ページへのアクセスを与えていても、どのトピックにも参加していないオーナーでは空になる。
		if len(output.SectionTopics) != 0 {
			t.Errorf("len(SectionTopics) = %d, want 0 (owner joined no topic)", len(output.SectionTopics))
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
		// A member with only page:read (no page:write) cannot edit pages in any topic.
		// [Ja] page:read のみ (page:write 無し) のメンバーはどのトピックのページも編集できない。
		if output.CanEditPageByTopic[publicTopicID] {
			t.Error("CanEditPageByTopic should be false for a member without page:write scope")
		}
		if output.CanEditPageByTopic[privateTopicID] {
			t.Error("CanEditPageByTopic should be false for a member without page:write scope")
		}
		// The limited member has joined the public topic, so the section lists it, but without
		// page:write the per-topic create action stays off.
		//
		// [Ja] 限定メンバーは公開トピックに参加しているためセクションに表示されるが、page:write が
		// 無いためトピックごとの作成導線は出ない。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d, want 1 (joined public topic)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != publicTopicID {
			t.Errorf("SectionTopics[0].ID = %v, want %v (public topic)", output.SectionTopics[0].ID, publicTopicID)
		}
		if output.CanCreatePageByTopic[publicTopicID] {
			t.Error("CanCreatePageByTopic should be false for a member without page:write scope")
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
		// The joined member sees the joined public topic in the section and, holding space:admin
		// (which implies page:write), may create a page there.
		//
		// [Ja] 参加メンバーはセクションに参加中の公開トピックを見て、space:admin (page:write を含意) を
		// 持つためそこにページを作成できる。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d, want 1 (joined public topic)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != publicTopicID {
			t.Errorf("SectionTopics[0].ID = %v, want %v (public topic)", output.SectionTopics[0].ID, publicTopicID)
		}
		if !output.CanCreatePageByTopic[publicTopicID] {
			t.Error("CanCreatePageByTopic should be true for a space:admin member in the joined topic")
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
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	uc := NewGetSpaceShowUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

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
		// Even with no pages, the joined topic appears in the section with a create action, which is
		// the per-topic replacement for the old space-level empty-state "new page" button.
		//
		// [Ja] ページが無くても参加トピックは作成導線付きでセクションに現れる。これは旧来の
		// スペースレベル空状態「新規ページ」ボタンを置き換えるトピックごとの導線である。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d, want 1 (joined topic)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != joinedTopicID {
			t.Errorf("SectionTopics[0].ID = %v, want %v (joined topic)", output.SectionTopics[0].ID, joinedTopicID)
		}
		if !output.CanCreatePageByTopic[joinedTopicID] {
			t.Error("CanCreatePageByTopic should be true for a space:admin member in the joined topic")
		}
	})
}
