package model

import "testing"

func TestScope_StringsToScopes(t *testing.T) {
	t.Parallel()

	// 保存値の変換では旧名を保持し、読み替えはPolicyに委ねる。
	tests := []struct {
		value string
		want  Scope
	}{
		{"page_trash:read", ScopePageTrashRead},
		{"page_trash:write", ScopePageTrashWrite},
		{"page_trash:delete", ScopePageTrashDelete},
		{"suggestion_application:write", ScopeSuggestionApplicationWrite},
		{"suggestion_closure:write", ScopeSuggestionClosureWrite},
		{"page:trash", ScopePageTrash},
		{"page:restore", ScopePageRestore},
		{"suggestion:apply", ScopeSuggestionApply},
		{"suggestion:close", ScopeSuggestionClose},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			scopes := StringsToScopes([]string{tt.value})
			if len(scopes) != 1 || scopes[0] != tt.want {
				t.Fatalf("変換結果 = %v、期待値 = [%s]", scopes, tt.want)
			}
			if got := scopes[0].String(); got != tt.value {
				t.Errorf("保存値 = %q、期待値 = %q", got, tt.value)
			}
		})
	}
}
