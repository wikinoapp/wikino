package usecase

import (
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func newGetAPITopicUC(tx *sql.Tx) *GetAPITopicUsecase {
	q := testutil.QueriesWithTx(tx)
	return NewGetAPITopicUsecase(repository.NewTopicRepository(q), repository.NewTopicMemberRepository(q))
}

func TestGetAPITopicUsecase_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		memberRole  model.SpaceRole
		tokenScopes []model.Scope
		identifier  model.SpaceIdentifier
		topicNumber int32
		wantFound   bool
	}{
		{
			name:        "公開トピックを返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			topicNumber: 1,
			wantFound:   true,
		},
		{
			name:        "トークンがtopic:readを持たなくても公開トピックは返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageRead},
			topicNumber: 1,
			wantFound:   true,
		},
		{
			name:        "参加している非公開トピックを、トークンがtopic:readを持てば返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			topicNumber: 2,
			wantFound:   true,
		},
		{
			name:        "参加している非公開トピックも、トークンがtopic:readを持たなければ未存在",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopePageWrite},
			topicNumber: 2,
		},
		{
			// どのスペースのロールもtopic:readを持つため、参加していない非公開トピックも開ける
			name:        "一般メンバーはトークンがtopic:readを持てば参加していない非公開トピックも返す",
			memberRole:  apiTopicRegularMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			topicNumber: 3,
			wantFound:   true,
		},
		{
			name:        "管理者はトークンがtopic:readを持てば参加していない非公開トピックも返す",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			topicNumber: 3,
			wantFound:   true,
		},
		{
			name:        "廃棄済みのトピックは未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			topicNumber: 4,
		},
		{
			name:        "存在しない番号は未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			topicNumber: 99,
		},
		{
			name:        "束縛先と異なるスペースは未存在",
			memberRole:  apiTopicAdminMemberRole,
			tokenScopes: []model.Scope{model.ScopeTopicRead},
			identifier:  "api-topic-get-other",
			topicNumber: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupAPITopicFixture(t, tx, "api-topic-get", tt.memberRole)
			setupAPITopicFixture(t, tx, "api-topic-get-other", tt.memberRole)
			identifier := f.space.Identifier
			if tt.identifier != "" {
				identifier = tt.identifier
			}

			output, err := newGetAPITopicUC(tx).Execute(t.Context(), GetAPITopicInput{
				Principal:       f.principal(tt.memberRole, tt.tokenScopes),
				SpaceIdentifier: identifier,
				TopicNumber:     tt.topicNumber,
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
			if output.Topic.Number != tt.topicNumber {
				t.Errorf("Number = %d、期待値 = %d", output.Topic.Number, tt.topicNumber)
			}
		})
	}
}
