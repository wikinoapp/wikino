package model_test

import (
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// adminOnlyScopesは、管理者のロールだけが持ち、ほかのどのロールにも含めないスコープと、その理由。
// 定義の表にスコープを足したときに、管理者以外のロールへ含めるかどうかを決め忘れないよう、
// どのロールにも含めないスコープはここに理由と一緒に書く
var adminOnlyScopes = map[model.Scope]string{
	model.ScopeSpaceRead:              "スペースの設定はスペースの管理者だけが扱う",
	model.ScopeSpaceWrite:             "スペースの設定の変更とエクスポートはスペースの管理者だけが行う",
	model.ScopeSpaceDelete:            "スペースの削除はスペースの管理者だけが行う",
	model.ScopeSpaceMemberWrite:       "スペースのメンバーの管理はスペースの管理者だけが行う",
	model.ScopeSpaceMemberDelete:      "スペースのメンバーの管理はスペースの管理者だけが行う",
	model.ScopeAttachmentDelete:       "スペースに属する添付ファイルの削除はスペースの管理者だけが行う",
	model.ScopeOAuthApplicationRead:   "OAuthアプリの登録と管理はスペースの管理者だけが行う",
	model.ScopeOAuthApplicationWrite:  "OAuthアプリの登録と管理はスペースの管理者だけが行う",
	model.ScopeOAuthApplicationDelete: "OAuthアプリの登録と管理はスペースの管理者だけが行う",
}

// expandは、定義の表の含意に従ってスコープを展開する。
// 含意は同じリソースの中の1段だけのため、1回たどれば足りる
func expand(scopes []model.Scope) []model.Scope {
	expanded := slices.Clone(scopes)
	for _, d := range model.ScopeDefinitions {
		if model.HasScope(scopes, d.Scope) {
			for _, implied := range d.Implies {
				if !model.HasScope(expanded, implied) {
					expanded = append(expanded, implied)
				}
			}
		}
	}
	return expanded
}

// roleScopesは、検査の対象にするロールの名前とスコープの組
type roleScopes struct {
	name   string
	scopes []model.Scope
}

func allRoleScopes() []roleScopes {
	var roles []roleScopes
	for _, r := range model.SpaceRoles {
		roles = append(roles, roleScopes{name: "スペースの" + string(r), scopes: r.Scopes()})
	}
	for _, r := range model.TopicRoles {
		roles = append(roles, roleScopes{name: "トピックの" + string(r), scopes: r.Scopes()})
	}
	return roles
}

func scopeDefinition(t *testing.T, scope model.Scope) (model.ScopeDefinition, bool) {
	t.Helper()

	i := slices.IndexFunc(model.ScopeDefinitions, func(d model.ScopeDefinition) bool { return d.Scope == scope })
	if i < 0 {
		return model.ScopeDefinition{}, false
	}
	return model.ScopeDefinitions[i], true
}

func TestRoles(t *testing.T) {
	t.Parallel()

	t.Run("定義の表のスコープは管理者以外のロールに含めるか、管理者だけのスコープとして理由が書かれている", func(t *testing.T) {
		t.Parallel()

		var covered []model.Scope
		for _, role := range allRoleScopes() {
			if role.name == "スペースの"+string(model.SpaceRoleAdmin) {
				continue
			}
			covered = append(covered, expand(role.scopes)...)
		}

		for _, d := range model.ScopeDefinitions {
			_, adminOnly := adminOnlyScopes[d.Scope]
			switch {
			case model.HasScope(covered, d.Scope) && adminOnly:
				t.Errorf("%sは管理者以外のロールに含まれているが、管理者だけのスコープにも書かれている", d.Scope)
			case !model.HasScope(covered, d.Scope) && !adminOnly:
				t.Errorf("%sを管理者以外のロールに含めるかが決まっていない。ロールに含めるか、adminOnlyScopesに理由を書く", d.Scope)
			}
		}
		for s := range adminOnlyScopes {
			if _, ok := scopeDefinition(t, s); !ok {
				t.Errorf("管理者だけのスコープ%sが定義の表に無い", s)
			}
		}
	})

	t.Run("ロールのスコープは定義の表にあり重複しない", func(t *testing.T) {
		t.Parallel()

		for _, role := range allRoleScopes() {
			if len(role.scopes) == 0 {
				t.Errorf("%sのスコープが空", role.name)
			}
			for i, s := range role.scopes {
				if _, ok := scopeDefinition(t, s); !ok {
					t.Errorf("%sのスコープ%sが定義の表に無い", role.name, s)
				}
				if slices.Contains(role.scopes[:i], s) {
					t.Errorf("%sのスコープ%sが重複している", role.name, s)
				}
			}
		}
	})

	t.Run("スペースの管理者は定義の表のすべてのスコープを持つ", func(t *testing.T) {
		t.Parallel()

		got := model.SpaceRoleAdmin.Scopes()
		if len(got) != len(model.ScopeDefinitions) {
			t.Fatalf("スコープの数 = %d、期待値 = %d", len(got), len(model.ScopeDefinitions))
		}
		for i, d := range model.ScopeDefinitions {
			if got[i] != d.Scope {
				t.Errorf("%d番目のスコープ = %s、期待値 = %s", i, got[i], d.Scope)
			}
		}
		if model.HasScope(got, model.ScopeSpaceAdmin) {
			t.Errorf("%sを含んでいる", model.ScopeSpaceAdmin)
		}
	})

	t.Run("スペース用のロールはスペースにしか付けられない", func(t *testing.T) {
		t.Parallel()

		for _, r := range model.SpaceRoles {
			if got := r.LowestLevel(); got != model.ScopeLevelSpace {
				t.Errorf("%sの付けられる最も下の単位 = %d、期待値 = %d", r, got, model.ScopeLevelSpace)
			}
		}
	})

	t.Run("トピック用のロールはトピックまで付けられるスコープだけで作る", func(t *testing.T) {
		t.Parallel()

		for _, r := range model.TopicRoles {
			for _, s := range r.Scopes() {
				if d, ok := scopeDefinition(t, s); ok && d.LowestLevel != model.ScopeLevelTopic {
					t.Errorf("%sがスペースにしか付けられない%sを含んでいる", r, s)
				}
			}
			if got := r.LowestLevel(); got != model.ScopeLevelTopic {
				t.Errorf("%sの付けられる最も下の単位 = %d、期待値 = %d", r, got, model.ScopeLevelTopic)
			}
		}
	})

	t.Run("権限の広いロールは狭いロールのスコープをすべて含む", func(t *testing.T) {
		t.Parallel()

		for i := 1; i < len(model.SpaceRoles); i++ {
			wider, narrower := model.SpaceRoles[i-1], model.SpaceRoles[i]
			widerScopes := expand(wider.Scopes())
			for _, s := range expand(narrower.Scopes()) {
				if !model.HasScope(widerScopes, s) {
					t.Errorf("%sが持つ%sを%sが持たない", narrower, s, wider)
				}
			}
		}
		for i := 1; i < len(model.TopicRoles); i++ {
			wider, narrower := model.TopicRoles[i-1], model.TopicRoles[i]
			widerScopes := expand(wider.Scopes())
			for _, s := range expand(narrower.Scopes()) {
				if !model.HasScope(widerScopes, s) {
					t.Errorf("トピックの%sが持つ%sをトピックの%sが持たない", narrower, s, wider)
				}
			}
		}
	})

	t.Run("未知のロールはスコープを持たない", func(t *testing.T) {
		t.Parallel()

		if got := model.SpaceRole("owner").Scopes(); got != nil {
			t.Errorf("SpaceRole(\"owner\").Scopes() = %v、期待値 = nil", got)
		}
		if got := model.TopicRole("owner").Scopes(); got != nil {
			t.Errorf("TopicRole(\"owner\").Scopes() = %v、期待値 = nil", got)
		}
	})

	t.Run("返したスコープを書き換えてもロールの定義は変わらない", func(t *testing.T) {
		t.Parallel()

		scopes := model.SpaceRoleEditor.Scopes()
		scopes[0] = model.ScopeSpaceDelete
		if model.HasScope(model.SpaceRoleEditor.Scopes(), model.ScopeSpaceDelete) {
			t.Error("SpaceRoleEditor.Scopes()の戻り値の書き換えが定義に及んでいる")
		}

		topicScopes := model.TopicRoleViewer.Scopes()
		topicScopes[0] = model.ScopeTopicDelete
		if model.HasScope(model.TopicRoleViewer.Scopes(), model.ScopeTopicDelete) {
			t.Error("TopicRoleViewer.Scopes()の戻り値の書き換えが定義に及んでいる")
		}
	})
}

func TestSpaceRole_RailsScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role model.SpaceRole
		want []model.Scope
	}{
		{
			name: "管理者はspace:adminだけを書く",
			role: model.SpaceRoleAdmin,
			want: []model.Scope{model.ScopeSpaceAdmin},
		},
		{
			name: "編集者はロールのスコープをそのまま書く",
			role: model.SpaceRoleEditor,
			want: model.SpaceRoleEditor.Scopes(),
		},
		{
			name: "閲覧者はロールのスコープをそのまま書く",
			role: model.SpaceRoleViewer,
			want: model.SpaceRoleViewer.Scopes(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.role.RailsScopes()
			if !slices.Equal(got, tt.want) {
				t.Errorf("RailsScopes() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestRoleScopes(t *testing.T) {
	t.Parallel()

	// 期待値をロールの実装から生成せず、権限の過剰な付与と欠落の両方を検出する。
	tests := []struct {
		name   string
		scopes func() []model.Scope
		want   []model.Scope
	}{
		{
			name:   "スペース編集者",
			scopes: model.SpaceRoleEditor.Scopes,
			want: []model.Scope{
				model.ScopeSpaceMemberRead,
				model.ScopeAttachmentWrite,
				model.ScopePersonalAccessTokenWrite,
				model.ScopePersonalAccessTokenDelete,
				model.ScopeOAuthGrantWrite,
				model.ScopeOAuthGrantDelete,
				model.ScopeTopicWrite,
				model.ScopeTopicMemberRead,
				model.ScopePageWrite,
				model.ScopePageTrashWrite,
				model.ScopePageTrashDelete,
				model.ScopeDraftPageWrite,
				model.ScopeDraftPageDelete,
				model.ScopeSuggestionWrite,
				model.ScopeSuggestionApplicationWrite,
				model.ScopeSuggestionClosureWrite,
				model.ScopeSuggestionCommentWrite,
			},
		},
		{
			name:   "スペース閲覧者",
			scopes: model.SpaceRoleViewer.Scopes,
			want: []model.Scope{
				model.ScopeSpaceMemberRead,
				model.ScopeAttachmentRead,
				model.ScopePersonalAccessTokenWrite,
				model.ScopePersonalAccessTokenDelete,
				model.ScopeOAuthGrantWrite,
				model.ScopeOAuthGrantDelete,
				model.ScopeTopicRead,
				model.ScopeTopicMemberRead,
				model.ScopePageRead,
				model.ScopeSuggestionWrite,
				model.ScopeSuggestionCommentWrite,
			},
		},
		{
			name:   "トピック管理者",
			scopes: model.TopicRoleAdmin.Scopes,
			want: []model.Scope{
				model.ScopeTopicWrite,
				model.ScopeTopicDelete,
				model.ScopeTopicVisibilityWrite,
				model.ScopeTopicMemberWrite,
				model.ScopeTopicMemberDelete,
				model.ScopePageWrite,
				model.ScopePageTrashWrite,
				model.ScopePageTrashDelete,
				model.ScopeDraftPageWrite,
				model.ScopeDraftPageDelete,
				model.ScopeSuggestionWrite,
				model.ScopeSuggestionApplicationWrite,
				model.ScopeSuggestionClosureWrite,
				model.ScopeSuggestionCommentWrite,
			},
		},
		{
			name:   "トピック編集者",
			scopes: model.TopicRoleEditor.Scopes,
			want: []model.Scope{
				model.ScopeTopicRead,
				model.ScopeTopicMemberRead,
				model.ScopePageWrite,
				model.ScopePageTrashWrite,
				model.ScopePageTrashDelete,
				model.ScopeDraftPageWrite,
				model.ScopeDraftPageDelete,
				model.ScopeSuggestionWrite,
				model.ScopeSuggestionApplicationWrite,
				model.ScopeSuggestionClosureWrite,
				model.ScopeSuggestionCommentWrite,
			},
		},
		{
			name:   "トピック閲覧者",
			scopes: model.TopicRoleViewer.Scopes,
			want: []model.Scope{
				model.ScopeTopicRead,
				model.ScopeTopicMemberRead,
				model.ScopePageRead,
				model.ScopeSuggestionWrite,
				model.ScopeSuggestionCommentWrite,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.scopes()
			for _, scope := range tt.want {
				if !slices.Contains(got, scope) {
					t.Errorf("必要なスコープ%sが無い", scope)
				}
			}
			for _, scope := range got {
				if !slices.Contains(tt.want, scope) {
					t.Errorf("許可していないスコープ%sを持っている", scope)
				}
			}
		})
	}
}
