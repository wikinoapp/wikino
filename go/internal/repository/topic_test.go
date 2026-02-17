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
		if topic.SpaceID != spaceID {
			t.Errorf("topic.SpaceID = %v, want %v", topic.SpaceID, spaceID)
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

	// topic_membersにレコードを直接挿入
	now := time.Now()
	for _, topicID := range []model.TopicID{topicID1, topicID2} {
		_, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO topic_members (space_id, topic_id, space_member_id, role, joined_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			string(spaceID), string(topicID), string(spaceMemberID), 0, now, now, now,
		)
		if err != nil {
			t.Fatalf("topic_member作成に失敗: %v", err)
		}
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
