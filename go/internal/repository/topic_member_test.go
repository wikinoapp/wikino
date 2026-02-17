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
		WithRole(0). // owner
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
		WithRole(0). // admin
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
		if member.Role != model.TopicMemberRoleAdmin {
			t.Errorf("member.Role = %v, want TopicMemberRoleAdmin", member.Role)
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
		WithRole(0).
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
		WithRole(0).
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
