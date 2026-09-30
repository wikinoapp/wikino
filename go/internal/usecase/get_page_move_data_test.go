package usecase

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetPageMoveDataUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	uc := NewGetPageMoveDataUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

	// テストデータを作成
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("gpmd-owner@example.com").
		WithAtname("gpmdowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpmd-space").
		WithName("GPMD Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("トピック1").
		WithVisibility(0). // public
		Build()
	topicID2 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("トピック2").
		WithVisibility(0). // public
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID2).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithNumber(1).
		WithTitle("テストページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	t.Run("存在しないスペースでAppErrorが返る", func(t *testing.T) {
		_, err := uc.Execute(ctx, GetPageMoveDataInput{
			SpaceIdentifier: "nonexistent",
			PageNumber:      1,
			UserID:          ownerID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("AppErrorを期待したが、nilだった")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("スペースメンバーでないユーザーでAppErrorが返る", func(t *testing.T) {
		nonMemberID := testutil.NewUserBuilder(t, tx).
			WithEmail("gpmd-nonmember@example.com").
			WithAtname("gpmdnonmember").
			Build()
		_, err := uc.Execute(ctx, GetPageMoveDataInput{
			SpaceIdentifier: "gpmd-space",
			PageNumber:      1,
			UserID:          nonMemberID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("AppErrorを期待したが、nilだった")
		}
		if ae.Code != model.AppErrCodeForbidden {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("存在しないページでAppErrorが返る", func(t *testing.T) {
		_, err := uc.Execute(ctx, GetPageMoveDataInput{
			SpaceIdentifier: "gpmd-space",
			PageNumber:      999,
			UserID:          ownerID,
		})

		ae := model.AsAppError(err)
		if ae == nil {
			t.Fatal("AppErrorを期待したが、nilだった")
		}
		if ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v、期待値 = %v", ae.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("正常系: すべてのデータが取得できる", func(t *testing.T) {
		output, err := uc.Execute(ctx, GetPageMoveDataInput{
			SpaceIdentifier: "gpmd-space",
			PageNumber:      1,
			UserID:          ownerID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output == nil {
			t.Fatal("出力がnil")
		}
		if output.Space.Name != "GPMD Space" {
			t.Errorf("Space.Name = %q、期待値 = %q", output.Space.Name, "GPMD Space")
		}
		if output.SpaceMember == nil {
			t.Fatal("SpaceMemberがnil")
		}
		if output.Page == nil {
			t.Fatal("Pageがnil")
		}
		if output.TopicMember == nil {
			t.Fatal("TopicMemberがnil")
		}
		if output.CurrentTopic == nil {
			t.Fatal("CurrentTopicがnil")
		}
		if output.CurrentTopic.Name != "トピック1" {
			t.Errorf("CurrentTopic.Name = %q、期待値 = %q", output.CurrentTopic.Name, "トピック1")
		}
		// AvailableTopicsは現在のトピックを除外するので、トピック2のみ
		if len(output.AvailableTopics) != 1 {
			t.Fatalf("AvailableTopicsの件数 = %d、期待値 = 1", len(output.AvailableTopics))
		}
		if output.AvailableTopics[0].Name != "トピック2" {
			t.Errorf("AvailableTopics[0].Name = %q、期待値 = %q", output.AvailableTopics[0].Name, "トピック2")
		}
	})
}

// TestGetPageMoveDataUsecase_Execute_AvailableTopicsは、移動先候補をページを作成できるトピックに
// 限ることを扱う。編集者はすべてのトピックを、閲覧者は所属トピックのうちトピック編集者以上のもの
// だけを候補にする
func TestGetPageMoveDataUsecase_Execute_AvailableTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := NewGetPageMoveDataUsecase(
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("gpmd-available").
		Build()
	// トピックは次の5つ。移動するページは1に属する
	//   - 1: 閲覧者はトピック編集者 (ページを編集できるよう)
	//   - 2: 閲覧者はトピック管理者
	//   - 3: 閲覧者はトピック閲覧者
	//   - 4: 閲覧者は参加していない
	//   - 5: 非公開で、閲覧者は参加していない
	topicIDs := map[int32]model.TopicID{}
	for number := int32(1); number <= 5; number++ {
		b := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(number).
			WithName(fmt.Sprintf("トピック%d", number))
		if number == 5 {
			b = b.WithVisibility(int32(model.TopicVisibilityPrivate))
		}
		topicIDs[number] = b.Build()
	}
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicIDs[1]).
		WithNumber(1).
		WithTitle("移動するページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	newMember := func(atname string, role model.SpaceRole, topicRoles map[int32]model.TopicRole) model.UserID {
		userID := testutil.NewUserBuilder(t, tx).
			WithEmail(atname + "@example.com").
			WithAtname(atname).
			Build()
		spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
			WithSpaceID(spaceID).
			WithUserID(userID).
			WithRole(role).
			Build()
		for number, topicRole := range topicRoles {
			testutil.NewTopicMemberBuilder(t, tx).
				WithSpaceID(spaceID).
				WithTopicID(topicIDs[number]).
				WithSpaceMemberID(spaceMemberID).
				WithRole(topicRole).
				Build()
		}
		return userID
	}
	editorID := newMember("gpmdeditor", model.SpaceRoleEditor, nil)
	viewerID := newMember("gpmdviewer", model.SpaceRoleViewer, map[int32]model.TopicRole{
		1: model.TopicRoleEditor,
		2: model.TopicRoleAdmin,
		3: model.TopicRoleViewer,
	})

	tests := []struct {
		name        string
		userID      model.UserID
		wantNumbers []int32
	}{
		{name: "編集者は現在のトピック以外のすべてのトピックが候補になる", userID: editorID, wantNumbers: []int32{2, 3, 4, 5}},
		{name: "閲覧者は所属トピックのうちトピック編集者以上のものだけが候補になる", userID: viewerID, wantNumbers: []int32{2}},
	}

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	// サブテストは親のトランザクションを共有するため、並列にしない
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(ctx, GetPageMoveDataInput{
				SpaceIdentifier: "gpmd-available",
				PageNumber:      1,
				UserID:          tt.userID,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}

			got := make([]int32, len(output.AvailableTopics))
			for i, topic := range output.AvailableTopics {
				got[i] = topic.Number
			}
			slices.Sort(got)
			if !slices.Equal(got, tt.wantNumbers) {
				t.Errorf("移動先候補のトピック番号 = %v、期待値 = %v", got, tt.wantNumbers)
			}
		})
	}
}
