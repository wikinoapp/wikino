package usecase

import (
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// apiDraftPageRevisionFixtureは、setupAPIDraftPageFixtureの下書きに作ったリビジョンのIDをタイトルで引けるようにしたもの
type apiDraftPageRevisionFixture struct {
	apiTopicFixture
	revisionIDs map[string]model.DraftPageRevisionID
}

// setupAPIDraftPageRevisionFixtureは、setupAPIDraftPageFixtureの下書きに次のリビジョンを作る。
// タイトルは「p{ページ番号}-{連番}」で、連番が大きいほど作成日時が新しい
//   - ページ1の下書き: p1-1・p1-2・p1-3
//   - ページ2の下書き: p2-1
//   - ページ3のほかのメンバーの下書き: p3-1
//   - ページ5の参加している非公開トピックの下書き: p5-1
func setupAPIDraftPageRevisionFixture(t *testing.T, tx *sql.Tx, identifier string, memberRole model.SpaceRole) apiDraftPageRevisionFixture {
	t.Helper()

	f := apiDraftPageRevisionFixture{
		apiTopicFixture: setupAPIDraftPageFixture(t, tx, identifier, memberRole),
		revisionIDs:     map[string]model.DraftPageRevisionID{},
	}
	revisions := []struct {
		pageNumber model.PageNumber
		title      string
	}{
		{pageNumber: 1, title: "p1-1"},
		{pageNumber: 1, title: "p1-2"},
		{pageNumber: 1, title: "p1-3"},
		{pageNumber: 2, title: "p2-1"},
		{pageNumber: 3, title: "p3-1"},
		{pageNumber: 5, title: "p5-1"},
	}
	for i, r := range revisions {
		var draftPageID, spaceMemberID string
		err := tx.QueryRowContext(t.Context(),
			`SELECT dp.id, dp.space_member_id FROM draft_pages dp
			 INNER JOIN pages p ON dp.page_id = p.id
			 WHERE p.space_id = $1 AND p.number = $2`,
			string(f.space.ID), int32(r.pageNumber),
		).Scan(&draftPageID, &spaceMemberID)
		if err != nil {
			t.Fatalf("ページ%dの下書きの取得に失敗: %v", r.pageNumber, err)
		}
		f.revisionIDs[r.title] = testutil.NewDraftPageRevisionBuilder(t, tx).
			WithDraftPageID(model.DraftPageID(draftPageID)).
			WithSpaceID(f.space.ID).
			WithSpaceMemberID(model.SpaceMemberID(spaceMemberID)).
			WithTitle(r.title).
			WithBody("本文 " + r.title).
			WithCreatedAt(apiPageBaseTime.Add(time.Duration(i) * time.Minute)).
			Build()
	}
	return f
}

func newListAPIDraftPageRevisionsUC(tx *sql.Tx) *ListAPIDraftPageRevisionsUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewListAPIDraftPageRevisionsUsecase(
		repository.NewPageRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)
}

func TestListAPIDraftPageRevisionsUsecase_Execute(t *testing.T) {
	t.Parallel()

	readScopes := []model.Scope{model.ScopeDraftPageRead, model.ScopeTopicRead}
	tests := []struct {
		name        string
		tokenScopes []model.Scope
		identifier  model.SpaceIdentifier
		pageNumber  int32
		// wantTitlesは返るリビジョンのタイトル。nilなら未存在のエラーを期待する
		wantTitles []string
	}{
		{
			name:        "トークンの持ち主の下書きのリビジョンだけを新しい順に返す",
			tokenScopes: []model.Scope{model.ScopeDraftPageRead},
			pageNumber:  1,
			wantTitles:  []string{"p1-3", "p1-2", "p1-1"},
		},
		{
			name:        "draft_page:writeはdraft_page:readを含意する",
			tokenScopes: []model.Scope{model.ScopeDraftPageWrite},
			pageNumber:  2,
			wantTitles:  []string{"p2-1"},
		},
		{
			name:        "トークンがtopic:readを持てば非公開トピックの下書きのリビジョンを返す",
			tokenScopes: readScopes,
			pageNumber:  5,
			wantTitles:  []string{"p5-1"},
		},
		{
			name:        "トークンがtopic:readを持たなければ非公開トピックの下書きは未存在",
			tokenScopes: []model.Scope{model.ScopeDraftPageRead},
			pageNumber:  5,
		},
		{
			name:        "ほかのメンバーの下書きは未存在",
			tokenScopes: readScopes,
			pageNumber:  3,
		},
		{
			name:        "下書きの無いページは未存在",
			tokenScopes: readScopes,
			pageNumber:  8,
		},
		{
			name:        "トークンがdraft_page:readを持たなければ未存在",
			tokenScopes: []model.Scope{model.ScopePageWrite},
			pageNumber:  1,
		},
		{
			name:        "トークンが属さないスペースは未存在",
			tokenScopes: readScopes,
			identifier:  "api-draft-rev-list-other",
			pageNumber:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIDraftPageRevisionFixture(t, tx, "api-draft-rev-list", apiTopicRegularMemberRole)
			setupAPIDraftPageRevisionFixture(t, tx, "api-draft-rev-list-other", apiTopicRegularMemberRole)
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}

			output, err := newListAPIDraftPageRevisionsUC(tx).Execute(t.Context(), ListAPIDraftPageRevisionsInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: identifier,
				PageNumber:      tt.pageNumber,
				Limit:           20,
			})

			if tt.wantTitles == nil {
				if output != nil {
					t.Errorf("output = %+v、期待値 = nil", output)
				}
				if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
					t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			got := make([]string, len(output.Revisions))
			for i, r := range output.Revisions {
				got[i] = r.Title
			}
			if !slices.Equal(got, tt.wantTitles) {
				t.Errorf("リビジョンのタイトル = %v、期待値 = %v", got, tt.wantTitles)
			}
			if output.HasNext {
				t.Error("HasNext = true、期待値 = false")
			}
		})
	}
}

func TestListAPIDraftPageRevisionsUsecase_Execute_Pagination(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAPIDraftPageRevisionFixture(t, tx, "api-draft-rev-list-page", apiTopicRegularMemberRole)
	uc := newListAPIDraftPageRevisionsUC(tx)
	principal := f.principal(apiTopicRegularMemberRole, []model.Scope{model.ScopeDraftPageRead})

	var got []string
	var after *APIDraftPageRevisionCursor
	for range 3 {
		output, err := uc.Execute(t.Context(), ListAPIDraftPageRevisionsInput{
			Principal:       principal,
			SpaceIdentifier: f.space.Identifier,
			PageNumber:      1,
			After:           after,
			Limit:           2,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		for _, r := range output.Revisions {
			got = append(got, r.Title)
		}
		if !output.HasNext {
			break
		}
		last := output.Revisions[len(output.Revisions)-1]
		after = &APIDraftPageRevisionCursor{CreatedAt: last.CreatedAt, RevisionID: last.ID}
	}

	if want := []string{"p1-3", "p1-2", "p1-1"}; !slices.Equal(got, want) {
		t.Errorf("辿ったリビジョンのタイトル = %v、期待値 = %v", got, want)
	}
}
