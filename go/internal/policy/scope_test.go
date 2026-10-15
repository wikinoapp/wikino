package policy

import (
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestExpandScopes(t *testing.T) {
	t.Parallel()

	t.Run("空のスコープは空を返す", func(t *testing.T) {
		t.Parallel()

		result := expandScopes(nil)
		if len(result) != 0 {
			t.Errorf("len(result) = %d、期待値 = 0", len(result))
		}

		result = expandScopes([]model.Scope{})
		if len(result) != 0 {
			t.Errorf("len(result) = %d、期待値 = 0", len(result))
		}
	})

	t.Run("readスコープは展開されない", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeTopicRead})
		assertScopes(t, result, []model.Scope{model.ScopeTopicRead})
	})

	t.Run("writeスコープはreadを含意する", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			input model.Scope
			want  model.Scope
		}{
			{"topic", model.ScopeTopicWrite, model.ScopeTopicRead},
			{"topic_member", model.ScopeTopicMemberWrite, model.ScopeTopicMemberRead},
			{"page", model.ScopePageWrite, model.ScopePageRead},
			{"page_trash", model.ScopePageTrashWrite, model.ScopePageTrashRead},
			{"draft_page", model.ScopeDraftPageWrite, model.ScopeDraftPageRead},
			{"suggestion", model.ScopeSuggestionWrite, model.ScopeSuggestionRead},
			{"suggestion_comment", model.ScopeSuggestionCommentWrite, model.ScopeSuggestionCommentRead},
			{"space", model.ScopeSpaceWrite, model.ScopeSpaceRead},
			{"space_member", model.ScopeSpaceMemberWrite, model.ScopeSpaceMemberRead},
			{"attachment", model.ScopeAttachmentWrite, model.ScopeAttachmentRead},
			{"personal_access_token", model.ScopePersonalAccessTokenWrite, model.ScopePersonalAccessTokenRead},
			{"oauth_grant", model.ScopeOAuthGrantWrite, model.ScopeOAuthGrantRead},
			{"oauth_application", model.ScopeOAuthApplicationWrite, model.ScopeOAuthApplicationRead},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result := expandScopes([]model.Scope{tt.input})
				assertHasScope(t, result, tt.input)
				assertHasScope(t, result, tt.want)
			})
		}
	})

	t.Run("編集提案の反映は含意を持たない", func(t *testing.T) {
		t.Parallel()

		scopes := []model.Scope{model.ScopeSuggestionApplicationWrite}
		result := expandScopes(scopes)
		assertScopes(t, result, []model.Scope{model.ScopeSuggestionApplicationWrite})
	})

	t.Run("topic:deleteは含意を持たない", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeTopicDelete})
		assertScopes(t, result, []model.Scope{model.ScopeTopicDelete})
	})

	t.Run("personal_access_token:deleteはreadを含意する", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopePersonalAccessTokenDelete})
		assertScopes(t, result, []model.Scope{model.ScopePersonalAccessTokenDelete, model.ScopePersonalAccessTokenRead})
	})

	t.Run("oauth_grant:deleteはreadを含意する", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeOAuthGrantDelete})
		assertScopes(t, result, []model.Scope{model.ScopeOAuthGrantDelete, model.ScopeOAuthGrantRead})
	})

	t.Run("oauth_application:deleteはreadを含意する", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeOAuthApplicationDelete})
		assertScopes(t, result, []model.Scope{model.ScopeOAuthApplicationDelete, model.ScopeOAuthApplicationRead})
	})

	t.Run("draft_page:deleteは含意を持たない", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeDraftPageDelete})
		assertScopes(t, result, []model.Scope{model.ScopeDraftPageDelete})
	})

	t.Run("draft_page:writeはdraft_page:deleteを含意しない", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeDraftPageWrite})
		assertHasScope(t, result, model.ScopeDraftPageWrite)
		assertHasScope(t, result, model.ScopeDraftPageRead)
		if slices.Contains(result, model.ScopeDraftPageDelete) {
			t.Error("draft_page:writeがdraft_page:deleteを含んでいる")
		}
	})

	t.Run("複数のwriteスコープがそれぞれreadを含意する", func(t *testing.T) {
		t.Parallel()

		input := []model.Scope{model.ScopeTopicWrite, model.ScopePageWrite}
		result := expandScopes(input)

		assertHasScope(t, result, model.ScopeTopicWrite)
		assertHasScope(t, result, model.ScopeTopicRead)
		assertHasScope(t, result, model.ScopePageWrite)
		assertHasScope(t, result, model.ScopePageRead)
	})

	t.Run("重複するスコープは除去される", func(t *testing.T) {
		t.Parallel()

		input := []model.Scope{model.ScopeTopicWrite, model.ScopeTopicRead}
		result := expandScopes(input)

		assertHasScope(t, result, model.ScopeTopicWrite)
		assertHasScope(t, result, model.ScopeTopicRead)
		assertNoDuplicates(t, result)
	})

	t.Run("space:adminは展開しない", func(t *testing.T) {
		t.Parallel()

		// 権限はロールから引くため、space:adminはRails版向けに保存する値でしかない
		result := expandScopes([]model.Scope{model.ScopeSpaceAdmin, model.ScopeTopicWrite})

		assertScopes(t, result, []model.Scope{model.ScopeSpaceAdmin, model.ScopeTopicWrite, model.ScopeTopicRead})
	})
}

