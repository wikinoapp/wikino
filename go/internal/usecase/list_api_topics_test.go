package usecase

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// apiTopicFixtureは公開APIのトピックのテストで使うスペースとトピック。
// トピックは次の4つを作る
//   - 1: 公開トピック
//   - 2: 非公開トピック。メンバーはトピック編集者として参加している
//   - 3: 非公開トピック。メンバーは参加していない
//   - 4: 廃棄済みの公開トピック
//
// どのスペースのロールも `topic:read` を持つため、トークンが `topic:read` を持てば2と3の両方を開ける
type apiTopicFixture struct {
	space         *model.Space
	spaceMemberID model.SpaceMemberID
	userID        model.UserID
	// topicIDsはトピック番号ごとのトピックID
	topicIDs map[int32]model.TopicID
}

func setupAPITopicFixture(t *testing.T, tx *sql.Tx, identifier string, memberRole model.SpaceRole) apiTopicFixture {
	t.Helper()

	// 1つのトランザクションで複数のフィクスチャを作れるよう、ユーザーをスペースの識別子で区別する
	atname := strings.ReplaceAll(identifier, "-", "_")
	userID := testutil.NewUserBuilder(t, tx).WithAtname(atname).WithEmail(atname + "@example.com").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(memberRole).
		Build()

	publicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("公開").Build()
	joinedPrivateID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(2).WithName("参加している非公開").
		WithVisibility(int32(model.TopicVisibilityPrivate)).Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(joinedPrivateID).
		WithSpaceMemberID(spaceMemberID).
		WithRole(model.TopicRoleEditor).
		Build()
	notJoinedPrivateID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(3).WithName("参加していない非公開").
		WithVisibility(int32(model.TopicVisibilityPrivate)).Build()
	discardedID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(4).WithName("廃棄済み").WithDiscarded().Build()

	return apiTopicFixture{
		space:         &model.Space{ID: spaceID, Identifier: model.SpaceIdentifier(identifier)},
		spaceMemberID: spaceMemberID,
		userID:        userID,
		topicIDs:      map[int32]model.TopicID{1: publicID, 2: joinedPrivateID, 3: notJoinedPrivateID, 4: discardedID},
	}
}

// principalはフィクスチャのメンバーが、tokenScopesを持つトークンで呼び出した主体を返す
func (f apiTopicFixture) principal(memberRole model.SpaceRole, tokenScopes []model.Scope) *model.APIPrincipal {
	return &model.APIPrincipal{
		User:        &model.User{ID: f.userID},
		Space:       f.space,
		SpaceMember: &model.SpaceMember{ID: f.spaceMemberID, SpaceID: f.space.ID, UserID: f.userID, Role: memberRole, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      policy.ExpandAPITokenScopes(tokenScopes),
	}
}

func newListAPITopicsUC(tx *sql.Tx) *ListAPITopicsUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewListAPITopicsUsecase(repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

// メンバーのスペースのロール
var (
	// 一般メンバー: ページを書け、トークンを作れる編集者
	apiTopicRegularMemberRole = model.SpaceRoleEditor
	// 管理者
	apiTopicAdminMemberRole = model.SpaceRoleAdmin
)

func TestListAPITopicsUsecase_Execute_Visibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		wantNumbers []int32
	}{
		{
			name:        "一般メンバーでトークンがtopic:readを持てば、参加していない非公開トピックも返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			wantNumbers: []int32{1, 2, 3},
		},
		{
			name:        "閲覧者でもトークンがtopic:readを持てば、すべての非公開トピックを返す",
			memberRole:  model.SpaceRoleViewer,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			wantNumbers: []int32{1, 2, 3},
		},
		{
			name:        "一般メンバーでトークンがtopic:readを持たなければ、公開トピックだけを返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite},
			wantNumbers: []int32{1},
		},
		{
			name:        "管理者でトークンがtopic:readを持てば、すべての非公開トピックを返す",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			wantNumbers: []int32{1, 2, 3},
		},
		{
			name:        "管理者でもトークンがtopic:readを持たなければ、公開トピックだけを返す",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead},
			wantNumbers: []int32{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPITopicFixture(t, tx, "api-topic-visibility", tt.memberRole)

			output, err := newListAPITopicsUC(tx).Execute(t.Context(), ListAPITopicsInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				Limit:           20,
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			assertTopicNumbers(t, output.Topics, tt.wantNumbers)
			if output.HasNext {
				t.Error("HasNext = true、期待値 = false")
			}
		})
	}
}

