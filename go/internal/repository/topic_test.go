package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestTopicRepository_FindBySpaceAndNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-find-space").
		WithName("Topic Find Space").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		WithDescription("General topic").
		WithVisibility(0).
		Build()

	t.Run("存在するトピックをスペースIDとナンバーで取得できる", func(t *testing.T) {
		topic, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 1)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if topic == nil {
			t.Fatal("FindBySpaceAndNumber() returned nil, want topic")
		}
		if topic.ID != topicID {
			t.Errorf("topic.ID = %v, want %v", topic.ID, topicID)
		}
		if topic.Space.ID != spaceID {
			t.Errorf("topic.Space.ID = %v, want %v", topic.Space.ID, spaceID)
		}
		if topic.Number != 1 {
			t.Errorf("topic.Number = %v, want 1", topic.Number)
		}
		if topic.Name != "General" {
			t.Errorf("topic.Name = %v, want General", topic.Name)
		}
		if topic.Description != "General topic" {
			t.Errorf("topic.Description = %v, want 'General topic'", topic.Description)
		}
		if topic.Visibility != model.TopicVisibilityPublic {
			t.Errorf("topic.Visibility = %v, want TopicVisibilityPublic", topic.Visibility)
		}
		if topic.DiscardedAt != nil {
			t.Errorf("topic.DiscardedAt = %v, want nil", topic.DiscardedAt)
		}
	})

	t.Run("存在しないナンバーはnilを返す", func(t *testing.T) {
		topic, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 999)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if topic != nil {
			t.Errorf("FindBySpaceAndNumber() = %v, want nil", topic)
		}
	})

	t.Run("存在しないスペースIDはnilを返す", func(t *testing.T) {
		topic, err := repo.FindBySpaceAndNumber(context.Background(), "00000000-0000-0000-0000-000000000000", 1)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber() error = %v", err)
		}
		if topic != nil {
			t.Errorf("FindBySpaceAndNumber() = %v, want nil", topic)
		}
	})
}

func TestTopicRepository_ListActiveBySpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-list-space").
		WithName("Topic List Space").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("First").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Second").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("Discarded").
		WithDiscarded().
		Build()

	t.Run("アクティブなトピック一覧をナンバー順で取得できる", func(t *testing.T) {
		topics, err := repo.ListActiveBySpace(context.Background(), spaceID)
		if err != nil {
			t.Fatalf("ListActiveBySpace() error = %v", err)
		}
		if len(topics) != 2 {
			t.Fatalf("len(topics) = %v, want 2", len(topics))
		}
		if topics[0].Name != "First" {
			t.Errorf("topics[0].Name = %v, want First", topics[0].Name)
		}
		if topics[1].Name != "Second" {
			t.Errorf("topics[1].Name = %v, want Second", topics[1].Name)
		}
	})

	t.Run("トピックがないスペースは空のスライスを返す", func(t *testing.T) {
		topics, err := repo.ListActiveBySpace(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("ListActiveBySpace() error = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v, want 0", len(topics))
		}
	})
}

func TestTopicRepository_FindBySpaceAndNames(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-names-space").
		WithName("Topic Names Space").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Alpha").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Beta").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("Gamma").
		Build()

	t.Run("指定した名前のトピックを取得できる", func(t *testing.T) {
		topics, err := repo.FindBySpaceAndNames(context.Background(), spaceID, []string{"Alpha", "Gamma"})
		if err != nil {
			t.Fatalf("FindBySpaceAndNames() error = %v", err)
		}
		if len(topics) != 2 {
			t.Fatalf("len(topics) = %v, want 2", len(topics))
		}
		names := map[string]bool{}
		for _, topic := range topics {
			names[topic.Name] = true
		}
		if !names["Alpha"] {
			t.Error("Alpha が結果に含まれていない")
		}
		if !names["Gamma"] {
			t.Error("Gamma が結果に含まれていない")
		}
	})

	t.Run("存在しない名前は結果に含まれない", func(t *testing.T) {
		topics, err := repo.FindBySpaceAndNames(context.Background(), spaceID, []string{"NotExist"})
		if err != nil {
			t.Fatalf("FindBySpaceAndNames() error = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v, want 0", len(topics))
		}
	})

	t.Run("空のスライスを渡すと空の結果を返す", func(t *testing.T) {
		topics, err := repo.FindBySpaceAndNames(context.Background(), spaceID, []string{})
		if err != nil {
			t.Fatalf("FindBySpaceAndNames() error = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v, want 0", len(topics))
		}
	})
}