func TestScopeDefinitions_Actions(t *testing.T) {
	t.Parallel()

	for _, d := range model.ScopeDefinitions {
		resource, action, ok := strings.Cut(d.Scope.String(), ":")
		if !ok || resource == "" || !slices.Contains([]string{"read", "write", "delete"}, action) {
			t.Errorf("スコープの形式が不正: %q", d.Scope)
		}
	}
}

func TestImplications(t *testing.T) {
	t.Parallel()

	t.Run("含意ルールはwrite→readのみ", func(t *testing.T) {
		t.Parallel()

		for upper, lowers := range implications {
			if len(lowers) != 1 {
				t.Errorf("implications[%s]の件数 = %d、期待値 = 1", upper, len(lowers))
			}
		}
	})
}

// assertHasScopeは結果にスコープが含まれていることを検証する
func assertHasScope(t *testing.T, result []model.Scope, expected model.Scope) {
	t.Helper()
	if !slices.Contains(result, expected) {
		t.Errorf("結果に%sが含まれていない: %v", expected, result)
	}
}

// assertScopesは結果が期待するスコープ集合と一致することを検証する (順序不問)
func assertScopes(t *testing.T, result, expected []model.Scope) {
	t.Helper()

	if len(result) != len(expected) {
		t.Errorf("len(result) = %d、期待値 = %d: result = %v", len(result), len(expected), result)
		return
	}
	for _, s := range expected {
		assertHasScope(t, result, s)
	}
}

// assertNoDuplicatesは結果に重複がないことを検証する
func assertNoDuplicates(t *testing.T, scopes []model.Scope) {
	t.Helper()

	seen := make(map[model.Scope]bool, len(scopes))
	for _, s := range scopes {
		if seen[s] {
			t.Errorf("スコープが重複している: %s", s)
		}
		seen[s] = true
	}
}

func TestExpandScopes_IndependentScopes(t *testing.T) {
	t.Parallel()

	for _, scope := range []model.Scope{
		model.ScopePageTrashRead, model.ScopePageTrashDelete,
		model.ScopePersonalAccessTokenRead,
		model.ScopeOAuthGrantRead,
		model.ScopeOAuthApplicationRead,
		model.ScopeSuggestionApplicationWrite, model.ScopeSuggestionClosureWrite,
		"unknown:write", "page_trash:admin", "suggestion_application:read", "suggestion_closure:delete",
	} {
		t.Run(scope.String(), func(t *testing.T) {
			t.Parallel()
			assertScopes(t, expandScopes([]model.Scope{scope}), []model.Scope{scope})
		})
	}
}

func TestExpandAPITokenScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []model.Scope
		expected []model.Scope
	}{
		{name: "スコープなし", input: nil, expected: []model.Scope{}},
		{name: "topic:read", input: []model.Scope{model.ScopeTopicRead}, expected: []model.Scope{model.ScopeTopicRead}},
		{name: "page:writeはpage:readを含意する", input: []model.Scope{model.ScopePageWrite}, expected: []model.Scope{model.ScopePageWrite, model.ScopePageRead}},
		{name: "draft_page:writeはdraft_page:readを含意する", input: []model.Scope{model.ScopeDraftPageWrite}, expected: []model.Scope{model.ScopeDraftPageWrite, model.ScopeDraftPageRead}},
		{name: "draft_page:delete", input: []model.Scope{model.ScopeDraftPageDelete}, expected: []model.Scope{model.ScopeDraftPageDelete}},
		{name: "attachment:read", input: []model.Scope{model.ScopeAttachmentRead}, expected: []model.Scope{model.ScopeAttachmentRead}},
		{name: "トークンに付与できないattachment:writeは落とし、含意のattachment:readも付けない", input: []model.Scope{model.ScopeAttachmentWrite}, expected: []model.Scope{}},
		{
			name:     "トークンに付与できないスコープは落とす",
			input:    []model.Scope{model.ScopePageRead, model.ScopeTopicWrite, model.ScopePersonalAccessTokenWrite, model.ScopeOAuthApplicationWrite},
			expected: []model.Scope{model.ScopePageRead},
		},
		{
			name:     "space:adminだけが保存されていても権限は生まれない",
			input:    []model.Scope{model.ScopeSpaceAdmin},
			expected: []model.Scope{},
		},
		{
			name:     "space:adminとpage:writeが保存されていてもページの権限だけを展開する",
			input:    []model.Scope{model.ScopeSpaceAdmin, model.ScopePageWrite},
			expected: []model.Scope{model.ScopePageWrite, model.ScopePageRead},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertScopes(t, ExpandAPITokenScopes(tt.input), tt.expected)
		})
	}
}

