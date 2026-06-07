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

func TestTopicMemberRepository_ListByUserAndTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicMemberRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("tm-byuser-target@example.com").
		WithAtname("tmbyusertarget").
		Build()
	otherUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("tm-byuser-other@example.com").
		WithAtname("tmbyuserother").
		Build()

	// Two spaces the target user belongs to. The user owns a distinct space_member in each.
	// [Ja] 対象ユーザーが参加する 2 スペース。ユーザーはスペースごとに別々の space_member を持つ。
	spaceID1 := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("tm-byuser-space-1").
		WithName("TopicMember ByUser Space 1").
		Build()
	spaceID2 := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("tm-byuser-space-2").
		WithName("TopicMember ByUser Space 2").
		Build()

	spaceMember1ID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID1).
		WithUserID(userID).
		Build()
	spaceMember2ID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID2).
		WithUserID(userID).
		Build()
	// Another user who also belongs to space 1.
	// [Ja] スペース 1 に参加する別ユーザー。
	otherSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID1).
		WithUserID(otherUserID).
		Build()

	topic1ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID1).WithNumber(1).WithName("S1 Topic 1").Build()
	topic2ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID2).WithNumber(1).WithName("S2 Topic 1").Build()
	topic3ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID1).WithNumber(2).WithName("S1 Topic 2").Build()

	// The target user joins topic 1 (space 1) and topic 2 (space 2), but not topic 3.
	// [Ja] 対象ユーザーはトピック 1 (スペース 1) とトピック 2 (スペース 2) に参加、トピック 3 には不参加。
	topicMember1ID := testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID1).WithTopicID(topic1ID).WithSpaceMemberID(spaceMember1ID).Build()
	topicMember2ID := testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID2).WithTopicID(topic2ID).WithSpaceMemberID(spaceMember2ID).Build()
	// The other user joins topic 1: expected to be excluded since we filter by the target user.
	// [Ja] 別ユーザーがトピック 1 に参加 (対象ユーザーで絞るため結果から除外される想定)。
	testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID1).WithTopicID(topic1ID).WithSpaceMemberID(otherSpaceMemberID).Build()

	t.Run("複数スペースにまたがるユーザーのメンバーシップを一括取得する", func(t *testing.T) {
		members, err := repo.ListByUserAndTopics(
			context.Background(),
			userID,
			[]model.SpaceID{spaceID1, spaceID2},
			[]model.TopicID{topic1ID, topic2ID, topic3ID},
		)
		if err != nil {
			t.Fatalf("ListByUserAndTopics() error = %v", err)
		}

		// Only the target user's memberships in topic 1 / topic 2 are returned. Topic 3 (not joined)
		// and the other user's topic 1 membership are excluded.
		//
		// [Ja] 対象ユーザーのトピック 1 / トピック 2 のメンバーシップのみ返る。トピック 3 (未参加) と
		// 別ユーザーのトピック 1 メンバーシップは除外される。
		if len(members) != 2 {
			t.Fatalf("len(members) = %d, want 2", len(members))
		}
		gotIDs := make(map[model.TopicMemberID]bool, len(members))
		for _, m := range members {
			gotIDs[m.ID] = true
		}
		if !gotIDs[topicMember1ID] || !gotIDs[topicMember2ID] {
			t.Errorf("members = %v, want to contain %v and %v", gotIDs, topicMember1ID, topicMember2ID)
		}
	})

	t.Run("指定スペースIDに含まれないトピックは返さない", func(t *testing.T) {
		// topic1 belongs to space 1, but space 1 is excluded from spaceIDs, so it must not be returned
		// even though the topic id is passed (space_id scoping via ANY filters it out).
		//
		// [Ja] topic1 はスペース 1 のトピックだが、spaceIDs からスペース 1 を外しているため、
		// トピック ID を渡しても返らない (space_id スコープの ANY 条件で除外される)。
		members, err := repo.ListByUserAndTopics(
			context.Background(),
			userID,
			[]model.SpaceID{spaceID2},
			[]model.TopicID{topic1ID, topic2ID},
		)
		if err != nil {
			t.Fatalf("ListByUserAndTopics() error = %v", err)
		}
		if len(members) != 1 {
			t.Fatalf("len(members) = %d, want 1", len(members))
		}
		if members[0].ID != topicMember2ID {
			t.Errorf("members[0].ID = %v, want %v", members[0].ID, topicMember2ID)
		}
	})

	t.Run("空のスペースIDリストはnilを返す", func(t *testing.T) {
		members, err := repo.ListByUserAndTopics(context.Background(), userID, []model.SpaceID{}, []model.TopicID{topic1ID})
		if err != nil {
			t.Fatalf("ListByUserAndTopics() error = %v", err)
		}
		if members != nil {
			t.Errorf("ListByUserAndTopics() = %v, want nil", members)
		}
	})

	t.Run("空のトピックIDリストはnilを返す", func(t *testing.T) {
		members, err := repo.ListByUserAndTopics(context.Background(), userID, []model.SpaceID{spaceID1}, []model.TopicID{})
		if err != nil {
			t.Fatalf("ListByUserAndTopics() error = %v", err)
		}
		if members != nil {
			t.Errorf("ListByUserAndTopics() = %v, want nil", members)
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
