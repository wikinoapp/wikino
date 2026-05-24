package policy

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestNewMemberPolicy(t *testing.T) {
	t.Parallel()

	t.Run("スペースス��ープとトピックスコープの和集合を取る", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(
			[]model.Scope{model.ScopePageWrite},
			[]model.Scope{model.ScopeSuggestionWrite},
		)

		if !p.effectiveScopes[model.ScopePageWrite] {
			t.Error("スペーススコープの page:write が含まれるべき")
		}
		if !p.effectiveScopes[model.ScopeSuggestionWrite] {
			t.Error("トピックスコープの suggestion:write が含まれるべき")
		}
	})

	t.Run("含意ルールが展開される", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(
			[]model.Scope{model.ScopePageWrite},
			nil,
		)

		if !p.effectiveScopes[model.ScopePageRead] {
			t.Error("page:write から page:read が含意展開されるべき")
		}
	})

	t.Run("space:adminで全スコープが展開される", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(
			[]model.Scope{model.ScopeSpaceAdmin},
			nil,
		)

		for _, s := range allResourceScopes() {
			if !p.effectiveScopes[s] {
				t.Errorf("space:admin で %s が含まれるべき", s)
			}
		}
	})

	t.Run("空のスコープで生成できる", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(nil, nil)

		if len(p.effectiveScopes) != 0 {
			t.Errorf("空のスコープで effectiveScopes は空であるべき、got %d", len(p.effectiveScopes))
		}
	})
}

func TestMemberPolicy_CanShowTopic(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックはスコープなしでも閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(nil, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPublic}

		if !p.CanShowTopic(topic) {
			t.Error("公開トピックは閲覧可能であるべき")
		}
	})

	t.Run("非公開トピックはtopic:readで閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicRead}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if !p.CanShowTopic(topic) {
			t.Error("topic:read を持つメンバーは非公開トピックを閲覧可能であるべき")
		}
	})

	t.Run("非公開トピックはtopic:readなしで閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if p.CanShowTopic(topic) {
			t.Error("topic:read を持たないメンバーは非公開トピックを閲覧できないべき")
		}
	})

	t.Run("space:adminは非公開トピックを閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if !p.CanShowTopic(topic) {
			t.Error("space:admin は非公開トピックを閲覧可能であるべき")
		}
	})
}

func TestMemberPolicy_CanCreatePage(t *testing.T) {
	t.Parallel()

	t.Run("page:writeで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)

		if !p.CanCreatePage() {
			t.Error("page:write を持つメンバーはページを作成可能であるべき")
		}
	})

	t.Run("page:writeなしで作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageRead}, nil)

		if p.CanCreatePage() {
			t.Error("page:write を持たないメンバーはページを作成できないべき")
		}
	})
}

func TestMemberPolicy_CanUpdatePage(t *testing.T) {
	t.Parallel()

	t.Run("page:writeで編集��能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)

		if !p.CanUpdatePage() {
			t.Error("page:write を持つメンバーはページを編集��能であるべき")
		}
	})

	t.Run("page:writeなしで編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageRead}, nil)

		if p.CanUpdatePage() {
			t.Error("page:write を持たないメンバーはページを編集できないべき")
		}
	})
}

func TestMemberPolicy_CanShowDraftPage(t *testing.T) {
	t.Parallel()

	t.Run("所有者はdraft_page:readで閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageRead}, nil)

		if !p.CanShowDraftPage(true) {
			t.Error("所有者かつ draft_page:read を持つメンバーは閲覧可能であるべき")
		}
	})

	t.Run("非所有者はdraft_page:readだけでは閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageRead}, nil)

		if p.CanShowDraftPage(false) {
			t.Error("非所有者は draft_page:read だけでは閲覧できないべき")
		}
	})

	t.Run("非所有者でもspace:adminなら閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanShowDraftPage(false) {
			t.Error("space:admin は非所有者でも閲覧可能であるべき")
		}
	})

	t.Run("draft_page:readなしでは所有者でも閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageRead}, nil)

		if p.CanShowDraftPage(true) {
			t.Error("draft_page:read を持たないメンバーは所有者でも閲覧できないべき")
		}
	})
}

