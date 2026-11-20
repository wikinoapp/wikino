package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// trashShowFixtureはゴミ箱の一覧のテストで使うスペースとメンバーを持つ。
type trashShowFixture struct {
	tx            *sql.Tx
	queries       *query.Queries
	identifier    model.SpaceIdentifier
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
	userID        model.UserID
}

// setupTrashShowFixtureは、指定したロールのメンバーがいるスペースを作る。
func setupTrashShowFixture(t *testing.T, suffix string, spaceRole model.SpaceRole) trashShowFixture {
	t.Helper()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("trash-show-" + suffix + "@example.com").
		WithAtname("trash_show_" + suffix).
		Build()

	identifier := "trash-show-" + suffix
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(spaceRole).
		Build()

	return trashShowFixture{
		tx:            tx,
		queries:       queries,
		identifier:    model.SpaceIdentifier(identifier),
		spaceID:       spaceID,
		spaceMemberID: spaceMemberID,
		userID:        userID,
	}
}

// addTopicはフィクスチャのスペースにトピックを作る。topicRoleを渡すと、メンバーをそのロールで
// トピックに参加させる。
func (f trashShowFixture) addTopic(t *testing.T, number int32, visibility model.TopicVisibility, topicRole model.TopicRole) model.TopicID {
	t.Helper()

	tx := f.tx
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithNumber(number).
		WithName("トピック" + string(rune('A'+number-1))).
		WithVisibility(int32(visibility)).
		Build()

	if topicRole != "" {
		testutil.NewTopicMemberBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(f.spaceMemberID).
			WithRole(topicRole).
			Build()
	}
	return topicID
}

// addTrashedPageはトピックに、指定した日時にゴミ箱へ入れたページを作る。
func (f trashShowFixture) addTrashedPage(t *testing.T, topicID model.TopicID, number model.PageNumber, title string, trashedAt time.Time) model.PageID {
	t.Helper()

	return testutil.NewPageBuilder(t, f.tx).
		WithSpaceID(f.spaceID).
		WithTopicID(topicID).
		WithNumber(number).
		WithTitle(title).
		WithTrashedAt(trashedAt).
		Build()
}

func (f trashShowFixture) usecase() *GetTrashShowUsecase {
	return NewGetTrashShowUsecase(
		repository.NewSpaceRepository(f.queries),
		repository.NewSpaceMemberRepository(f.queries),
		repository.NewPageRepository(f.queries),
		repository.NewTopicRepository(f.queries),
		repository.NewTopicMemberRepository(f.queries),
	)
}

func pageTitles(pages []*model.Page) []string {
	titles := make([]string, len(pages))
	for i, pg := range pages {
		titles[i] = *pg.Title
	}
	return titles
}

func TestGetTrashShowUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	now := time.Now()

	t.Run("page_trash:readを持つトピックの、復元できるページだけを新しい順に返す", func(t *testing.T) {
		t.Parallel()

		// スペースの閲覧者はゴミ箱を見られず、トピックの編集者として参加したトピックでだけ見られる
		f := setupTrashShowFixture(t, "per-topic", model.SpaceRoleViewer)
		editorTopicID := f.addTopic(t, 1, model.TopicVisibilityPublic, model.TopicRoleEditor)
		viewerTopicID := f.addTopic(t, 2, model.TopicVisibilityPublic, model.TopicRoleViewer)
		notJoinedTopicID := f.addTopic(t, 3, model.TopicVisibilityPublic, "")

		f.addTrashedPage(t, editorTopicID, 1, "古いページ", now.Add(-2*time.Hour))
		f.addTrashedPage(t, editorTopicID, 2, "新しいページ", now.Add(-1*time.Hour))
		f.addTrashedPage(t, editorTopicID, 3, "期限を過ぎたページ", now.AddDate(0, 0, -model.PageTrashRetentionDays).Add(-time.Minute))
		f.addTrashedPage(t, viewerTopicID, 4, "閲覧者のトピックのページ", now.Add(-time.Hour))
		f.addTrashedPage(t, notJoinedTopicID, 5, "参加していないトピックのページ", now.Add(-time.Hour))
		testutil.NewPageBuilder(t, f.tx).
			WithSpaceID(f.spaceID).
			WithTopicID(editorTopicID).
			WithNumber(6).
			WithTitle("ゴミ箱に入れていないページ").
			Build()

		output, err := f.usecase().Execute(ctx, GetTrashShowInput{
			SpaceIdentifier: f.identifier,
			UserID:          f.userID,
			Page:            1,
			PageLimit:       100,
			Now:             now,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := pageTitles(output.Pages)
		want := []string{"新しいページ", "古いページ"}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("ページ = %v、期待値 = %v", got, want)
		}
		if output.TotalCount != 2 {
			t.Errorf("TotalCount = %d、期待値 = 2", output.TotalCount)
		}
		if _, ok := output.TopicMap[editorTopicID]; !ok {
			t.Error("TopicMapにページのトピックが無い")
		}
	})

	t.Run("廃棄したページと、廃棄したトピックのページは出さない", func(t *testing.T) {
		t.Parallel()

		f := setupTrashShowFixture(t, "discarded", model.SpaceRoleAdmin)
		topicID := f.addTopic(t, 1, model.TopicVisibilityPublic, "")
		discardedTopicID := testutil.NewTopicBuilder(t, f.tx).
			WithSpaceID(f.spaceID).
			WithNumber(2).
			WithName("廃棄したトピック").
			WithDiscarded().
			Build()

		f.addTrashedPage(t, topicID, 1, "残っているページ", now.Add(-time.Hour))
		f.addTrashedPage(t, discardedTopicID, 2, "廃棄したトピックのページ", now.Add(-time.Hour))
		testutil.NewPageBuilder(t, f.tx).
			WithSpaceID(f.spaceID).
			WithTopicID(topicID).
			WithNumber(3).
			WithTitle("廃棄したページ").
			WithTrashedAt(now.Add(-time.Hour)).
			WithDiscarded().
			Build()

		output, err := f.usecase().Execute(ctx, GetTrashShowInput{
			SpaceIdentifier: f.identifier,
			UserID:          f.userID,
			Page:            1,
			PageLimit:       100,
			Now:             now,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := pageTitles(output.Pages)
		if len(got) != 1 || got[0] != "残っているページ" {
			t.Errorf("ページ = %v、期待値 = [残っているページ]", got)
		}
	})

	t.Run("オフセットでページ送りする", func(t *testing.T) {
		t.Parallel()

		f := setupTrashShowFixture(t, "pagination", model.SpaceRoleAdmin)
		topicID := f.addTopic(t, 1, model.TopicVisibilityPublic, "")
		for i := range 3 {
			f.addTrashedPage(t, topicID, model.PageNumber(i+1), "ページ"+string(rune('1'+i)), now.Add(-time.Duration(i+1)*time.Hour))
		}

		output, err := f.usecase().Execute(ctx, GetTrashShowInput{
			SpaceIdentifier: f.identifier,
			UserID:          f.userID,
			Page:            2,
			PageLimit:       2,
			Now:             now,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		got := pageTitles(output.Pages)
		if len(got) != 1 || got[0] != "ページ3" {
			t.Errorf("2ページ目のページ = %v、期待値 = [ページ3]", got)
		}
		if output.TotalCount != 3 {
			t.Errorf("TotalCount = %d、期待値 = 3", output.TotalCount)
		}
	})

	t.Run("どのトピックでもpage_trash:readを持たないメンバーは拒否する", func(t *testing.T) {
		t.Parallel()

		f := setupTrashShowFixture(t, "no-trash", model.SpaceRoleViewer)
		f.addTopic(t, 1, model.TopicVisibilityPublic, model.TopicRoleViewer)

		_, err := f.usecase().Execute(ctx, GetTrashShowInput{
			SpaceIdentifier: f.identifier,
			UserID:          f.userID,
			Page:            1,
			PageLimit:       100,
			Now:             now,
		})

		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeForbidden {
			t.Fatalf("ForbiddenのAppErrorを期待したが、%vだった", err)
		}
	})

	t.Run("スペースのメンバーでない閲覧者は拒否する", func(t *testing.T) {
		t.Parallel()

		f := setupTrashShowFixture(t, "outsider", model.SpaceRoleAdmin)
		f.addTopic(t, 1, model.TopicVisibilityPublic, "")
		outsiderID := testutil.NewUserBuilder(t, f.tx).
			WithEmail("trash-show-outsider-other@example.com").
			WithAtname("trash_show_outsider_other").
			Build()

		_, err := f.usecase().Execute(ctx, GetTrashShowInput{
			SpaceIdentifier: f.identifier,
			UserID:          outsiderID,
			Page:            1,
			PageLimit:       100,
			Now:             now,
		})

		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeForbidden {
			t.Fatalf("ForbiddenのAppErrorを期待したが、%vだった", err)
		}
	})

	t.Run("存在しないスペースは見つからない", func(t *testing.T) {
		t.Parallel()

		f := setupTrashShowFixture(t, "missing", model.SpaceRoleAdmin)

		_, err := f.usecase().Execute(ctx, GetTrashShowInput{
			SpaceIdentifier: "trash-show-no-such-space",
			UserID:          f.userID,
			Page:            1,
			PageLimit:       100,
			Now:             now,
		})

		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Fatalf("ResourceNotFoundのAppErrorを期待したが、%vだった", err)
		}
	})
}

// 組み込みのロールはどれもスペースでtopic:readを持つため、開けない非公開トピックはUseCaseの
// テストでは作れない。topic:readを持たないスコープの組み合わせで、判定そのものを確かめる。
func TestTopicAccess_trashVisibleTopics(t *testing.T) {
	t.Parallel()

	publicTopic := &model.Topic{ID: "public", Visibility: model.TopicVisibilityPublic}
	privateTopic := &model.Topic{ID: "private", Visibility: model.TopicVisibilityPrivate}
	joinedPrivateTopic := &model.Topic{ID: "joined-private", Visibility: model.TopicVisibilityPrivate}
	noTrashTopic := &model.Topic{ID: "no-trash", Visibility: model.TopicVisibilityPublic}

	// スペースではpage_trash:readだけを持ち、topic:readを持たない。参加したトピックではロールの
	// スコープが加わる。no-trashのトピックでは、どちらのスコープも持たない
	access := &topicAccess{
		authorize: func(topicMember *model.TopicMember) policy.Authorizer {
			if topicMember == nil {
				return policy.NewMemberPolicy([]model.Scope{model.ScopePageTrashRead}, nil)
			}
			if topicMember.TopicID == noTrashTopic.ID {
				return policy.NewMemberPolicy(nil, nil)
			}
			return policy.NewMemberPolicy([]model.Scope{model.ScopePageTrashRead}, topicMember.Role.Scopes())
		},
		topics: []*model.Topic{publicTopic, privateTopic, joinedPrivateTopic, noTrashTopic},
		memberByTopic: map[model.TopicID]*model.TopicMember{
			joinedPrivateTopic.ID: {TopicID: joinedPrivateTopic.ID, Role: model.TopicRoleViewer},
			noTrashTopic.ID:       {TopicID: noTrashTopic.ID, Role: model.TopicRoleViewer},
		},
	}

	got := access.trashVisibleTopics()

	gotIDs := make([]model.TopicID, len(got))
	for i, topic := range got {
		gotIDs[i] = topic.ID
	}
	want := []model.TopicID{publicTopic.ID, joinedPrivateTopic.ID}
	if len(gotIDs) != len(want) || gotIDs[0] != want[0] || gotIDs[1] != want[1] {
		t.Errorf("トピック = %v、期待値 = %v", gotIDs, want)
	}
}
