package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestTopicRepository_NextTopicNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewTopicRepository(testutil.QueriesWithTx(tx))
	targetSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-next-number-target").
		Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-next-number-other").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(targetSpaceID).
		WithNumber(2).
		WithName("Active").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(targetSpaceID).
		WithNumber(5).
		WithName("Discarded").
		WithDiscarded().
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithNumber(99).
		WithName("Other Space").
		Build()

	number, err := repo.NextTopicNumber(context.Background(), targetSpaceID)
	if err != nil {
		t.Fatalf("NextTopicNumber()のエラー = %v", err)
	}
	if number != 6 {
		t.Errorf("NextTopicNumber() = %d、期待値 = 6", number)
	}
}

func TestTopicRepository_ExistsBySpaceAndName(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewTopicRepository(testutil.QueriesWithTx(tx))
	targetSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-exists-name-target").
		Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-exists-name-other").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(targetSpaceID).
		WithNumber(1).
		WithName("Active").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(targetSpaceID).
		WithNumber(2).
		WithName("Discarded").
		WithDiscarded().
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithNumber(1).
		WithName("Other Space Only").
		Build()

	tests := []struct {
		name string
		want bool
	}{
		{name: "Active", want: true},
		{name: "Discarded", want: true},
		{name: "Other Space Only", want: false},
		{name: "Missing", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := repo.ExistsBySpaceAndName(context.Background(), targetSpaceID, tt.name)
			if err != nil {
				t.Fatalf("ExistsBySpaceAndName()のエラー = %v", err)
			}
			if exists != tt.want {
				t.Errorf("ExistsBySpaceAndName() = %t、期待値 = %t", exists, tt.want)
			}
		})
	}
}

func TestTopicRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewTopicRepository(testutil.QueriesWithTx(tx))
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-create-repository").
		Build()

	topic, err := repo.Create(context.Background(), CreateTopicInput{
		SpaceID:     spaceID,
		Number:      7,
		Name:        "Created Topic",
		Description: "Created description",
		Visibility:  model.TopicVisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if topic.ID == "" {
		t.Error("topic.IDが空")
	}
	if topic.Space == nil || topic.Space.ID != spaceID {
		t.Errorf("topic.Space = %v、期待値 = スペースID %v", topic.Space, spaceID)
	}
	if topic.Number != 7 {
		t.Errorf("topic.Number = %d、期待値 = 7", topic.Number)
	}
	if topic.Name != "Created Topic" {
		t.Errorf("topic.Name = %q、期待値 = %q", topic.Name, "Created Topic")
	}
	if topic.Description != "Created description" {
		t.Errorf("topic.Description = %q、期待値 = %q", topic.Description, "Created description")
	}
	if topic.Visibility != model.TopicVisibilityPrivate {
		t.Errorf("topic.Visibility = %v、期待値 = %v", topic.Visibility, model.TopicVisibilityPrivate)
	}
	if topic.DiscardedAt != nil {
		t.Errorf("topic.DiscardedAt = %v、期待値 = nil", topic.DiscardedAt)
	}
}

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
			t.Fatalf("FindBySpaceAndNumber()のエラー = %v", err)
		}
		if topic == nil {
			t.Fatal("FindBySpaceAndNumber()がnilを返した、期待値 = トピック")
		}
		if topic.ID != topicID {
			t.Errorf("topic.ID = %v、期待値 = %v", topic.ID, topicID)
		}
		if topic.Space.ID != spaceID {
			t.Errorf("topic.Space.ID = %v、期待値 = %v", topic.Space.ID, spaceID)
		}
		if topic.Number != 1 {
			t.Errorf("topic.Number = %v、期待値 = 1", topic.Number)
		}
		if topic.Name != "General" {
			t.Errorf("topic.Name = %v、期待値 = General", topic.Name)
		}
		if topic.Description != "General topic" {
			t.Errorf("topic.Description = %v、期待値 = 'General topic'", topic.Description)
		}
		if topic.Visibility != model.TopicVisibilityPublic {
			t.Errorf("topic.Visibility = %v、期待値 = TopicVisibilityPublic", topic.Visibility)
		}
		if topic.DiscardedAt != nil {
			t.Errorf("topic.DiscardedAt = %v、期待値 = nil", topic.DiscardedAt)
		}
	})

	t.Run("存在しないナンバーはnilを返す", func(t *testing.T) {
		topic, err := repo.FindBySpaceAndNumber(context.Background(), spaceID, 999)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber()のエラー = %v", err)
		}
		if topic != nil {
			t.Errorf("FindBySpaceAndNumber() = %v、期待値 = nil", topic)
		}
	})

	t.Run("存在しないスペースIDはnilを返す", func(t *testing.T) {
		topic, err := repo.FindBySpaceAndNumber(context.Background(), "00000000-0000-0000-0000-000000000000", 1)
		if err != nil {
			t.Fatalf("FindBySpaceAndNumber()のエラー = %v", err)
		}
		if topic != nil {
			t.Errorf("FindBySpaceAndNumber() = %v、期待値 = nil", topic)
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
			t.Fatalf("ListActiveBySpace()のエラー = %v", err)
		}
		if len(topics) != 2 {
			t.Fatalf("len(topics) = %v、期待値 = 2", len(topics))
		}
		if topics[0].Name != "First" {
			t.Errorf("topics[0].Name = %v、期待値 = First", topics[0].Name)
		}
		if topics[1].Name != "Second" {
			t.Errorf("topics[1].Name = %v、期待値 = Second", topics[1].Name)
		}
	})

	t.Run("トピックがないスペースは空のスライスを返す", func(t *testing.T) {
		topics, err := repo.ListActiveBySpace(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("ListActiveBySpace()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
		}
	})
}

