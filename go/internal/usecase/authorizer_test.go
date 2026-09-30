package usecase

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
)

func TestNewAuthorizer_Roles(t *testing.T) {
	t.Parallel()

	privateTopic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

	type want struct {
		showPrivateTopic bool
		createTopic      bool
		updateVisibility bool
		createPage       bool
		trashPage        bool
		applySuggestion  bool
		createSuggestion bool
		updateSpace      bool
		createPAT        bool
		showOAuthApps    bool
		createComment    bool
		deleteDraft      bool
	}

	tests := []struct {
		name      string
		spaceRole model.SpaceRole
		topicRole model.TopicRole
		want      want
	}{
		{
			name:      "管理者はすべての操作ができる",
			spaceRole: model.SpaceRoleAdmin,
			want: want{
				showPrivateTopic: true, createTopic: true, updateVisibility: true, createPage: true, trashPage: true,
				applySuggestion: true, createSuggestion: true, updateSpace: true, createPAT: true, showOAuthApps: true,
				createComment: true, deleteDraft: true,
			},
		},
		{
			name:      "編集者はトピックとページを書けるが、公開範囲とスペースの設定は変えられない",
			spaceRole: model.SpaceRoleEditor,
			want: want{
				showPrivateTopic: true, createTopic: true, createPage: true, trashPage: true,
				applySuggestion: true, createSuggestion: true, createPAT: true,
				createComment: true, deleteDraft: true,
			},
		},
		{
			name:      "閲覧者はページを書けないが、編集提案とコメントは書ける",
			spaceRole: model.SpaceRoleViewer,
			want: want{
				showPrivateTopic: true, createSuggestion: true, createPAT: true, createComment: true,
			},
		},
		{
			name:      "閲覧者にトピック編集者を付けると、そのトピックのページを書ける",
			spaceRole: model.SpaceRoleViewer,
			topicRole: model.TopicRoleEditor,
			want: want{
				showPrivateTopic: true, createPage: true, trashPage: true, applySuggestion: true,
				createSuggestion: true, createPAT: true, createComment: true, deleteDraft: true,
			},
		},
		{
			name:      "編集者にトピック管理者を付けると、そのトピックの公開範囲を変えられる",
			spaceRole: model.SpaceRoleEditor,
			topicRole: model.TopicRoleAdmin,
			want: want{
				showPrivateTopic: true, createTopic: true, updateVisibility: true, createPage: true, trashPage: true,
				applySuggestion: true, createSuggestion: true, createPAT: true,
				createComment: true, deleteDraft: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spaceMember := &model.SpaceMember{Role: tt.spaceRole}
			var topicMember *model.TopicMember
			if tt.topicRole != "" {
				topicMember = &model.TopicMember{Role: tt.topicRole}
			}
			a := newAuthorizer(spaceMember, topicMember)

			got := want{
				showPrivateTopic: a.CanShowTopic(privateTopic),
				createTopic:      a.CanCreateTopic(),
				updateVisibility: a.CanUpdateTopicVisibility(),
				createPage:       a.CanCreatePage(),
				trashPage:        a.CanTrashPage(),
				applySuggestion:  a.CanApplySuggestion(),
				createSuggestion: a.CanCreateSuggestion(privateTopic),
				updateSpace:      a.CanUpdateSpace(),
				createPAT:        a.CanCreatePersonalAccessToken(),
				showOAuthApps:    a.CanShowOAuthApplications(),
				createComment:    a.CanCreateSuggestionComment(),
				deleteDraft:      a.CanDeleteDraftPage(),
			}
			if got != tt.want {
				t.Errorf("判定が期待と異なる\n得られた値: %+v\n期待した値: %+v", got, tt.want)
			}
		})
	}
}

func TestNewAuthorizer_RoleIsEmpty(t *testing.T) {
	t.Parallel()

	// ロールが空のスペースメンバー (NOT NULLにする前に残りうる行) は、何の権限も持たない
	a := newAuthorizer(&model.SpaceMember{}, &model.TopicMember{})
	if a.CanShowTopic(&model.Topic{Visibility: model.TopicVisibilityPrivate}) || a.CanCreatePage() {
		t.Error("ロールが空のメンバーに権限があってはならない")
	}
}

func TestNewAuthorizer_NonMemberIsGuest(t *testing.T) {
	t.Parallel()

	if _, ok := newAuthorizer(nil, nil).(*policy.GuestPolicy); !ok {
		t.Error("スペースメンバーでなければGuestPolicyを返すことを期待した")
	}
}

func TestNewAPIAuthorizer_IntersectsTokenScopesWithRole(t *testing.T) {
	t.Parallel()

	privateTopic := &model.Topic{Visibility: model.TopicVisibilityPrivate}
	tokenScopes := policy.ExpandAPITokenScopes([]model.Scope{model.ScopePageWrite, model.ScopeTopicRead})

	t.Run("閲覧者はpage:writeのトークンでもページを書けない", func(t *testing.T) {
		t.Parallel()

		a := newAPIAuthorizer(&model.APIPrincipal{
			SpaceMember: &model.SpaceMember{Role: model.SpaceRoleViewer},
			Scopes:      tokenScopes,
		}, nil)
		if a.CanCreatePage() {
			t.Error("閲覧者のロールに無いpage:writeがトークンで有効になってはならない")
		}
		if !a.CanShowTopic(privateTopic) {
			t.Error("閲覧者のtopic:readとトークンのtopic:readの積で非公開トピックを閲覧できるべき")
		}
	})

	t.Run("トピック編集者を付けた閲覧者は、そのトピックでページを書ける", func(t *testing.T) {
		t.Parallel()

		a := newAPIAuthorizer(&model.APIPrincipal{
			SpaceMember: &model.SpaceMember{Role: model.SpaceRoleViewer},
			Scopes:      tokenScopes,
		}, &model.TopicMember{Role: model.TopicRoleEditor})
		if !a.CanCreatePage() {
			t.Error("トピック編集者のpage:writeとトークンのpage:writeの積でページを書けるべき")
		}
	})

	t.Run("管理者でもトークンに無いスコープは使えない", func(t *testing.T) {
		t.Parallel()

		a := newAPIAuthorizer(&model.APIPrincipal{
			SpaceMember: &model.SpaceMember{Role: model.SpaceRoleAdmin},
			Scopes:      tokenScopes,
		}, nil)
		if a.CanUpdateTopic() || a.CanUpdateSpace() {
			t.Error("トークンが持たないスコープが有効になってはならない")
		}
	})
}
