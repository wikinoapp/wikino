package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestTopicMemberRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewTopicMemberRepository(testutil.QueriesWithTx(tx))
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("topic-member-create@example.com").
		WithAtname("topicmembercreate").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-member-create").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithName("Created Topic Member").
		Build()

	beforeCreate := time.Now().Truncate(time.Microsecond)
	member, err := repo.Create(context.Background(), CreateTopicMemberInput{
		SpaceID:       spaceID,
		TopicID:       topicID,
		SpaceMemberID: spaceMemberID,
	})
	afterCreate := time.Now().Truncate(time.Microsecond)
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if member.ID == "" {
		t.Error("member.IDが空")
	}
	if member.SpaceID != spaceID {
		t.Errorf("member.SpaceID = %v、期待値 = %v", member.SpaceID, spaceID)
	}
	if member.TopicID != topicID {
		t.Errorf("member.TopicID = %v、期待値 = %v", member.TopicID, topicID)
	}
	if member.SpaceMemberID != spaceMemberID {
		t.Errorf("member.SpaceMemberID = %v、期待値 = %v", member.SpaceMemberID, spaceMemberID)
	}
	if len(member.Scopes) != 0 {
		t.Errorf("member.Scopes = %v、期待値 = 空", member.Scopes)
	}
	if member.JoinedAt.Before(beforeCreate) || member.JoinedAt.After(afterCreate) {
		t.Errorf("member.JoinedAt = %v、期待値 = %vから%vの間", member.JoinedAt, beforeCreate, afterCreate)
	}
	if member.LastPageModifiedAt != nil {
		t.Errorf("member.LastPageModifiedAt = %v、期待値 = nil", member.LastPageModifiedAt)
	}
}

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
			t.Fatalf("FindBySpaceMemberAndTopic()のエラー = %v", err)
		}
		if member == nil {
			t.Fatal("FindBySpaceMemberAndTopic()がnilを返した、期待値 = メンバー")
		}
		if member.ID != topicMemberID {
			t.Errorf("member.ID = %v、期待値 = %v", member.ID, topicMemberID)
		}
		if member.SpaceID != spaceID {
			t.Errorf("member.SpaceID = %v、期待値 = %v", member.SpaceID, spaceID)
		}
		if member.TopicID != topicID {
			t.Errorf("member.TopicID = %v、期待値 = %v", member.TopicID, topicID)
		}
		if member.SpaceMemberID != spaceMemberID {
			t.Errorf("member.SpaceMemberID = %v、期待値 = %v", member.SpaceMemberID, spaceMemberID)
		}
		if len(member.Scopes) != 0 {
			t.Errorf("member.Scopes = %v、期待値 = 空", member.Scopes)
		}
		if member.LastPageModifiedAt != nil {
			t.Errorf("member.LastPageModifiedAt = %v、期待値 = nil", member.LastPageModifiedAt)
		}
	})

	t.Run("存在しないスペースメンバーIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindBySpaceMemberAndTopic(context.Background(), spaceID, "00000000-0000-0000-0000-000000000000", topicID)
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic()のエラー = %v", err)
		}
		if member != nil {
			t.Errorf("FindBySpaceMemberAndTopic() = %v、期待値 = nil", member)
		}
	})

	t.Run("存在しないトピックIDはnilを返す", func(t *testing.T) {
		member, err := repo.FindBySpaceMemberAndTopic(context.Background(), spaceID, spaceMemberID, "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic()のエラー = %v", err)
		}
		if member != nil {
			t.Errorf("FindBySpaceMemberAndTopic() = %v、期待値 = nil", member)
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

	// メンバーはトピック1と2に参加 (3には不参加)。別メンバーはトピック1に参加。
	testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topic1ID).WithSpaceMemberID(spaceMemberID).Build()
	testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topic2ID).WithSpaceMemberID(spaceMemberID).Build()
	testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topic1ID).WithSpaceMemberID(otherSpaceMemberID).Build()

	t.Run("参加トピックのメンバーシップのみ一括取得する", func(t *testing.T) {
		members, err := repo.ListBySpaceMemberAndTopics(context.Background(), spaceID, spaceMemberID, []model.TopicID{topic1ID, topic2ID, topic3ID})
		if err != nil {
			t.Fatalf("ListBySpaceMemberAndTopics()のエラー = %v", err)
		}
		// トピック1と2のみ返る。トピック3はこのメンバーのメンバーシップが無い。
		if len(members) != 2 {
			t.Fatalf("len(members) = %d、期待値 = 2", len(members))
		}
		gotTopicIDs := make(map[model.TopicID]bool, len(members))
		for _, m := range members {
			if m.SpaceMemberID != spaceMemberID {
				t.Errorf("member.SpaceMemberID = %v、期待値 = %v (別のメンバーの行が返されている)", m.SpaceMemberID, spaceMemberID)
			}
			gotTopicIDs[m.TopicID] = true
		}
		if !gotTopicIDs[topic1ID] || !gotTopicIDs[topic2ID] {
			t.Errorf("返されたトピックID = %v、期待値 = トピック1と2", gotTopicIDs)
		}
		if gotTopicIDs[topic3ID] {
			t.Error("トピック3が返されている (メンバーは参加していない)")
		}
	})

	t.Run("空のトピックIDリストはnilを返す", func(t *testing.T) {
		members, err := repo.ListBySpaceMemberAndTopics(context.Background(), spaceID, spaceMemberID, []model.TopicID{})
		if err != nil {
			t.Fatalf("ListBySpaceMemberAndTopics()のエラー = %v", err)
		}
		if members != nil {
			t.Errorf("ListBySpaceMemberAndTopics() = %v、期待値 = nil", members)
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

	// 対象ユーザーが参加する2スペース。ユーザーはスペースごとに別々のspace_memberを持つ。
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
	// スペース1に参加する別ユーザー。
	otherSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID1).
		WithUserID(otherUserID).
		Build()

	topic1ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID1).WithNumber(1).WithName("S1 Topic 1").Build()
	topic2ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID2).WithNumber(1).WithName("S2 Topic 1").Build()
	topic3ID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID1).WithNumber(2).WithName("S1 Topic 2").Build()

	// 対象ユーザーはトピック1 (スペース1) とトピック2 (スペース2) に参加、トピック3には不参加。
	topicMember1ID := testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID1).WithTopicID(topic1ID).WithSpaceMemberID(spaceMember1ID).Build()
	topicMember2ID := testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID2).WithTopicID(topic2ID).WithSpaceMemberID(spaceMember2ID).Build()
	// 別ユーザーがトピック1に参加 (対象ユーザーで絞るため結果から除外される想定)。
	testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(spaceID1).WithTopicID(topic1ID).WithSpaceMemberID(otherSpaceMemberID).Build()

	t.Run("複数スペースにまたがるユーザーのメンバーシップを一括取得する", func(t *testing.T) {
		members, err := repo.ListByUserAndTopics(
			context.Background(),
			userID,
			[]model.SpaceID{spaceID1, spaceID2},
			[]model.TopicID{topic1ID, topic2ID, topic3ID},
		)
		if err != nil {
			t.Fatalf("ListByUserAndTopics()のエラー = %v", err)
		}

		// 対象ユーザーのトピック1 / トピック2のメンバーシップのみ返る。トピック3 (未参加) と
		// 別ユーザーのトピック1メンバーシップは除外される。
		if len(members) != 2 {
			t.Fatalf("len(members) = %d、期待値 = 2", len(members))
		}
		gotIDs := make(map[model.TopicMemberID]bool, len(members))
		for _, m := range members {
			gotIDs[m.ID] = true
		}
		if !gotIDs[topicMember1ID] || !gotIDs[topicMember2ID] {
			t.Errorf("members = %v、期待値 = %vと%vを含む", gotIDs, topicMember1ID, topicMember2ID)
		}
	})

	t.Run("指定スペースIDに含まれないトピックは返さない", func(t *testing.T) {
		// topic1はスペース1のトピックだが、spaceIDsからスペース1を外しているため、
		// トピックIDを渡しても返らない (space_idスコープのANY条件で除外される)。
		members, err := repo.ListByUserAndTopics(
			context.Background(),
			userID,
			[]model.SpaceID{spaceID2},
			[]model.TopicID{topic1ID, topic2ID},
		)
		if err != nil {
			t.Fatalf("ListByUserAndTopics()のエラー = %v", err)
		}
		if len(members) != 1 {
			t.Fatalf("len(members) = %d、期待値 = 1", len(members))
		}
		if members[0].ID != topicMember2ID {
			t.Errorf("members[0].ID = %v、期待値 = %v", members[0].ID, topicMember2ID)
		}
	})

	t.Run("空のスペースIDリストはnilを返す", func(t *testing.T) {
		members, err := repo.ListByUserAndTopics(context.Background(), userID, []model.SpaceID{}, []model.TopicID{topic1ID})
		if err != nil {
			t.Fatalf("ListByUserAndTopics()のエラー = %v", err)
		}
		if members != nil {
			t.Errorf("ListByUserAndTopics() = %v、期待値 = nil", members)
		}
	})

	t.Run("空のトピックIDリストはnilを返す", func(t *testing.T) {
		members, err := repo.ListByUserAndTopics(context.Background(), userID, []model.SpaceID{spaceID1}, []model.TopicID{})
		if err != nil {
			t.Fatalf("ListByUserAndTopics()のエラー = %v", err)
		}
		if members != nil {
			t.Errorf("ListByUserAndTopics() = %v、期待値 = nil", members)
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
			t.Fatalf("UpdateLastPageModifiedAt()のエラー = %v", err)
		}

		// 更新後の値を確認
		member, err := repo.FindBySpaceMemberAndTopic(context.Background(), spaceID, spaceMemberID, topicID)
		if err != nil {
			t.Fatalf("FindBySpaceMemberAndTopic()のエラー = %v", err)
		}
		if member == nil {
			t.Fatal("FindBySpaceMemberAndTopic()がnilを返した")
		}
		if member.LastPageModifiedAt == nil {
			t.Fatal("member.LastPageModifiedAtがnil")
		}
		if !member.LastPageModifiedAt.Truncate(time.Microsecond).Equal(modifiedAt) {
			t.Errorf("member.LastPageModifiedAt = %v、期待値 = %v", member.LastPageModifiedAt, modifiedAt)
		}
	})
}