func TestMemberPolicy_CanUpdateDraftPage(t *testing.T) {
	t.Parallel()

	t.Run("所有者はdraft_page:writeで編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageWrite}, nil)

		if !p.CanUpdateDraftPage(true) {
			t.Error("所有者かつ draft_page:write を持つメンバーは編集可能であるべき")
		}
	})

	t.Run("非所��者はdraft_page:writeだけでは編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageWrite}, nil)

		if p.CanUpdateDraftPage(false) {
			t.Error("非所有者は draft_page:write だけでは編集できないべき")
		}
	})

	t.Run("非所有者でもspace:adminなら編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanUpdateDraftPage(false) {
			t.Error("space:admin は非所有者でも編���可能であるべき")
		}
	})

	t.Run("draft_page:writeなしでは所有者でも編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)

		if p.CanUpdateDraftPage(true) {
			t.Error("draft_page:write を持たないメンバーは所有者でも編集できないべき")
		}
	})
}

func TestMemberPolicy_CanDeleteDraftPage(t *testing.T) {
	t.Parallel()

	t.Run("draft_page:deleteで削除可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageDelete}, nil)

		if !p.CanDeleteDraftPage() {
			t.Error("draft_page:delete を持つメンバーは削除可能であるべき")
		}
	})

	t.Run("space:adminで削除可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanDeleteDraftPage() {
			t.Error("space:admin は draft_page:delete を含意展開するため削除可能であるべき")
		}
	})

	t.Run("draft_page:writeだけでは削除不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageWrite}, nil)

		if p.CanDeleteDraftPage() {
			t.Error("draft_page:write だけでは削除できないべき (draft_page:delete が必要)")
		}
	})
}

func TestMemberPolicy_CanCreateSuggestion(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックにsuggestion:writeで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPublic}

		if !p.CanCreateSuggestion(topic) {
			t.Error("suggestion:write を持つメンバーは公開トピックに編集提案を作成可能であるべき")
		}
	})

	t.Run("非公開トピックにsuggestion:write+topic:readで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite, model.ScopeTopicRead}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if !p.CanCreateSuggestion(topic) {
			t.Error("suggestion:write+topic:read を持つメンバーは非公開トピックに編集提案を作成可能であるべき")
		}
	})

	t.Run("非公開トピックにsuggestion:writeのみでは作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if p.CanCreateSuggestion(topic) {
			t.Error("topic:read なしでは非公開トピックに編集提案を作成できないべき")
		}
	})

	t.Run("suggestion:writeなしでは作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicRead}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPublic}

		if p.CanCreateSuggestion(topic) {
			t.Error("suggestion:write を持たないメンバーは編集提案を作成できないべき")
		}
	})
}

func TestMemberPolicy_CanApplySuggestion(t *testing.T) {
	t.Parallel()

	t.Run("suggestion:applyで反映可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionApply}, nil)

		if !p.CanApplySuggestion() {
			t.Error("suggestion:apply を持つメンバーは反映可能であるべき")
		}
	})

	t.Run("suggestion:applyなしで反映不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)

		if p.CanApplySuggestion() {
			t.Error("suggestion:apply を持たないメンバーは反映できないべき")
		}
	})
}

func TestMemberPolicy_CanCloseSuggestion(t *testing.T) {
	t.Parallel()

	t.Run("suggestion:closeで他人の提案もクローズ可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionClose}, nil)

		if !p.CanCloseSuggestion(false) {
			t.Error("suggestion:close を持つメンバーは他人の提案もクローズ可能であるべ��")
		}
	})

	t.Run("作成者は自分の提案をクローズ可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)

		if !p.CanCloseSuggestion(true) {
			t.Error("作成者は自分の提案をクローズ可能であるべき")
		}
	})

	t.Run("suggestion:closeなしで他人の提案はクロー���不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)

		if p.CanCloseSuggestion(false) {
			t.Error("suggestion:close を持たない非作成者はクローズできないべき")
		}
	})
}

func TestMemberPolicy_CanUpdateSuggestion(t *testing.T) {
	t.Parallel()

	t.Run("suggestion:writeでオープン提案を編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if !p.CanUpdateSuggestion(suggestion) {
			t.Error("suggestion:write を持つメンバーはオープンな編集提案を編集可能であるべき")
		}
	})

	t.Run("クローズ済み提案は編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusClosed}

		if p.CanUpdateSuggestion(suggestion) {
			t.Error("クローズ済みの編集提案は編集できないべき")
		}
	})

	t.Run("反映済み提案は編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusApplied}

		if p.CanUpdateSuggestion(suggestion) {
			t.Error("反映済みの編集提案は編集できない���き")
		}
	})

	t.Run("suggestion:writeなしで編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionRead}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if p.CanUpdateSuggestion(suggestion) {
			t.Error("suggestion:write を持たないメンバーは編集でき���いべき")
		}
	})
}

