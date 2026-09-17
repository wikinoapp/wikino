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

	t.Run("deleteスコープは含意を持たない", func(t *testing.T) {
		t.Parallel()

		result := expandScopes([]model.Scope{model.ScopeTopicDelete})
		assertScopes(t, result, []model.Scope{model.ScopeTopicDelete})
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

	t.Run("通常actionはread・write・deleteだけでadminは唯一の例外", func(t *testing.T) {
		t.Parallel()

		for _, scope := range expandScopes([]model.Scope{model.ScopeSpaceAdmin}) {
			if scope == model.ScopeSpaceAdmin {
				if scope.String() != "space:admin" {
					t.Errorf("特別スコープ = %q、期待値 = space:admin", scope)
				}
				continue
			}
			resource, action, ok := strings.Cut(scope.String(), ":")
			if !ok || resource == "" || !slices.Contains([]string{"read", "write", "delete"}, action) {
				t.Errorf("正式スコープの形式が不正: %q", scope)
			}
		}
	})

	t.Run("space:adminを含まない", func(t *testing.T) {
		t.Parallel()

		if slices.Contains(scopes, model.ScopeSpaceAdmin) {
			t.Error("allResourceScopes()にspace:adminが含まれている")
		}
	})

	t.Run("全リソースのスコープを含む", func(t *testing.T) {
		t.Parallel()

		expected := []model.Scope{
			model.ScopeSpaceRead, model.ScopeSpaceWrite, model.ScopeSpaceDelete,
			model.ScopeTopicRead, model.ScopeTopicWrite, model.ScopeTopicDelete,
			model.ScopeTopicMemberRead, model.ScopeTopicMemberWrite, model.ScopeTopicMemberDelete,
			model.ScopePageRead, model.ScopePageWrite,
			model.ScopePageTrashRead, model.ScopePageTrashWrite, model.ScopePageTrashDelete,
			model.ScopeDraftPageRead, model.ScopeDraftPageWrite, model.ScopeDraftPageDelete,
			model.ScopeSuggestionRead, model.ScopeSuggestionWrite, model.ScopeSuggestionApplicationWrite, model.ScopeSuggestionClosureWrite,
			model.ScopeSuggestionCommentRead, model.ScopeSuggestionCommentWrite,
			model.ScopeSpaceMemberRead, model.ScopeSpaceMemberWrite, model.ScopeSpaceMemberDelete,
			model.ScopeAttachmentRead, model.ScopeAttachmentWrite, model.ScopeAttachmentDelete,
		}

		for _, s := range expected {
			assertHasScope(t, scopes, s)
		}

		if len(scopes) != len(expected) {
			t.Errorf("len(allResourceScopes()) = %d、期待値 = %d", len(scopes), len(expected))
		}
	})
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

func TestExpandScopes_LegacyScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		legacy    model.Scope
		canonical model.Scope
	}{
		{model.ScopePageTrash, model.ScopePageTrashWrite},
		{model.ScopePageRestore, model.ScopePageTrashDelete},
		{model.ScopeSuggestionApply, model.ScopeSuggestionApplicationWrite},
		{model.ScopeSuggestionClose, model.ScopeSuggestionClosureWrite},
	}
	for _, tt := range tests {
		t.Run(tt.legacy.String(), func(t *testing.T) {
			t.Parallel()

			want := expandScopes([]model.Scope{tt.canonical})
			for _, input := range [][]model.Scope{{tt.legacy}, {tt.legacy, tt.canonical, tt.legacy}} {
				original := slices.Clone(input)
				assertScopes(t, expandScopes(input), want)
				if !slices.Equal(input, original) {
					t.Errorf("入力スコープが変更された: %v", input)
				}
			}
		})
	}
}

func TestExpandScopes_IndependentScopes(t *testing.T) {
	t.Parallel()

	for _, scope := range []model.Scope{
		model.ScopePageTrashRead, model.ScopePageTrashDelete,
		model.ScopeSuggestionApplicationWrite, model.ScopeSuggestionClosureWrite,
		"page:trash:write", "page:restore_extra", "suggestion:apply_extra", "suggestion:close_extra",
	} {
		t.Run(scope.String(), func(t *testing.T) {
			t.Parallel()
			assertScopes(t, expandScopes([]model.Scope{scope}), []model.Scope{scope})
		})
	}
}