func TestTopicRepository_ListJoinedByUser(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicRepository(q)

	// ユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("joined-by-user@example.com").
		WithAtname("joinedbyuser").
		Build()

	// スペース1を作成
	space1ID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("joined-user-space1").
		WithName("Space One").
		Build()

	spaceMember1ID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(space1ID).
		WithUserID(userID).
		WithRole(0).
		Build()

	// スペース2を作成
	space2ID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("joined-user-space2").
		WithName("Space Two").
		Build()

	spaceMember2ID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(space2ID).
		WithUserID(userID).
		WithRole(0).
		Build()

	// スペース1のトピック（last_page_modified_atあり）
	topic1ID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(space1ID).
		WithNumber(1).
		WithName("Topic Alpha").
		Build()

	recentTime := time.Now()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(space1ID).
		WithTopicID(topic1ID).
		WithSpaceMemberID(spaceMember1ID).
		WithLastPageModifiedAt(recentTime).
		Build()

	// スペース2のトピック（last_page_modified_atが古い）
	topic2ID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(space2ID).
		WithNumber(1).
		WithName("Topic Beta").
		Build()

	olderTime := recentTime.Add(-1 * time.Hour)
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(space2ID).
		WithTopicID(topic2ID).
		WithSpaceMemberID(spaceMember2ID).
		WithLastPageModifiedAt(olderTime).
		Build()

	// スペース1の別トピック（last_page_modified_atなし）
	topic3ID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(space1ID).
		WithNumber(2).
		WithName("Topic Gamma").
		Build()

	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(space1ID).
		WithTopicID(topic3ID).
		WithSpaceMemberID(spaceMember1ID).
		Build()

	t.Run("参加しているトピック一覧をlast_page_modified_at降順で取得できる", func(t *testing.T) {
		topics, err := repo.ListJoinedByUser(context.Background(), userID, 10)
		if err != nil {
			t.Fatalf("ListJoinedByUser() error = %v", err)
		}
		if len(topics) != 3 {
			t.Fatalf("len(topics) = %v, want 3", len(topics))
		}
		// last_page_modified_atが最新のものが先頭
		if topics[0].Name != "Topic Alpha" {
			t.Errorf("topics[0].Name = %v, want 'Topic Alpha'", topics[0].Name)
		}
		// 次にlast_page_modified_atが古いもの
		if topics[1].Name != "Topic Beta" {
			t.Errorf("topics[1].Name = %v, want 'Topic Beta'", topics[1].Name)
		}
		// last_page_modified_atがNULLのものは最後（NULLS LAST）、number DESCで並ぶ
		if topics[2].Name != "Topic Gamma" {
			t.Errorf("topics[2].Name = %v, want 'Topic Gamma'", topics[2].Name)
		}
	})

	t.Run("スペース情報が正しく取得できる", func(t *testing.T) {
		topics, err := repo.ListJoinedByUser(context.Background(), userID, 10)
		if err != nil {
			t.Fatalf("ListJoinedByUser() error = %v", err)
		}
		// Topic Alpha はスペース1に所属
		if topics[0].Space.ID != space1ID {
			t.Errorf("topics[0].Space.ID = %v, want %v", topics[0].Space.ID, space1ID)
		}
		if string(topics[0].Space.Identifier) != "joined-user-space1" {
			t.Errorf("topics[0].Space.Identifier = %v, want 'joined-user-space1'", topics[0].Space.Identifier)
		}
		if topics[0].Space.Name != "Space One" {
			t.Errorf("topics[0].Space.Name = %v, want 'Space One'", topics[0].Space.Name)
		}
		// Topic Beta はスペース2に所属
		if topics[1].Space.ID != space2ID {
			t.Errorf("topics[1].Space.ID = %v, want %v", topics[1].Space.ID, space2ID)
		}
		if string(topics[1].Space.Identifier) != "joined-user-space2" {
			t.Errorf("topics[1].Space.Identifier = %v, want 'joined-user-space2'", topics[1].Space.Identifier)
		}
	})

	t.Run("LIMITが正しく適用される", func(t *testing.T) {
		topics, err := repo.ListJoinedByUser(context.Background(), userID, 2)
		if err != nil {
			t.Fatalf("ListJoinedByUser() error = %v", err)
		}
		if len(topics) != 2 {
			t.Errorf("len(topics) = %v, want 2", len(topics))
		}
	})

	t.Run("非アクティブなスペースメンバーのトピックは除外される", func(t *testing.T) {
		inactiveUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("inactive-member@example.com").
			WithAtname("inactivemember").
			Build()

		inactiveSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("inactive-space").
			WithName("Inactive Space").
			Build()

		inactiveSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(inactiveSpaceID).
			WithUserID(inactiveUserID).
			WithRole(0).
			WithActive(false).
			Build()

		inactiveTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(inactiveSpaceID).
			WithNumber(1).
			WithName("Inactive Topic").
			Build()

		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(inactiveSpaceID).
			WithTopicID(inactiveTopicID).
			WithSpaceMemberID(inactiveSpaceMemberID).
			Build()

		topics, err := repo.ListJoinedByUser(context.Background(), inactiveUserID, 10)
		if err != nil {
			t.Fatalf("ListJoinedByUser() error = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v, want 0", len(topics))
		}
	})

	t.Run("削除済みトピックは除外される", func(t *testing.T) {
		discardedUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("discarded-topic@example.com").
			WithAtname("discardedtopic").
			Build()

		discardedSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("discarded-topic-space").
			WithName("Discarded Topic Space").
			Build()

		discardedSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(discardedSpaceID).
			WithUserID(discardedUserID).
			WithRole(0).
			Build()

		discardedTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(discardedSpaceID).
			WithNumber(1).
			WithName("Discarded Topic").
			WithDiscarded().
			Build()

		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(discardedSpaceID).
			WithTopicID(discardedTopicID).
			WithSpaceMemberID(discardedSpaceMemberID).
			Build()

		topics, err := repo.ListJoinedByUser(context.Background(), discardedUserID, 10)
		if err != nil {
			t.Fatalf("ListJoinedByUser() error = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v, want 0", len(topics))
		}
	})

	t.Run("トピックに参加していないユーザーは空のスライスを返す", func(t *testing.T) {
		topics, err := repo.ListJoinedByUser(context.Background(), "00000000-0000-0000-0000-000000000000", 10)
		if err != nil {
			t.Fatalf("ListJoinedByUser() error = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v, want 0", len(topics))
		}
	})
}