func TestTopicRepository_ListPublicBySpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicRepository(q)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("public-topic-space").
		WithName("Public Topic Space").
		Build()

	// ORDER BY numberを検証するため、number順とは異なる順序で公開トピックを作成する。
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Public Second").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public First").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	// 非公開トピックと廃棄済みの公開トピックは、いずれも除外される。
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("Private").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(4).
		WithName("Discarded Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		WithDiscarded().
		Build()

	// 別スペースの公開トピックが対象スペースの結果に混入しないこと (クエリのspace_id
	// スコープが効いていること) を検証する。
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("other-public-topic-space").
		WithName("Other Public Topic Space").
		Build()

	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithNumber(1).
		WithName("Other Space Public").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	t.Run("公開トピックのみをナンバー順で取得できる", func(t *testing.T) {
		topics, err := repo.ListPublicBySpace(context.Background(), spaceID)
		if err != nil {
			t.Fatalf("ListPublicBySpace()のエラー = %v", err)
		}
		// len == 2と下の名前一致により、別スペースの公開トピック ("Other Space
		// Public") がspace_idスコープで除外されることも合わせて確認している。
		if len(topics) != 2 {
			t.Fatalf("len(topics) = %v、期待値 = 2", len(topics))
		}
		if topics[0].Name != "Public First" {
			t.Errorf("topics[0].Name = %v、期待値 = 'Public First'", topics[0].Name)
		}
		if topics[1].Name != "Public Second" {
			t.Errorf("topics[1].Name = %v、期待値 = 'Public Second'", topics[1].Name)
		}
		for _, topic := range topics {
			if topic.Visibility != model.TopicVisibilityPublic {
				t.Errorf("トピック%qのVisibility = %v、期待値 = TopicVisibilityPublic", topic.Name, topic.Visibility)
			}
		}
	})

	t.Run("トピックがないスペースは空のスライスを返す", func(t *testing.T) {
		topics, err := repo.ListPublicBySpace(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("ListPublicBySpace()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
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
			t.Fatalf("FindBySpaceAndNames()のエラー = %v", err)
		}
		if len(topics) != 2 {
			t.Fatalf("len(topics) = %v、期待値 = 2", len(topics))
		}
		names := map[string]bool{}
		for _, topic := range topics {
			names[topic.Name] = true
		}
		if !names["Alpha"] {
			t.Error("Alphaが結果に含まれていない")
		}
		if !names["Gamma"] {
			t.Error("Gammaが結果に含まれていない")
		}
	})

	t.Run("存在しない名前は結果に含まれない", func(t *testing.T) {
		topics, err := repo.FindBySpaceAndNames(context.Background(), spaceID, []string{"NotExist"})
		if err != nil {
			t.Fatalf("FindBySpaceAndNames()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
		}
	})

	t.Run("空のスライスを渡すと空の結果を返す", func(t *testing.T) {
		topics, err := repo.FindBySpaceAndNames(context.Background(), spaceID, []string{})
		if err != nil {
			t.Fatalf("FindBySpaceAndNames()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
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
		Build()

	// スペース2を作成
	space2ID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("joined-user-space2").
		WithName("Space Two").
		Build()

	spaceMember2ID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(space2ID).
		WithUserID(userID).
		Build()

	// スペース1のトピック (last_page_modified_atあり)
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

	// スペース2のトピック (last_page_modified_atが古い)
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

	// スペース1の別トピック (last_page_modified_atなし)
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
			t.Fatalf("ListJoinedByUser()のエラー = %v", err)
		}
		if len(topics) != 3 {
			t.Fatalf("len(topics) = %v、期待値 = 3", len(topics))
		}
		// last_page_modified_atが最新のものが先頭
		if topics[0].Name != "Topic Alpha" {
			t.Errorf("topics[0].Name = %v、期待値 = 'Topic Alpha'", topics[0].Name)
		}
		// 次にlast_page_modified_atが古いもの
		if topics[1].Name != "Topic Beta" {
			t.Errorf("topics[1].Name = %v、期待値 = 'Topic Beta'", topics[1].Name)
		}
		// last_page_modified_atがNULLのものは最後 (NULLS LAST)、number DESCで並ぶ
		if topics[2].Name != "Topic Gamma" {
			t.Errorf("topics[2].Name = %v、期待値 = 'Topic Gamma'", topics[2].Name)
		}
	})

	t.Run("スペース情報が正しく取得できる", func(t *testing.T) {
		topics, err := repo.ListJoinedByUser(context.Background(), userID, 10)
		if err != nil {
			t.Fatalf("ListJoinedByUser()のエラー = %v", err)
		}
		// Topic Alphaはスペース1に所属
		if topics[0].Space.ID != space1ID {
			t.Errorf("topics[0].Space.ID = %v、期待値 = %v", topics[0].Space.ID, space1ID)
		}
		if string(topics[0].Space.Identifier) != "joined-user-space1" {
			t.Errorf("topics[0].Space.Identifier = %v、期待値 = 'joined-user-space1'", topics[0].Space.Identifier)
		}
		if topics[0].Space.Name != "Space One" {
			t.Errorf("topics[0].Space.Name = %v、期待値 = 'Space One'", topics[0].Space.Name)
		}
		// Topic Betaはスペース2に所属
		if topics[1].Space.ID != space2ID {
			t.Errorf("topics[1].Space.ID = %v、期待値 = %v", topics[1].Space.ID, space2ID)
		}
		if string(topics[1].Space.Identifier) != "joined-user-space2" {
			t.Errorf("topics[1].Space.Identifier = %v、期待値 = 'joined-user-space2'", topics[1].Space.Identifier)
		}
	})

	t.Run("LIMITが正しく適用される", func(t *testing.T) {
		topics, err := repo.ListJoinedByUser(context.Background(), userID, 2)
		if err != nil {
			t.Fatalf("ListJoinedByUser()のエラー = %v", err)
		}
		if len(topics) != 2 {
			t.Errorf("len(topics) = %v、期待値 = 2", len(topics))
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
			t.Fatalf("ListJoinedByUser()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
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
			t.Fatalf("ListJoinedByUser()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
		}
	})

	t.Run("削除済みスペースのトピックは除外される", func(t *testing.T) {
		discardedSpaceUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("joined-discarded-space@example.com").
			WithAtname("joineddiscardedspace").
			Build()

		discardedSpaceOnlyID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("joined-discarded-only-space").
			WithName("Joined Discarded Only Space").
			WithDiscarded().
			Build()

		discardedSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(discardedSpaceOnlyID).
			WithUserID(discardedSpaceUserID).
			Build()

		activeTopicInDiscardedSpaceID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(discardedSpaceOnlyID).
			WithNumber(1).
			WithName("Active Topic In Discarded Space").
			Build()

		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(discardedSpaceOnlyID).
			WithTopicID(activeTopicInDiscardedSpaceID).
			WithSpaceMemberID(discardedSpaceMemberID).
			Build()

		topics, err := repo.ListJoinedByUser(context.Background(), discardedSpaceUserID, 10)
		if err != nil {
			t.Fatalf("ListJoinedByUser()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
		}
	})

	t.Run("トピックに参加していないユーザーは空のスライスを返す", func(t *testing.T) {
		topics, err := repo.ListJoinedByUser(context.Background(), "00000000-0000-0000-0000-000000000000", 10)
		if err != nil {
			t.Fatalf("ListJoinedByUser()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
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
			t.Fatalf("ListJoinedBySpaceMember()のエラー = %v", err)
		}
		if len(topics) != 2 {
			t.Fatalf("len(topics) = %v、期待値 = 2", len(topics))
		}
		if topics[0].Name != "Joined Topic 1" {
			t.Errorf("topics[0].Name = %v、期待値 = 'Joined Topic 1'", topics[0].Name)
		}
		if topics[1].Name != "Joined Topic 2" {
			t.Errorf("topics[1].Name = %v、期待値 = 'Joined Topic 2'", topics[1].Name)
		}
	})

	t.Run("トピックに参加していないスペースメンバーは空のスライスを返す", func(t *testing.T) {
		topics, err := repo.ListJoinedBySpaceMember(context.Background(), "00000000-0000-0000-0000-000000000000", spaceID)
		if err != nil {
			t.Fatalf("ListJoinedBySpaceMember()のエラー = %v", err)
		}
		if len(topics) != 0 {
			t.Errorf("len(topics) = %v、期待値 = 0", len(topics))
		}
	})
}

func TestTopicRepository_FindFirstJoinedBySpaceMember(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewTopicRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("first-joined-topic@example.com").
		WithAtname("firstjoinedtopic").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("first-joined-space").
		WithName("First Joined Space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	// 参加トピックを2件作成し、id昇順で最初のものが返ることを検証する。
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

	// 参加していないトピック (結果に含まれないことを確認する)。
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(3).
		WithName("Not Joined Topic").
		Build()

	for _, topicID := range []model.TopicID{topicID1, topicID2} {
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(spaceMemberID).
			Build()
	}

	t.Run("参加トピックのうちid昇順で最初のものを返す", func(t *testing.T) {
		topic, err := repo.FindFirstJoinedBySpaceMember(context.Background(), spaceMemberID, spaceID)
		if err != nil {
			t.Fatalf("FindFirstJoinedBySpaceMember()のエラー = %v", err)
		}
		if topic == nil {
			t.Fatal("FindFirstJoinedBySpaceMember()がnilを返した、期待値 = トピック")
		}
		// ULID (uuid) は文字列比較がDBのid ASCと一致するため、生成順ではなく
		// idの小さいほうを期待値として算出する。
		wantID := topicID1
		if string(topicID2) < string(topicID1) {
			wantID = topicID2
		}
		if topic.ID != wantID {
			t.Errorf("topic.ID = %v、期待値 = %v (参加中でIDが最小のトピック)", topic.ID, wantID)
		}
		if topic.Space.ID != spaceID {
			t.Errorf("topic.Space.ID = %v、期待値 = %v", topic.Space.ID, spaceID)
		}
	})

	t.Run("トピックに参加していないスペースメンバーはnilを返す", func(t *testing.T) {
		topic, err := repo.FindFirstJoinedBySpaceMember(context.Background(), "00000000-0000-0000-0000-000000000000", spaceID)
		if err != nil {
			t.Fatalf("FindFirstJoinedBySpaceMember()のエラー = %v", err)
		}
		if topic != nil {
			t.Errorf("FindFirstJoinedBySpaceMember() = %v、期待値 = nil", topic)
		}
	})

	t.Run("削除済みトピックは除外される", func(t *testing.T) {
		discardedUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("first-joined-discarded@example.com").
			WithAtname("firstjoineddiscarded").
			Build()

		discardedSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("first-joined-discarded-space").
			WithName("First Joined Discarded Space").
			Build()

		discardedSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(discardedSpaceID).
			WithUserID(discardedUserID).
			Build()

		// このメンバーは削除済みトピックにのみ参加しているため、結果はnilになる。
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

		topic, err := repo.FindFirstJoinedBySpaceMember(context.Background(), discardedSpaceMemberID, discardedSpaceID)
		if err != nil {
			t.Fatalf("FindFirstJoinedBySpaceMember()のエラー = %v", err)
		}
		if topic != nil {
			t.Errorf("FindFirstJoinedBySpaceMember() = %v、期待値 = nil (削除済みトピックは除外される)", topic)
		}
	})

	t.Run("idが最小でも参加していないトピックは除外される", func(t *testing.T) {
		// 生成順に関わらずidの小さいトピックを未参加にできるよう、独立したスペースを使う。
		excludeUserID := testutil.NewUserBuilder(t, tx).
			WithEmail("first-joined-exclude@example.com").
			WithAtname("firstjoinedexclude").
			Build()

		excludeSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("first-joined-exclude-space").
			WithName("First Joined Exclude Space").
			Build()

		excludeSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(excludeSpaceID).
			WithUserID(excludeUserID).
			Build()

		topicA := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(excludeSpaceID).
			WithNumber(1).
			WithName("Exclude Topic A").
			Build()

		topicB := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(excludeSpaceID).
			WithNumber(2).
			WithName("Exclude Topic B").
			Build()

		// 文字列比較 (DBのid ASCと一致) でidの大きいほうを判定し、生成順に依存しないようにする。
		largerID := topicA
		if string(topicB) > string(topicA) {
			largerID = topicB
		}

		// idが大きいトピックにのみ参加し、idが小さいトピックは未参加のままにする。
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(excludeSpaceID).
			WithTopicID(largerID).
			WithSpaceMemberID(excludeSpaceMemberID).
			Build()

		topic, err := repo.FindFirstJoinedBySpaceMember(context.Background(), excludeSpaceMemberID, excludeSpaceID)
		if err != nil {
			t.Fatalf("FindFirstJoinedBySpaceMember()のエラー = %v", err)
		}
		if topic == nil {
			t.Fatal("FindFirstJoinedBySpaceMember()がnilを返した、期待値 = 参加中のトピック")
		}
		// 未参加でidが小さいトピックではなく、参加済みでidが大きいトピックが返る。
		if topic.ID != largerID {
			t.Errorf("topic.ID = %v、期待値 = %v (未参加のトピックのほうがIDが小さくても参加中のトピック)", topic.ID, largerID)
		}
	})
}

// TestTopicRepository_ExistsBySpaceAndNameExcludingIDは一般設定が行う一意性チェックを扱う。
// 更新するトピック自身は除かれ、それ以外のトピックは削除済みも含めて数えられる。
func TestTopicRepository_ExistsBySpaceAndNameExcludingID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewTopicRepository(testutil.QueriesWithTx(tx))
	targetSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-exists-excluding-target").
		Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-exists-excluding-other").
		Build()

	selfID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(targetSpaceID).
		WithNumber(1).
		WithName("Self").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(targetSpaceID).
		WithNumber(2).
		WithName("Sibling").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(targetSpaceID).
		WithNumber(3).
		WithName("Discarded").
		WithDiscarded().
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(otherSpaceID).
		WithNumber(1).
		WithName("Other Space Only").
		Build()

	tests := []struct {
		name      string
		topicName string
		want      bool
	}{
		{name: "更新するトピック自身の名前は数えない", topicName: "Self", want: false},
		{name: "同じスペースの別のトピックの名前は数える", topicName: "Sibling", want: true},
		{name: "削除済みトピックの名前も数える", topicName: "Discarded", want: true},
		{name: "別のスペースのトピックの名前は数えない", topicName: "Other Space Only", want: false},
		{name: "使われていない名前は数えない", topicName: "Unused", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := repo.ExistsBySpaceAndNameExcludingID(context.Background(), targetSpaceID, tt.topicName, selfID)
			if err != nil {
				t.Fatalf("ExistsBySpaceAndNameExcludingID()のエラー = %v", err)
			}
			if exists != tt.want {
				t.Errorf("ExistsBySpaceAndNameExcludingID(%q) = %v、期待値 = %v", tt.topicName, exists, tt.want)
			}
		})
	}
}

// TestTopicRepository_Updateは一般設定が保存する更新を扱う。同じidを渡されても別の
// スペースのトピックには届かないことも含める。
func TestTopicRepository_Update(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewTopicRepository(testutil.QueriesWithTx(tx))
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-update-target").
		Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("topic-update-other").
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Before").
		WithDescription("before").
		Build()

	topic, err := repo.Update(context.Background(), UpdateTopicInput{
		ID:          topicID,
		SpaceID:     spaceID,
		Name:        "After",
		Description: "after",
		Visibility:  model.TopicVisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("Update()のエラー = %v", err)
	}
	if topic.Name != "After" {
		t.Errorf("Name = %q、期待値 = %q", topic.Name, "After")
	}
	if topic.Description != "after" {
		t.Errorf("Description = %q、期待値 = %q", topic.Description, "after")
	}
	if topic.Visibility != model.TopicVisibilityPrivate {
		t.Errorf("Visibility = %v、期待値 = %v", topic.Visibility, model.TopicVisibilityPrivate)
	}
	if topic.Number != 1 {
		t.Errorf("Number = %d、期待値 = 1", topic.Number)
	}

	if _, err := repo.Update(context.Background(), UpdateTopicInput{
		ID:          topicID,
		SpaceID:     otherSpaceID,
		Name:        "Wrong Space",
		Description: "",
		Visibility:  model.TopicVisibilityPublic,
	}); err == nil {
		t.Error("別スペースのIDを指定したUpdate()でエラーが返らなかった、期待値 = どの行も更新されない")
	}

	stored, err := repo.FindBySpaceAndID(context.Background(), spaceID, topicID)
	if err != nil {
		t.Fatalf("FindBySpaceAndID()のエラー = %v", err)
	}
	if stored.Name != "After" {
		t.Errorf("Name = %q、期待値 = %q", stored.Name, "After")
	}
}
