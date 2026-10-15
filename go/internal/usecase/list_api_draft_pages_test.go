package usecase

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// setupAPIDraftPageFixtureは、setupAPITopicFixtureのトピックに次のページと下書きを作る。
// 下書きはどれもフィクスチャのメンバーのもので、番号が小さいほど更新日時が新しい
// (番号nのページの下書きの更新日時は基準日時のn時間前)
//   - 1: 公開トピックの公開済みのページの下書き
//   - 2: 公開トピックの未公開のページ (新しいページ) の下書き
//   - 3: 公開トピックのページ。ほかのメンバーの下書きだけがある
//   - 4: 公開トピックのゴミ箱のページの下書き
//   - 5: 参加している非公開トピックのページの下書き
//   - 6: 参加していない非公開トピックのページの下書き
//   - 7: 廃棄済みのトピックのページの下書き
//   - 8: 公開トピックの下書きの無いページ
func setupAPIDraftPageFixture(t *testing.T, tx *sql.Tx, identifier string, memberRole model.SpaceRole) apiTopicFixture {
	t.Helper()

	f := setupAPITopicFixture(t, tx, identifier, memberRole)
	otherAtname := strings.ReplaceAll(identifier, "-", "_") + "_drafter"
	otherUserID := testutil.NewUserBuilder(t, tx).WithAtname(otherAtname).WithEmail(otherAtname + "@example.com").Build()
	otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(f.space.ID).WithUserID(otherUserID).Build()

	pages := []struct {
		number      model.PageNumber
		topicNumber int32
		draftMember model.SpaceMemberID
		modify      func(b *testutil.PageBuilder) *testutil.PageBuilder
	}{
		{number: 1, topicNumber: 1, draftMember: f.spaceMemberID},
		{number: 2, topicNumber: 1, draftMember: f.spaceMemberID, modify: (*testutil.PageBuilder).WithUnpublished},
		{number: 3, topicNumber: 1, draftMember: otherMemberID},
		{number: 4, topicNumber: 1, draftMember: f.spaceMemberID, modify: (*testutil.PageBuilder).WithTrashed},
		{number: 5, topicNumber: 2, draftMember: f.spaceMemberID},
		{number: 6, topicNumber: 3, draftMember: f.spaceMemberID},
		{number: 7, topicNumber: 4, draftMember: f.spaceMemberID},
		{number: 8, topicNumber: 1},
	}
	for _, p := range pages {
		modifiedAt := apiPageBaseTime.Add(-time.Duration(p.number) * time.Hour)
		b := testutil.NewPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithTopicID(f.topicIDs[p.topicNumber]).
			WithNumber(p.number).
			WithTitle(fmt.Sprintf("ページ%d", p.number)).
			WithModifiedAt(modifiedAt)
		if p.modify != nil {
			b = p.modify(b)
		}
		pageID := b.Build()
		if p.draftMember == "" {
			continue
		}
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(f.space.ID).
			WithPageID(pageID).
			WithSpaceMemberID(p.draftMember).
			WithTopicID(f.topicIDs[p.topicNumber]).
			WithTitle(fmt.Sprintf("下書き%d", p.number)).
			WithBody(fmt.Sprintf("下書きの本文%d", p.number)).
			WithModifiedAt(modifiedAt).
			Build()
	}
	return f
}

func newListAPIDraftPagesUC(tx *sql.Tx) *ListAPIDraftPagesUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewListAPIDraftPagesUsecase(repository.NewDraftPageRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

func TestListAPIDraftPagesUsecase_Execute(t *testing.T) {
	t.Parallel()

	readScopes := []model.Scope{model.ScopeDraftPageRead, model.ScopeTopicRead}
	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		wantNumbers []model.PageNumber
	}{
		{
			name:        "トークンがtopic:readを持たなければ公開トピックの下書きだけを返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeDraftPageRead},
			wantNumbers: []model.PageNumber{1, 2},
		},
		{
			name:        "トークンがtopic:readを持てば非公開トピックの下書きも返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: readScopes,
			wantNumbers: []model.PageNumber{1, 2, 5, 6},
		},
		{
			name:        "draft_page:writeはdraft_page:readを含意する",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeDraftPageWrite},
			wantNumbers: []model.PageNumber{1, 2},
		},
		{
			// 閲覧者はスペースのロールではdraft_page:readを持たないが、トピックの編集者として参加しているトピックでは持つ
			name:        "メンバーが下書きを読めるトピックの下書きだけを返す",
			memberRole:  model.SpaceRoleViewer,
			tokenScopes: readScopes,
			wantNumbers: []model.PageNumber{5},
		},
		{
			name:        "トークンがdraft_page:readを持たなければ返さない",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite, model.ScopeTopicRead},
			wantNumbers: []model.PageNumber{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIDraftPageFixture(t, tx, "api-draft-list", tt.memberRole)

			output, err := newListAPIDraftPagesUC(tx).Execute(t.Context(), ListAPIDraftPagesInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: f.space.Identifier,
				Limit:           20,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}

			got := make([]model.PageNumber, len(output.DraftPages))
			for i, draft := range output.DraftPages {
				got[i] = draft.Page.Number
				topic := output.Topics[draft.TopicID]
				if topic == nil || topic.ID != draft.TopicID {
					t.Errorf("下書き%dのトピック = %+v、期待値 = 下書きの所属トピック", draft.Page.Number, topic)
				}
			}
			if !slices.Equal(got, tt.wantNumbers) {
				t.Errorf("ページ番号 = %v、期待値 = %v", got, tt.wantNumbers)
			}
			if output.HasNext {
				t.Error("HasNext = true、期待値 = false")
			}
		})
	}
}

func TestListAPIDraftPagesUsecase_Execute_Pagination(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPIDraftPageFixture(t, tx, "api-draft-list-page", apiTopicRegularMemberRole)
	uc := newListAPIDraftPagesUC(tx)
	principal := f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopeDraftPageRead, model.ScopeTopicRead})

	var got []model.PageNumber
	var after *APIDraftPageCursor
	for range 3 {
		output, err := uc.Execute(t.Context(), ListAPIDraftPagesInput{
			Principal:       principal,
			SpaceIdentifier: f.space.Identifier,
			After:           after,
			Limit:           3,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		for _, draft := range output.DraftPages {
			got = append(got, draft.Page.Number)
		}
		if !output.HasNext {
			break
		}
		last := output.DraftPages[len(output.DraftPages)-1]
		after = &APIDraftPageCursor{ModifiedAt: last.ModifiedAt, DraftPageID: last.ID}
	}

	if want := []model.PageNumber{1, 2, 5, 6}; !slices.Equal(got, want) {
		t.Errorf("辿ったページ番号 = %v、期待値 = %v", got, want)
	}
}

func TestListAPIDraftPagesUsecase_Execute_OtherSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPIDraftPageFixture(t, tx, "api-draft-list-own", apiTopicAdminMemberRole)
	setupAPIDraftPageFixture(t, tx, "api-draft-list-other", apiTopicAdminMemberRole)

	output, err := newListAPIDraftPagesUC(tx).Execute(t.Context(), ListAPIDraftPagesInput{
		Principal:       f.principal(apiTopicAdminMemberRole, []model.Scope{model.ScopeDraftPageRead}),
		SpaceIdentifier: "api-draft-list-other",
		Limit:           20,
	})
	if output != nil {
		t.Errorf("output = %+v、期待値 = nil", output)
	}
	if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
	}
}
