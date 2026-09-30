package usecase

import (
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetAPIPageUC(tx *sql.Tx) *GetAPIPageUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewGetAPIPageUsecase(repository.NewPageRepository(q), repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

func TestGetAPIPageUsecase_Execute(t *testing.T) {
	t.Parallel()

	readScopes := []model.Scope{model.ScopePageRead, model.ScopeTopicRead}
	tests := []struct {
		name         string
		memberScopes []model.Scope
		tokenScopes  []model.Scope
		identifier   model.SpaceIdentifier
		pageNumber   int32
		wantFound    bool
	}{
		{
			name:         "公開トピックのページを返す",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  []model.Scope{model.ScopePageRead},
			pageNumber:   1,
			wantFound:    true,
		},
		{
			name:         "トークンがtopic:readを持てば参加している非公開トピックのページを返す",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  readScopes,
			pageNumber:   6,
			wantFound:    true,
		},
		{
			name:         "トークンがtopic:readを持たなければ参加している非公開トピックのページは未存在",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  []model.Scope{model.ScopePageRead},
			pageNumber:   6,
		},
		{
			name:         "一般メンバーは参加していない非公開トピックのページを開けない",
			memberScopes: apiTopicRegularMemberScopes,
			tokenScopes:  readScopes,
			pageNumber:   7,
		},
		{
			name:         "管理者はトークンがtopic:readを持てば参加していない非公開トピックのページも返す",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  readScopes,
			pageNumber:   7,
			wantFound:    true,
		},
		{
			name:         "未公開のページは未存在",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  readScopes,
			pageNumber:   3,
		},
		{
			name:         "ゴミ箱のページはゴミ箱を開けるメンバーにも未存在",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  readScopes,
			pageNumber:   4,
		},
		{
			name:         "廃棄済みのページは未存在",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  readScopes,
			pageNumber:   5,
		},
		{
			name:         "廃棄済みのトピックのページは未存在",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  readScopes,
			pageNumber:   8,
		},
		{
			name:         "存在しない番号は未存在",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  readScopes,
			pageNumber:   99,
		},
		{
			name:         "束縛先と異なるスペースは未存在",
			memberScopes: apiTopicAdminMemberScopes,
			tokenScopes:  readScopes,
			identifier:   "api-page-get-other",
			pageNumber:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPIPageFixture(t, tx, "api-page-get", tt.memberScopes)
			setupAPIPageFixture(t, tx, "api-page-get-other", tt.memberScopes)
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}

			output, err := newGetAPIPageUC(tx).Execute(t.Context(), GetAPIPageInput{
				Principal:       f.principal(tt.memberScopes, tt.tokenScopes),
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
				t.Fatalf("Execute() error = %v", err)
			}
			if output.Page.Number != model.PageNumber(tt.pageNumber) {
				t.Errorf("Number = %d、期待値 = %d", output.Page.Number, tt.pageNumber)
			}
			if output.Topic.ID != output.Page.TopicID {
				t.Errorf("Topic.ID = %s、期待値 = ページの所属トピック %s", output.Topic.ID, output.Page.TopicID)
			}
		})
	}
}