func TestTopicRoleScopes(t *testing.T) {
	t.Parallel()

	t.Run("トピックメンバーでなければnil", func(t *testing.T) {
		t.Parallel()

		spaceMember := &model.SpaceMember{Role: model.SpaceRoleViewer, MaxTopicRole: new(model.TopicRoleViewer)}
		if got := TopicRoleScopes(spaceMember, nil); got != nil {
			t.Errorf("TopicRoleScopes() = %v、期待値 = nil", got)
		}
	})

	t.Run("上限が無ければトピックのロールのスコープをそのまま返す", func(t *testing.T) {
		t.Parallel()

		for _, spaceMember := range []*model.SpaceMember{{Role: model.SpaceRoleViewer}, nil} {
			got := TopicRoleScopes(spaceMember, &model.TopicMember{Role: model.TopicRoleAdmin})
			assertScopes(t, got, model.TopicRoleAdmin.Scopes())
		}
	})

	tests := []struct {
		name      string
		topicRole model.TopicRole
		maxRole   model.TopicRole
		expected  model.TopicRole
	}{
		{name: "上限がトピック編集者なら、トピック管理者を付けてもトピック編集者のスコープだけが効く", topicRole: model.TopicRoleAdmin, maxRole: model.TopicRoleEditor, expected: model.TopicRoleEditor},
		{name: "上限がトピック閲覧者なら、トピック管理者を付けてもトピック閲覧者のスコープだけが効く", topicRole: model.TopicRoleAdmin, maxRole: model.TopicRoleViewer, expected: model.TopicRoleViewer},
		{name: "上限がトピック閲覧者なら、トピック編集者を付けてもトピック閲覧者のスコープだけが効く", topicRole: model.TopicRoleEditor, maxRole: model.TopicRoleViewer, expected: model.TopicRoleViewer},
		// トピック編集者はpage:readを持たずpage:writeを持つため、上限の側を展開しないとトピック閲覧者のpage:readが落ちる
		{name: "上限の含意を展開してから積を取る", topicRole: model.TopicRoleViewer, maxRole: model.TopicRoleEditor, expected: model.TopicRoleViewer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spaceMember := &model.SpaceMember{Role: model.SpaceRoleViewer, MaxTopicRole: new(tt.maxRole)}
			got := TopicRoleScopes(spaceMember, &model.TopicMember{Role: tt.topicRole})
			assertScopes(t, got, expandScopes(tt.expected.Scopes()))
		})
	}
}

func TestHeldAPITokenScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spaceMember  *model.SpaceMember
		topicMembers []*model.TopicMember
		expected     []model.Scope
	}{
		{
			name:        "スペース編集者はトークンに付与できるすべてのスコープを持つ",
			spaceMember: &model.SpaceMember{Role: model.SpaceRoleEditor},
			expected:    model.APITokenScopes,
		},
		{
			name:        "スペース閲覧者は読み取りのスコープだけを持つ",
			spaceMember: &model.SpaceMember{Role: model.SpaceRoleViewer},
			expected:    []model.Scope{model.ScopeTopicRead, model.ScopePageRead, model.ScopeAttachmentRead},
		},
		{
			name:         "スペース閲覧者でも、トピック編集者のロールがあればそのスコープを持つ",
			spaceMember:  &model.SpaceMember{Role: model.SpaceRoleViewer},
			topicMembers: []*model.TopicMember{{Role: model.TopicRoleViewer}, {Role: model.TopicRoleEditor}},
			expected:     model.APITokenScopes,
		},
		{
			name:         "トピックのロールの上限で削られたスコープは持たない",
			spaceMember:  &model.SpaceMember{Role: model.SpaceRoleViewer, MaxTopicRole: new(model.TopicRoleViewer)},
			topicMembers: []*model.TopicMember{{Role: model.TopicRoleEditor}},
			expected:     []model.Scope{model.ScopeTopicRead, model.ScopePageRead, model.ScopeAttachmentRead},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := HeldAPITokenScopes(tt.spaceMember, tt.topicMembers)
			if !slices.Equal(got, tt.expected) {
				t.Errorf("HeldAPITokenScopes() = %v、期待値 = %v", got, tt.expected)
			}
		})
	}
}