func TestTopicRepository_ListJoinedBySpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("topic-joined@example.com").
		WithAtname("topicjoined").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-joined-space").
		WithName("Topic Joined Space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0).
		Build()

	topicID1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Joined Topic 1").
		Build()

	topicID2 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Joined Topic 2").
		Build()

	// 参加していないトピック
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("Not Joined Topic").
		Build()

	// トピックメンバーをビルダーで作成
	for _, topicID := range []model.TopicID{topicID1, topicID2} {
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(spaceMemberID).
			Build()
	}

	t.Run("参加しているトピック一覧をナンバー順で取得できる", func(t *testing.T) {
		topics, err := repo.ListJoinedBySpaceMember(context.Background(), spaceMemberID, spaceID)
		if err != nil {
			t.Fatalf("ListJoinedBySpaceMember() error = %v", err)
		}
		if len(topics) != 2 {
			t.Fatalf("len(topics) = %v, want 2", len(topics))
		}
		if topics[0].Name != "Joined Topic 1" {
			t.Errorf("topics[0].Name = %v, want 'Joined Topic 1'", topics[0].Name)
		}
		if topics[1].Name != "Joined Topic 2" {
			t.Errorf("topics[1].Name = %v, want 'Joined Topic 2'", topics[1].Name)
		}
	})

	t.Run("トピックに参加していないスペースメンバーは空のスライスを返す", func(t *testing.T) {
		topics, err := repo.ListJoinedBySpaceMember(context.Background(), "00000000-0000-0000-0000-000000000000", spaceID)
		if err != nil {
			t.Fatalf("ListJoinedBySpaceMember() error = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v, want 0", len(topics))
		}
	})
}