func TestListAPITopicsUsecase_Execute_Cursor(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPITopicFixture(t, tx, "api-topic-cursor", apiTopicAdminMemberRole)
	// 番号の間に見えないトピック (廃棄済みの4) を挟み、番号が連続しなくても辿れることを確かめる
	for number := int32(5); number <= 7; number++ {
		testutil.NewTopicBuilder(t, tx).WithSpaceID(f.space.ID).WithNumber(number).WithName(fmt.Sprintf("追加のトピック%d", number)).Build()
	}
	principal := f.principal(apiTopicAdminMemberRole, []model.Scope{model.ScopeTopicRead})
	uc := newListAPITopicsUC(tx)

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("カーソルで全件を重複なく辿れる", func(t *testing.T) {
		var got []int32
		var afterNumber *int32
		for range 10 {
			output, err := uc.Execute(t.Context(), ListAPITopicsInput{
				Principal:       principal,
				SpaceIdentifier: f.space.Identifier,
				AfterNumber:     afterNumber,
				Limit:           2,
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(output.Topics) > 2 {
				t.Fatalf("1ページの件数 = %d、期待値 = 2以下", len(output.Topics))
			}
			for _, topic := range output.Topics {
				got = append(got, topic.Number)
			}
			if !output.HasNext {
				break
			}
			afterNumber = &output.Topics[len(output.Topics)-1].Number
		}

		want := []int32{1, 2, 3, 5, 6, 7}
		if len(got) != len(want) {
			t.Fatalf("辿ったトピック = %v、期待値 = %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("辿ったトピック = %v、期待値 = %v", got, want)
			}
		}
	})

	t.Run("残りがちょうどlimit件なら次のページは無い", func(t *testing.T) {
		afterNumber := int32(3)
		output, err := uc.Execute(t.Context(), ListAPITopicsInput{
			Principal:       principal,
			SpaceIdentifier: f.space.Identifier,
			AfterNumber:     &afterNumber,
			Limit:           3,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		assertTopicNumbers(t, output.Topics, []int32{5, 6, 7})
		if output.HasNext {
			t.Error("HasNext = true、期待値 = false")
		}
	})

	t.Run("最後のトピックより後を指すカーソルは空の一覧", func(t *testing.T) {
		afterNumber := int32(7)
		output, err := uc.Execute(t.Context(), ListAPITopicsInput{
			Principal:       principal,
			SpaceIdentifier: f.space.Identifier,
			AfterNumber:     &afterNumber,
			Limit:           20,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(output.Topics) != 0 || output.HasNext {
			t.Errorf("Topics = %d件、HasNext = %v、期待値 = 0件、false", len(output.Topics), output.HasNext)
		}
	})
}

func TestListAPITopicsUsecase_Execute_OtherSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPITopicFixture(t, tx, "api-topic-bound", apiTopicAdminMemberRole)
	setupAPITopicFixture(t, tx, "api-topic-other", apiTopicAdminMemberRole)

	output, err := newListAPITopicsUC(tx).Execute(t.Context(), ListAPITopicsInput{
		Principal:       f.principal(apiTopicAdminMemberRole, []model.Scope{model.ScopeTopicRead}),
		SpaceIdentifier: "api-topic-other",
		Limit:           20,
	})
	if output != nil {
		t.Errorf("output = %+v、期待値 = nil", output)
	}
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
	}
}

func assertTopicNumbers(t *testing.T, topics []*model.Topic, want []int32) {
	t.Helper()

	got := make([]int32, len(topics))
	for i, topic := range topics {
		got[i] = topic.Number
	}
	if len(got) != len(want) {
		t.Fatalf("トピックの番号 = %v、期待値 = %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("トピックの番号 = %v、期待値 = %v", got, want)
		}
	}
}