func TestMemberPolicy_CanAddSuggestionPage(t *testing.T) {
	t.Parallel()

	t.Run("suggestion:writeでオープン提案にページ追加可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if !p.CanAddSuggestionPage(suggestion) {
			t.Error("suggestion:write を持つメンバーはオープンな提案にページを追加可能であるべき")
		}
	})

	t.Run("クローズ済み提案にはページ追加不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusClosed}

		if p.CanAddSuggestionPage(suggestion) {
			t.Error("クローズ済みの提案にはページを追加できないべき")
		}
	})
}

func TestMemberPolicy_CanRemoveSuggestionPage(t *testing.T) {
	t.Parallel()

	t.Run("suggestion:writeでオープン提案からページ削除可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if !p.CanRemoveSuggestionPage(suggestion) {
			t.Error("suggestion:write を持つメンバーはオープンな提案からページを削除可能であるべき")
		}
	})

	t.Run("クローズ済み提案からはページ削除不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusClosed}

		if p.CanRemoveSuggestionPage(suggestion) {
			t.Error("クローズ済みの提案からはページを削除できない���き")
		}
	})
}

func TestMemberPolicy_CanEditSuggestionPage(t *testing.T) {
	t.Parallel()

	t.Run("suggestion:writeで編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)

		if !p.CanEditSuggestionPage() {
			t.Error("suggestion:write を持��メンバーは編集提案ページを編集可能であるべき")
		}
	})

	t.Run("suggestion:writeなしで編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionRead}, nil)

		if p.CanEditSuggestionPage() {
			t.Error("suggestion:write ���持たないメンバーは編集提案ページを編集できないべき")
		}
	})
}

func TestMemberPolicy_CanCreateSuggestionComment(t *testing.T) {
	t.Parallel()

	t.Run("suggestion_comment:writeで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionCommentWrite}, nil)

		if !p.CanCreateSuggestionComment() {
			t.Error("suggestion_comment:write を持つメンバーはコメントを作成可能であるべき")
		}
	})

	t.Run("suggestion_comment:writeなしで作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionCommentRead}, nil)

		if p.CanCreateSuggestionComment() {
			t.Error("suggestion_comment:write を持たないメンバーはコメントを作成できないべき")
		}
	})
}

func TestMemberPolicy_CanUpdateSuggestionComment(t *testing.T) {
	t.Parallel()

	t.Run("suggestion_comment:writeでオープン提案のコメントを編集可��", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionCommentWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if !p.CanUpdateSuggestionComment(suggestion) {
			t.Error("suggestion_comment:write を持つメンバーはオープンな提案のコメントを編集可能であるべき")
		}
	})

	t.Run("クローズ済み提案のコメントは編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionCommentWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusClosed}

		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("クローズ済みの提案のコメントは編集できないべき")
		}
	})

	t.Run("suggestion_comment:writeなしで編集不���", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("suggestion_comment:write を持たないメンバーはコメントを編集できないべき")
		}
	})
}

func TestMemberPolicy_CanCreateTopic(t *testing.T) {
	t.Parallel()

	t.Run("topic:writeで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicWrite}, nil)

		if !p.CanCreateTopic() {
			t.Error("topic:write を持つメンバーはトピックを作成可能であるべき")
		}
	})

	t.Run("space:adminで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanCreateTopic() {
			t.Error("space:admin は topic:write を含意展開するためトピックを作成可能であるべき")
		}
	})

	t.Run("topic:writeなしで作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicRead}, nil)

		if p.CanCreateTopic() {
			t.Error("topic:write を持たないメンバーはトピックを作成できないべき")
		}
	})
}

// TestMemberPolicy_Authorizer は MemberPolicy が Authorizer インターフェースを満たすことを検証する
func TestMemberPolicy_Authorizer(t *testing.T) {
	t.Parallel()

	var _ Authorizer = NewMemberPolicy(nil, nil)
}
