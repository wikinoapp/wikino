package policy

import (
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestExpandScopes(t *testing.T) {
	t.Parallel()

	t.Run("空のスコープは空を返す", func(t *testing.T) {
		t.Parallel()

		result := expandScopes(nil)
		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0", len(result))
		}

		result = expandScopes([]model.Scope{})
		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0", len(result))
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
			{"draft_page", model.ScopeDraftPageWrite, model.ScopeDraftPageRead},
			{"suggestion", model.ScopeSuggestionWrite, model.ScopeSuggestionRead},
			{"suggestion_comment", model.ScopeSuggestionCommentWrite, model.ScopeSuggestionCommentRead},
			{"space", model.ScopeSpaceWrite, model.ScopeSpaceRead},
			{"space_member", model.ScopeSpaceMemberWrite, model.ScopeSpaceMemberRead},
			{"attachment", model.ScopeAttachmentWrite, model.ScopeAttachmentRead},
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

	t.Run("ドメイン固有アクションは含意を持たない", func(t *testing.T) {
		t.Parallel()

		scopes := []model.Scope{model.ScopeSuggestionApply}
		result := expandScopes(scopes)
		assertScopes(t, result, []model.Scope{model.ScopeSuggestionApply})
	})

	t.Run("deleteスコープは含意を持たない", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeTopicDelete})
		assertScopes(t, result, []model.Scope{model.ScopeTopicDelete})
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

	t.Run("space:adminは全リソーススコープを包括する", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeSpaceAdmin})

		assertHasScope(t, result, model.ScopeSpaceAdmin)

		for _, s := range allResourceScopes() {
			assertHasScope(t, result, s)
		}

		assertNoDuplicates(t, result)
	})

	t.Run("space:adminと個別スコープの組み合わせでも重複しない", func(t *testing.T) {
		t.Parallel()

		input := []model.Scope{model.ScopeSpaceAdmin, model.ScopeTopicWrite}
		result := expandScopes(input)

		assertHasScope(t, result, model.ScopeSpaceAdmin)
		assertHasScope(t, result, model.ScopeTopicWrite)
		assertHasScope(t, result, model.ScopeTopicRead)
		assertNoDuplicates(t, result)
	})
}

func TestAllResourceScopes(t *testing.T) {
	t.Parallel()

	scopes := allResourceScopes()

	t.Run("space:adminを含まない", func(t *testing.T) {
		t.Parallel()

		if slices.Contains(scopes, model.ScopeSpaceAdmin) {
			t.Error("allResourceScopes() should not contain space:admin")
		}
	})

	t.Run("全リソースのスコープを含む", func(t *testing.T) {
		t.Parallel()

		expected := []model.Scope{
			model.ScopeSpaceRead, model.ScopeSpaceWrite, model.ScopeSpaceDelete,
			model.ScopeTopicRead, model.ScopeTopicWrite, model.ScopeTopicDelete,
			model.ScopeTopicMemberRead, model.ScopeTopicMemberWrite, model.ScopeTopicMemberDelete,
			model.ScopePageRead, model.ScopePageWrite, model.ScopePageTrash, model.ScopePageRestore,
			model.ScopeDraftPageRead, model.ScopeDraftPageWrite,
			model.ScopeSuggestionRead, model.ScopeSuggestionWrite, model.ScopeSuggestionApply, model.ScopeSuggestionClose,
			model.ScopeSuggestionCommentRead, model.ScopeSuggestionCommentWrite,
			model.ScopeSpaceMemberRead, model.ScopeSpaceMemberWrite, model.ScopeSpaceMemberDelete,
			model.ScopeAttachmentRead, model.ScopeAttachmentWrite, model.ScopeAttachmentDelete,
		}

		for _, s := range expected {
			assertHasScope(t, scopes, s)
		}

		if len(scopes) != len(expected) {
			t.Errorf("len(allResourceScopes()) = %d, want %d", len(scopes), len(expected))
		}
	})
}

func TestImplications(t *testing.T) {
	t.Parallel()

	t.Run("含意ルールはwrite→readのみ", func(t *testing.T) {
		t.Parallel()

		for upper, lowers := range implications {
			if len(lowers) != 1 {
				t.Errorf("implications[%s] has %d entries, want 1", upper, len(lowers))
			}
		}
	})
}

// assertHasScope は結果にスコープが含まれていることを検証する
func assertHasScope(t *testing.T, result []model.Scope, expected model.Scope) {
	t.Helper()
	if !slices.Contains(result, expected) {
		t.Errorf("result does not contain %s, got %v", expected, result)
	}
}

// assertScopes は結果が期待するスコープ集合と一致することを検証する（順序不問）
func assertScopes(t *testing.T, result, expected []model.Scope) {
	t.Helper()

	if len(result) != len(expected) {
		t.Errorf("len(result) = %d, want %d; result = %v", len(result), len(expected), result)
		return
	}
	for _, s := range expected {
		assertHasScope(t, result, s)
	}
}

// assertNoDuplicates は結果に重複がないことを検証する
func assertNoDuplicates(t *testing.T, scopes []model.Scope) {
	t.Helper()

	seen := make(map[model.Scope]bool, len(scopes))
	for _, s := range scopes {
		if seen[s] {
			t.Errorf("duplicate scope found: %s", s)
		}
		seen[s] = true
	}
}
