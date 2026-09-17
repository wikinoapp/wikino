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

	// スペースオーナー (デフォルトでspace:adminスコープを持つ)。
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gss-owner@example.com").
		WithAtname("gssowner").
		Build()
	// topic:writeを持たない限定スコープのメンバー (CanCreateTopic=falseの検証用)。
	limitedID := testutil.NewUserBuilder(t, tx).
		WithEmail("gss-limited@example.com").
		WithAtname("gsslimited").
		Build()
	// スペースに参加していないログイン済みユーザー (ゲスト相当の挙動の検証用)。
	nonMemberID := testutil.NewUserBuilder(t, tx).
		WithEmail("gss-nonmember@example.com").
		WithAtname("gssnonmember").
		Build()
	// トピックに参加済みのメンバー。ページが存在してもFirstJoinedTopicが取得されることの検証用。
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

	// 参加済みメンバーを公開トピックに参加させ、idが最小の参加トピックを持たせる。
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithSpaceMemberID(joinedMemberSpaceMemberID).
		Build()
	// 限定メンバーを追加のトピックスコープ無しで公開トピックに参加させ、トピックセクションには
	// 表示されるがCanCreatePageByTopicはfalseのままになる (page:writeを持たない) ようにする。
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithSpaceMemberID(limitedSpaceMemberID).
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// 公開トピックのピン留めページと通常ページ。
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

	// 非公開トピックのピン留めページと通常ページ (メンバーのみ閲覧可)。
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output != nil {
			t.Error("存在しないスペースなのに出力がnilではない")
		}
	})

	t.Run("非メンバーは公開トピックのページのみ取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetSpaceShowInput{
			SpaceIdentifier: "gss-space",
			Page:            1,
			PageLimit:       100,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.SpaceMember != nil {
			t.Error("未ログインのユーザーなのにSpaceMemberがnilではない")
		}
		if output.JoinedSpace {
			t.Error("未ログインのユーザーなのにJoinedSpaceがtrue")
		}
		if len(output.PinnedPages) != 1 {
			t.Errorf("len(PinnedPages) = %d、期待値 = 1 (公開トピックのピン留めのみ)", len(output.PinnedPages))
		}
		if len(output.Pages) != 1 {
			t.Errorf("len(Pages) = %d、期待値 = 1 (公開トピックの通常ページのみ)", len(output.Pages))
		}
		if output.TotalCount != 1 {
			t.Errorf("TotalCount = %d、期待値 = 1", output.TotalCount)
		}
		if output.CanCreateTopic {
			t.Error("ゲストなのにCanCreateTopicがtrue")
		}
		if output.FirstJoinedTopic != nil {
			t.Error("ゲストなのにFirstJoinedTopicがnilではない")
		}
		// ゲストに見えるカードのために公開トピックのラベルは解決される。
		if output.TopicMap[publicTopicID] == nil {
			t.Error("ゲストなのにTopicMapに公開トピックが含まれていない")
		}
		// ゲストはどのページも編集できない。
		if output.CanEditPageByTopic[publicTopicID] {
			t.Error("ゲストなのにCanEditPageByTopicがtrue")
		}
		// トピックセクションはゲストには公開トピックのみを表示し、どこにも作成導線を出さない。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d、期待値 = 1 (公開トピックのみ)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != publicTopicID {
			t.Errorf("SectionTopics[0].ID = %v、期待値 = %v (公開トピック)", output.SectionTopics[0].ID, publicTopicID)
		}
		if output.CanCreatePageByTopic[publicTopicID] {
			t.Error("ゲストなのにCanCreatePageByTopicがtrue")
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.SpaceMember != nil {
			t.Error("ログイン中の非メンバーなのにSpaceMemberがnilではない")
		}
		if output.JoinedSpace {
			t.Error("ログイン中の非メンバーなのにJoinedSpaceがtrue")
		}
		if len(output.PinnedPages) != 1 {
			t.Errorf("len(PinnedPages) = %d、期待値 = 1 (公開トピックのピン留めのみ)", len(output.PinnedPages))
		}
		if len(output.Pages) != 1 {
			t.Errorf("len(Pages) = %d、期待値 = 1 (公開トピックの通常ページのみ)", len(output.Pages))
		}
		if output.TotalCount != 1 {
			t.Errorf("TotalCount = %d、期待値 = 1", output.TotalCount)
		}
		if output.CanCreateTopic {
			t.Error("ログイン中の非メンバーなのにCanCreateTopicがtrue")
		}
		if output.FirstJoinedTopic != nil {
			t.Error("ログイン中の非メンバーなのにFirstJoinedTopicがnilではない")
		}
		// ログイン済み非メンバーはゲストと同じspaceMember == nil経路を通るため、トピック
		// セクションには公開トピックのみが表示され、どこにも作成導線は出ない。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d、期待値 = 1 (公開トピックのみ)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != publicTopicID {
			t.Errorf("SectionTopics[0].ID = %v、期待値 = %v (公開トピック)", output.SectionTopics[0].ID, publicTopicID)
		}
		if output.CanCreatePageByTopic[publicTopicID] {
			t.Error("ログイン中の非メンバーなのにCanCreatePageByTopicがtrue")
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.SpaceMember == nil {
			t.Fatal("メンバーなのにSpaceMemberがnil")
		}
		if !output.JoinedSpace {
			t.Error("メンバーなのにJoinedSpaceがfalse")
		}
		if len(output.PinnedPages) != 2 {
			t.Errorf("len(PinnedPages) = %d、期待値 = 2 (公開 + 非公開のピン留め)", len(output.PinnedPages))
		}
		if len(output.Pages) != 2 {
			t.Errorf("len(Pages) = %d、期待値 = 2 (公開 + 非公開の通常ページ)", len(output.Pages))
		}
		if output.TotalCount != 2 {
			t.Errorf("TotalCount = %d、期待値 = 2", output.TotalCount)
		}
		// space:adminはtopic:writeを含意するためCanCreateTopicはtrue。
		if !output.CanCreateTopic {
			t.Error("space:adminメンバーなのにCanCreateTopicがfalse")
		}
		// TopicMapは一覧ページが属する両トピックを含む (カードラベル用)。
		if output.TopicMap[publicTopicID] == nil {
			t.Error("TopicMapに公開トピックが含まれていない")
		}
		if output.TopicMap[privateTopicID] == nil {
			t.Error("TopicMapに非公開トピックが含まれていない")
		}
		// space:adminは全トピックでpage:writeを含意するため、両トピックとも編集可能。
		if !output.CanEditPageByTopic[publicTopicID] {
			t.Error("space:adminメンバーなのに公開トピックのCanEditPageByTopicがfalse")
		}
		if !output.CanEditPageByTopic[privateTopicID] {
			t.Error("space:adminメンバーなのに非公開トピックのCanEditPageByTopicがfalse")
		}
		// オーナーはどのトピックにも参加していない (topic_memberなし) ためFirstJoinedTopicはnil。
		if output.FirstJoinedTopic != nil {
			t.Error("どのトピックにも参加していないメンバーなのにFirstJoinedTopicがnilではない")
		}
		// トピックセクションはメンバーの参加トピックを並べるため、space:adminが全トピックの
		// ページへのアクセスを与えていても、どのトピックにも参加していないオーナーでは空になる。
		if len(output.SectionTopics) != 0 {
			t.Errorf("len(SectionTopics) = %d、期待値 = 0 (オーナーはどのトピックにも参加していない)", len(output.SectionTopics))
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if !output.JoinedSpace {
			t.Error("メンバーなのにJoinedSpaceがfalse")
		}
		if output.CanCreateTopic {
			t.Error("topic:writeスコープを持たないメンバーなのにCanCreateTopicがtrue")
		}
		// page:readのみ (page:write無し) のメンバーはどのトピックのページも編集できない。
		if output.CanEditPageByTopic[publicTopicID] {
			t.Error("page:writeスコープを持たないメンバーなのにCanEditPageByTopicがtrue")
		}
		if output.CanEditPageByTopic[privateTopicID] {
			t.Error("page:writeスコープを持たないメンバーなのにCanEditPageByTopicがtrue")
		}
		// 限定メンバーは公開トピックに参加しているためセクションに表示されるが、page:writeが
		// 無いためトピックごとの作成導線は出ない。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d、期待値 = 1 (参加中の公開トピック)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != publicTopicID {
			t.Errorf("SectionTopics[0].ID = %v、期待値 = %v (公開トピック)", output.SectionTopics[0].ID, publicTopicID)
		}
		if output.CanCreatePageByTopic[publicTopicID] {
			t.Error("page:writeスコープを持たないメンバーなのにCanCreatePageByTopicがtrue")
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if !output.JoinedSpace {
			t.Error("メンバーなのにJoinedSpaceがfalse")
		}
		// ページが存在する状態でもFirstJoinedTopicが取得されることを検証し、
		// 「空状態判定でゲートしない」という実装判断の退行を防ぐ。
		if len(output.Pages) == 0 {
			t.Error("このスペースのメンバーなのにPagesが空")
		}
		if output.FirstJoinedTopic == nil {
			t.Fatal("トピックに参加しているメンバーなのにFirstJoinedTopicがnil")
		}
		if output.FirstJoinedTopic.ID != publicTopicID {
			t.Errorf("FirstJoinedTopic.ID = %v、期待値 = %v", output.FirstJoinedTopic.ID, publicTopicID)
		}
		// 参加メンバーはセクションに参加中の公開トピックを見て、space:admin (page:writeを含意) を
		// 持つためそこにページを作成できる。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d、期待値 = 1 (参加中の公開トピック)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != publicTopicID {
			t.Errorf("SectionTopics[0].ID = %v、期待値 = %v (公開トピック)", output.SectionTopics[0].ID, publicTopicID)
		}
		if !output.CanCreatePageByTopic[publicTopicID] {
			t.Error("参加中のトピックのspace:adminメンバーなのにCanCreatePageByTopicがfalse")
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

	// メンバーが参加するトピック (ページは無い)。
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
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if len(output.PinnedPages) != 0 {
			t.Errorf("len(PinnedPages) = %d、期待値 = 0", len(output.PinnedPages))
		}
		if len(output.Pages) != 0 {
			t.Errorf("len(Pages) = %d、期待値 = 0", len(output.Pages))
		}
		if output.TotalCount != 0 {
			t.Errorf("TotalCount = %d、期待値 = 0", output.TotalCount)
		}
		if output.FirstJoinedTopic == nil {
			t.Fatal("参加中のトピックがあるメンバーなのにFirstJoinedTopicがnil")
		}
		if output.FirstJoinedTopic.ID != joinedTopicID {
			t.Errorf("FirstJoinedTopic.ID = %v、期待値 = %v", output.FirstJoinedTopic.ID, joinedTopicID)
		}
		// ページが無くても参加トピックは作成導線付きでセクションに現れる。これは旧来の
		// スペースレベル空状態「新規ページ」ボタンを置き換えるトピックごとの導線である。
		if len(output.SectionTopics) != 1 {
			t.Fatalf("len(SectionTopics) = %d、期待値 = 1 (参加中のトピック)", len(output.SectionTopics))
		}
		if output.SectionTopics[0].ID != joinedTopicID {
			t.Errorf("SectionTopics[0].ID = %v、期待値 = %v (参加中のトピック)", output.SectionTopics[0].ID, joinedTopicID)
		}
		if !output.CanCreatePageByTopic[joinedTopicID] {
			t.Error("参加中のトピックのspace:adminメンバーなのにCanCreatePageByTopicがfalse")
		}
	})
}
