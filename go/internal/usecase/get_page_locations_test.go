package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPageLocationsUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPageLocationsUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	memberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpl-member@example.com").
		WithAtname("gplmember").
		Build()
	inactiveUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpl-inactive@example.com").
		WithAtname("gplinactive").
		Build()
	nonMemberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpl-non-member@example.com").
		WithAtname("gplnonmember").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpl-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberUserID).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(inactiveUserID).
		WithActive(false).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Wiki入門").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("別の話題").
		Build()

	t.Run("正常系: メンバーはキーワードに一致するページを取得できる", func(t *testing.T) {
		output, err := uc.Execute(context.Background(), GetPageLocationsInput{
			SpaceIdentifier: "gpl-space",
			UserID:          memberUserID,
			Query:           "wiki",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("output = nil、非nilを期待")
		}

		want := []repository.PageLocation{{TopicName: "General", PageTitle: "Wiki入門"}}
		if len(output.Locations) != len(want) {
			t.Fatalf("Locations = %v、期待値 = %v", output.Locations, want)
		}
		if output.Locations[0] != want[0] {
			t.Errorf("Locations[0] = %v、期待値 = %v", output.Locations[0], want[0])
		}
	})

	tests := []struct {
		name            string
		spaceIdentifier model.SpaceIdentifier
		userID          model.UserID
	}{
		{
			name:            "スペースが存在しないときはnilを返す",
			spaceIdentifier: "gpl-missing-space",
			userID:          memberUserID,
		},
		{
			name:            "スペースのメンバーでないときはnilを返す",
			spaceIdentifier: "gpl-space",
			userID:          nonMemberUserID,
		},
		{
			name:            "無効化されたメンバーのときはnilを返す",
			spaceIdentifier: "gpl-space",
			userID:          inactiveUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetPageLocationsInput{
				SpaceIdentifier: tt.spaceIdentifier,
				UserID:          tt.userID,
				Query:           "wiki",
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output != nil {
				t.Errorf("output = %v、nilを期待", output)
			}
		})
	}
}

func TestGetPageLocationsUsecase_Execute_TopicVisibility(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPageLocationsUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	restrictedUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpl-vis-restricted@example.com").
		WithAtname("gplvisrestricted").
		Build()
	topicReaderUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpl-vis-topic-reader@example.com").
		WithAtname("gplvistopicreader").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpl-vis-space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(restrictedUserID).
		WithScopes([]model.Scope{}).
		Build()
	topicReaderSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(topicReaderUserID).
		WithScopes([]model.Scope{}).
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
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithSpaceMemberID(topicReaderSpaceMemberID).
		WithScopes([]model.Scope{model.ScopeTopicRead}).
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("公開トピックのWiki").
		WithModifiedAt(baseTime).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("非公開トピックのWiki").
		WithModifiedAt(baseTime.Add(time.Hour)).
		Build()

	tests := []struct {
		name   string
		userID model.UserID
		want   []repository.PageLocation
	}{
		{
			name:   "topic:readを持たないメンバーには公開トピックのページだけを返す",
			userID: restrictedUserID,
			want: []repository.PageLocation{
				{TopicName: "Public", PageTitle: "公開トピックのWiki"},
			},
		},
		{
			name:   "トピックメンバーとしてtopic:readを持つメンバーには非公開トピックのページも返す",
			userID: topicReaderUserID,
			want: []repository.PageLocation{
				{TopicName: "Private", PageTitle: "非公開トピックのWiki"},
				{TopicName: "Public", PageTitle: "公開トピックのWiki"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetPageLocationsInput{
				SpaceIdentifier: "gpl-vis-space",
				UserID:          tt.userID,
				Query:           "wiki",
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output == nil {
				t.Fatal("output = nil、非nilを期待")
			}
			if len(output.Locations) != len(tt.want) {
				t.Fatalf("Locations = %v、期待値 = %v", output.Locations, tt.want)
			}
			for i, want := range tt.want {
				if output.Locations[i] != want {
					t.Errorf("Locations[%d] = %v、期待値 = %v", i, output.Locations[i], want)
				}
			}
		})
	}
}

func TestGetPageLocationsUsecase_Execute_LinkedFromDrafts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPageLocationsUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	aliceUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpl-draft-alice@example.com").
		WithAtname("gpldraftalice").
		Build()
	bobUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpl-draft-bob@example.com").
		WithAtname("gpldraftbob").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpl-draft-space").
		Build()
	aliceSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(aliceUserID).
		Build()
	bobSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(bobUserID).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sourcePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("リンク元").
		WithModifiedAt(baseTime).
		Build()
	aliceTargetID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Aliceの下書きのWiki").
		WithUnpublished().
		WithModifiedAt(baseTime.Add(time.Hour)).
		Build()
	bobTargetID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Bobの下書きのWiki").
		WithUnpublished().
		WithModifiedAt(baseTime.Add(2 * time.Hour)).
		Build()

	// 同じ公開ページを、それぞれが自分の下書きで編集している
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(sourcePageID).
		WithTopicID(topicID).
		WithSpaceMemberID(aliceSpaceMemberID).
		WithLinkedPageIDs([]model.PageID{aliceTargetID}).
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(sourcePageID).
		WithTopicID(topicID).
		WithSpaceMemberID(bobSpaceMemberID).
		WithLinkedPageIDs([]model.PageID{bobTargetID}).
		Build()

	tests := []struct {
		name   string
		userID model.UserID
		want   []repository.PageLocation
	}{
		{
			name:   "自分の下書きからリンクされている未公開ページだけを返す (Alice)",
			userID: aliceUserID,
			want: []repository.PageLocation{
				{TopicName: "General", PageTitle: "Aliceの下書きのWiki"},
			},
		},
		{
			name:   "自分の下書きからリンクされている未公開ページだけを返す (Bob)",
			userID: bobUserID,
			want: []repository.PageLocation{
				{TopicName: "General", PageTitle: "Bobの下書きのWiki"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetPageLocationsInput{
				SpaceIdentifier: "gpl-draft-space",
				UserID:          tt.userID,
				Query:           "wiki",
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output == nil {
				t.Fatal("output = nil、非nilを期待")
			}
			if len(output.Locations) != len(tt.want) {
				t.Fatalf("Locations = %v、期待値 = %v", output.Locations, tt.want)
			}
			for i, want := range tt.want {
				if output.Locations[i] != want {
					t.Errorf("Locations[%d] = %v、期待値 = %v", i, output.Locations[i], want)
				}
			}
		})
	}
}
