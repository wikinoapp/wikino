package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestTopicMemberRepository_FindBySpaceMemberAndTopic(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicMemberRepository(q)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("topicmember@example.com").
		WithAtname("topicmember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topicmember-test-space").
		WithName("TopicMember Test Space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithActive(true).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	topicMemberID := testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	t.Run("トピックメンバーを取得できる", func(t *testing.T) {
		member, err := repo.FindBySpaceMemberAndTopic(context.Background(), spaceID, spaceMemberID, topicID)
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic() error = %v", err)
		}
		if member == nil {
			t.Fatal("FindBySpaceMemberAndTopic() returned nil, want member")
		}
		if member.ID != topicMemberID {
			t.Errorf("member.ID = %v, want %v", member.ID, topicMemberID)
		}
		if member.SpaceID != spaceID {
			t.Errorf("member.SpaceID = %v, want %v", member.SpaceID, spaceID)
		}
		if member.TopicID != topicID {
			t.Errorf("member.TopicID = %v, want %v", member.TopicID, topicID)
		}
		if member.SpaceMemberID != spaceMemberID {
			t.Errorf("member.SpaceMemberID = %v, want %v", member.SpaceMemberID, spaceMemberID)
		}
		if len(member.Scopes) != 0 {
			t.Errorf("member.Scopes = %v, want empty", member.Scopes)
		}
		if member.LastPageModifiedAt != nil {
			t.Errorf("member.LastPageModifiedAt = %v, want nil", member.LastPageModifiedAt)
		}
	})

	t.Run("存在しないスペースメンバーIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindBySpaceMemberAndTopic(context.Background(), spaceID, "00000000-0000-0000-0000-000000000000", topicID)
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic() error = %v", err)
		}
		if member != nil {
			t.Errorf("FindBySpaceMemberAndTopic() = %v, want nil", member)
		}
	})

	t.Run("存在しないトピックIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindBySpaceMemberAndTopic(context.Background(), spaceID, spaceMemberID, "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic() error = %v", err)
		}
		if member != nil {
			t.Errorf("FindBySpaceMemberAndTopic() = %v, want nil", member)
		}
	})
}

func TestTopicMemberRepository_ListBySpaceMemberAndTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicMemberRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("tm-list-space").
		WithName("TopicMember List Space").
		Build()

	memberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("tm-list-member@example.com").
		WithAtname("tmlistmember").
		Build()
	otherUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("tm-list-other@example.com").
		WithAtname("tmlistother").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberUserID).
		Build()
	otherSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(otherUserID).
		Build()

	topic1ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("Topic 1").Build()
	topic2ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(2).WithName("Topic 2").Build()
	topic3ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(3).WithName("Topic 3").Build()

	// The member joins topics 1 and 2 (not 3). The other member joins topic 1.
	// [Ja] メンバーはトピック 1 と 2 に参加 (3 には不参加)。別メンバーはトピック 1 に参加。
	testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topic1ID).WithSpaceMemberID(spaceMemberID).Build()
	testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topic2ID).WithSpaceMemberID(spaceMemberID).Build()
	testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topic1ID).WithSpaceMemberID(otherSpaceMemberID).Build()

	t.Run("参加トピックのメンバーシップのみ一括取得する", func(t *testing.T) {
		members, err := repo.ListBySpaceMemberAndTopics(context.Background(), spaceID, spaceMemberID, []model.TopicID{topic1ID, topic2ID, topic3ID})
		if err != nil {
			t.Fatalf("ListBySpaceMemberAndTopics() error = %v", err)
		}
		// Only topics 1 and 2 are returned; topic 3 has no membership for this member.
		// [Ja] トピック 1 と 2 のみ返る。トピック 3 はこのメンバーのメンバーシップが無い。
		if len(members) != 2 {
			t.Fatalf("len(members) = %d, want 2", len(members))
		}
		gotTopicIDs := make(map[model.TopicID]bool, len(members))
		for _, m := range members {
			if m.SpaceMemberID != spaceMemberID {
				t.Errorf("member.SpaceMemberID = %v, want %v (must not return another member's row)", m.SpaceMemberID, spaceMemberID)
			}
			gotTopicIDs[m.TopicID] = true
		}
		if !gotTopicIDs[topic1ID] || !gotTopicIDs[topic2ID] {
			t.Errorf("returned topic ids = %v, want topics 1 and 2", gotTopicIDs)
		}
		if gotTopicIDs[topic3ID] {
			t.Error("topic 3 should not be returned (member has not joined it)")
		}
	})

	t.Run("空のトピックIDリストはnilを返す", func(t *testing.T) {
		members, err := repo.ListBySpaceMemberAndTopics(context.Background(), spaceID, spaceMemberID, []model.TopicID{})
		if err != nil {
			t.Fatalf("ListBySpaceMemberAndTopics() error = %v", err)
		}
		if members != nil {
			t.Errorf("ListBySpaceMemberAndTopics() = %v, want nil", members)
		}
	})
}

func TestTopicMemberRepository_UpdateLastPageModifiedAt(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicMemberRepository(q)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("topicmember-update@example.com").
		WithAtname("topicmemberupdate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topicmember-update-space").
		WithName("TopicMember Update Space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithActive(true).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	t.Run("last_page_modified_atを更新できる", func(t *testing.T) {
		modifiedAt := time.Now().Truncate(time.Microsecond)
		err := repo.UpdateLastPageModifiedAt(context.Background(), spaceID, topicID, spaceMemberID, modifiedAt)
		if err != nil {
			t.Fatalf("UpdateLastPageModifiedAt() error = %v", err)
		}

		// 更新後の値を確認
		member, err := repo.FindBySpaceMemberAndTopic(context.Background(), spaceID, spaceMemberID, topicID)
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic() error = %v", err)
		}
		if member == nil {
			t.Fatal("FindBySpaceMemberAndTopic() returned nil")
		}
		if member.LastPageModifiedAt == nil {
			t.Fatal("member.LastPageModifiedAt is nil, want non-nil")
		}
		if !member.LastPageModifiedAt.Truncate(time.Microsecond).Equal(modifiedAt) {
			t.Errorf("member.LastPageModifiedAt = %v, want %v", member.LastPageModifiedAt, modifiedAt)
		}
	})
}
