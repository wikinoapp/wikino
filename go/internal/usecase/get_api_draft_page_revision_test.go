package usecase

import (
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetAPIDraftPageRevisionUC(tx *sql.Tx) *GetAPIDraftPageRevisionUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewGetAPIDraftPageRevisionUsecase(
		repository.NewPageRepository(q),
		repository.NewDraftPageRepository(q),
		repository.NewDraftPageRevisionRepository(q),
		repository.NewTopicRepository(q),
		repository.NewTopicMemberRepository(q),
	)
}

func TestGetAPIDraftPageRevisionUsecase_Execute(t *testing.T) {
	t.Parallel()

	readScopes := []model.Scope{model.ScopeDraftPageRead, model.ScopeTopicRead}
	tests := []struct {
		name        string
		tokenScopes []model.Scope
		identifier  model.SpaceIdentifier
		pageNumber  int32
		// revisionTitleは指定するリビジョンのタイトル。空ならどの下書きにも無いIDを指定する
		revisionTitle string
		wantFound     bool
	}{
		{
			name:          "トークンの持ち主の下書きのリビジョンを返す",
			tokenScopes:   []model.Scope{model.ScopeDraftPageRead},
			pageNumber:    1,
			revisionTitle: "p1-2",
			wantFound:     true,
		},
		{
			name:          "トークンがtopic:readを持てば非公開トピックの下書きのリビジョンを返す",
			tokenScopes:   readScopes,
			pageNumber:    5,
			revisionTitle: "p5-1",
			wantFound:     true,
		},
		{
			name:          "別のページの下書きのリビジョンは未存在",
			tokenScopes:   readScopes,
			pageNumber:    1,
			revisionTitle: "p2-1",
		},
		{
			name:          "ほかのメンバーの下書きのリビジョンは、そのページを指定しても未存在",
			tokenScopes:   readScopes,
			pageNumber:    3,
			revisionTitle: "p3-1",
		},
		{
			name:          "ほかのメンバーの下書きのリビジョンは、自分の下書きのページを指定しても未存在",
			tokenScopes:   readScopes,
			pageNumber:    1,
			revisionTitle: "p3-1",
		},
		{
			name:          "トークンがtopic:readを持たなければ非公開トピックの下書きのリビジョンは未存在",
			tokenScopes:   []model.Scope{model.ScopeDraftPageRead},
			pageNumber:    5,
			revisionTitle: "p5-1",
		},
		{
			name:          "トークンがdraft_page:readを持たなければ未存在",
			tokenScopes:   []model.Scope{model.ScopePageWrite},
			pageNumber:    1,
			revisionTitle: "p1-2",
		},
		{
			name:        "存在しないリビジョンは未存在",
			tokenScopes: readScopes,
			pageNumber:  1,
		},
		{
			name:          "トークンが属さないスペースは未存在",
			tokenScopes:   readScopes,
			identifier:    "api-draft-rev-get-other",
			pageNumber:    1,
			revisionTitle: "p1-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIDraftPageRevisionFixture(t, tx, "api-draft-rev-get", apiTopicRegularMemberRole)
			setupAPIDraftPageRevisionFixture(t, tx, "api-draft-rev-get-other", apiTopicRegularMemberRole)
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}
			revisionID := model.DraftPageRevisionID("00000000-0000-0000-0000-000000000000")
			if tt.revisionTitle != "" {
				revisionID = f.revisionIDs[tt.revisionTitle]
			}

			output, err := newGetAPIDraftPageRevisionUC(tx).Execute(t.Context(), GetAPIDraftPageRevisionInput{
				Principal:       f.principal(apiTopicRegularMemberRole, tt.tokenScopes),
				SpaceIdentifier: identifier,
				PageNumber:      tt.pageNumber,
				RevisionID:      revisionID,
			})

			if !tt.wantFound {
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
			if output.Revision.ID != revisionID || output.Revision.Title != tt.revisionTitle || output.Revision.Body != "本文 "+tt.revisionTitle {
				t.Errorf("リビジョン = %+v、期待値 = %sのリビジョン", output.Revision, tt.revisionTitle)
			}
		})
	}
}
