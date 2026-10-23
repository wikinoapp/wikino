package usecase

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetAPIDraftPageUC(tx *sql.Tx) *GetAPIDraftPageUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewGetAPIDraftPageUsecase(repository.NewPageRepository(q), repository.NewDraftPageRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

func TestGetAPIDraftPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	readScopes := []model.Scope{model.ScopeDraftPageRead, model.ScopeTopicRead}
	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		identifier  model.SpaceIdentifier
		pageNumber  int32
		wantFound   bool
	}{
		{
			name:        "公開済みのページの下書きを返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeDraftPageRead},
			pageNumber:  1,
			wantFound:   true,
		},
		{
			name:        "未公開のページ (新しいページ) の下書きを返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeDraftPageRead},
			pageNumber:  2,
			wantFound:   true,
		},
		{
			name:        "ほかのメンバーの下書きは未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: readScopes,
			pageNumber:  3,
		},
		{
			name:        "ゴミ箱のページの下書きは未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: readScopes,
			pageNumber:  4,
		},
		{
			name:        "トークンがtopic:readを持てば非公開トピックの下書きを返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: readScopes,
			pageNumber:  6,
			wantFound:   true,
		},
		{
			name:        "トークンがtopic:readを持たなければ非公開トピックの下書きは未存在",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeDraftPageRead},
			pageNumber:  5,
		},
		{
			// 閲覧者はスペースのロールではdraft_page:readを持たないが、トピックの編集者として参加しているトピックでは持つ
			name:        "メンバーが下書きを読めるトピックの下書きを返す",
			memberRole:  model.SpaceRoleViewer,
			tokenScopes: readScopes,
			pageNumber:  5,
			wantFound:   true,
		},
		{
			name:        "メンバーが下書きを読めないトピックの下書きは未存在",
			memberRole:  model.SpaceRoleViewer,
			tokenScopes: readScopes,
			pageNumber:  1,
		},
		{
			name:        "トークンがdraft_page:readを持たなければ未存在",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite},
			pageNumber:  1,
		},
		{
			name:        "廃棄済みのトピックの下書きは未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: readScopes,
			pageNumber:  7,
		},
		{
			name:        "下書きの無いページは未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: readScopes,
			pageNumber:  8,
		},
		{
			name:        "存在しない番号は未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: readScopes,
			pageNumber:  99,
		},
		{
			name:        "トークンが属さないスペースは未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: readScopes,
			identifier:  "api-draft-get-other",
			pageNumber:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIDraftPageFixture(t, tx, "api-draft-get", tt.memberRole)
			setupAPIDraftPageFixture(t, tx, "api-draft-get-other", tt.memberRole)
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}

			output, err := newGetAPIDraftPageUC(tx).Execute(t.Context(), GetAPIDraftPageInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: identifier,
				PageNumber:      tt.pageNumber,
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
			draft := output.DraftPage
			if draft.Page == nil || draft.Page.Number != model.PageNumber(tt.pageNumber) {
				t.Errorf("Page = %+v、期待値 = ページ%d", draft.Page, tt.pageNumber)
			}
			if draft.SpaceMemberID != f.spaceMemberID {
				t.Errorf("SpaceMemberID = %s、期待値 = %s", draft.SpaceMemberID, f.spaceMemberID)
			}
			if want := fmt.Sprintf("下書きの本文%d", tt.pageNumber); draft.Body != want {
				t.Errorf("Body = %q、期待値 = %q", draft.Body, want)
			}
			if output.Topic.ID != draft.TopicID {
				t.Errorf("Topic.ID = %s、期待値 = 下書きの所属トピック %s", output.Topic.ID, draft.TopicID)
			}
		})
	}
}
