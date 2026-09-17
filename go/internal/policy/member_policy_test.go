package policy

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestNewMemberPolicy(t *testing.T) {
	t.Parallel()

	t.Run("スペーススコープとトピックスコープの和集合を取る", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(
			[]model.Scope{model.ScopePageWrite},
			[]model.Scope{model.ScopeSuggestionWrite},
		)

		if !p.effectiveScopes[model.ScopePageWrite] {
			t.Error("スペーススコープのpage:writeが含まれるべき")
		}
		if !p.effectiveScopes[model.ScopeSuggestionWrite] {
			t.Error("トピックスコープのsuggestion:writeが含まれるべき")
		}
	})

	t.Run("含意ルールが展開される", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(
			[]model.Scope{model.ScopePageWrite},
			nil,
		)

		if !p.effectiveScopes[model.ScopePageRead] {
			t.Error("page:writeからpage:readが含意展開されるべき")
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
				t.Errorf("space:adminで%sが含まれるべき", s)
			}
		}
	})

	t.Run("空のスコープで生成できる", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(nil, nil)

		if len(p.effectiveScopes) != 0 {
			t.Errorf("空のスコープでのeffectiveScopesの件数 = %d、期待値 = 0", len(p.effectiveScopes))
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
			t.Error("topic:readを持つメンバーは非公開トピックを閲覧可能であるべき")
		}
	})

	t.Run("非公開トピックはtopic:readなしで閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if p.CanShowTopic(topic) {
			t.Error("topic:readを持たないメンバーは非公開トピックを閲覧できないべき")
		}
	})

	t.Run("space:adminは非公開トピックを閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if !p.CanShowTopic(topic) {
			t.Error("space:adminは非公開トピックを閲覧可能であるべき")
		}
	})
}

func TestMemberPolicy_CanCreatePage(t *testing.T) {
	t.Parallel()

	t.Run("page:writeで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)

		if !p.CanCreatePage() {
			t.Error("page:writeを持つメンバーはページを作成可能であるべき")
		}
	})

	t.Run("page:writeなしで作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageRead}, nil)

		if p.CanCreatePage() {
			t.Error("page:writeを持たないメンバーはページを作成できないべき")
		}
	})
}

func TestMemberPolicy_CanUpdatePage(t *testing.T) {
	t.Parallel()

	t.Run("page:writeで編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)

		if !p.CanUpdatePage() {
			t.Error("page:writeを持つメンバーはページを編集可能であるべき")
		}
	})

	t.Run("page:writeなしで編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageRead}, nil)

		if p.CanUpdatePage() {
			t.Error("page:writeを持たないメンバーはページを編集できないべき")
		}
	})
}

func TestMemberPolicy_CanShowTrash(t *testing.T) {
	t.Parallel()

	t.Run("page_trash:writeで閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageTrashWrite}, nil)

		if !p.CanShowTrash() {
			t.Error("page_trash:writeを持つメンバーはゴミ箱を閲覧可能であるべき")
		}
	})

	t.Run("space:adminで閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanShowTrash() {
			t.Error("space:adminはpage_trash:writeを含意展開するためゴミ箱を閲覧可能であるべき")
		}
	})

	t.Run("page:writeだけでは閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)

		if p.CanShowTrash() {
			t.Error("page:writeだけではゴミ箱を閲覧できないべき (page_trash:readが必要)")
		}
	})

	t.Run("page:readだけでは閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageRead}, nil)

		if p.CanShowTrash() {
			t.Error("page:readだけではゴミ箱を閲覧できないべき (page_trash:readが必要)")
		}
	})

	t.Run("page_trash:deleteだけでは閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageTrashDelete}, nil)

		if p.CanShowTrash() {
			t.Error("page_trash:deleteだけではゴミ箱を閲覧できないべき (page_trash:readが必要)")
		}
	})

	t.Run("トピックスコープのpage_trash:writeでも閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(nil, []model.Scope{model.ScopePageTrashWrite})

		if !p.CanShowTrash() {
			t.Error("トピックスコープでpage_trash:writeを持つメンバーもゴミ箱を閲覧可能であるべき")
		}
	})
}

func TestMemberPolicy_CanTrashPage(t *testing.T) {
	t.Parallel()

	t.Run("page_trash:writeでゴミ箱に入れられる", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageTrashWrite}, nil)

		if !p.CanTrashPage() {
			t.Error("page_trash:writeを持つメンバーはページをゴミ箱に入れられるべき")
		}
	})

	t.Run("space:adminでゴミ箱に入れられる", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanTrashPage() {
			t.Error("space:adminはpage_trash:writeを含意展開するためページをゴミ箱に入れられるべき")
		}
	})

	t.Run("page:writeだけでは入れられない", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)

		if p.CanTrashPage() {
			t.Error("page:writeだけではページをゴミ箱に入れられないべき (page_trash:writeが必要)")
		}
	})

	t.Run("page:readだけでは入れられない", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageRead}, nil)

		if p.CanTrashPage() {
			t.Error("page:readだけではページをゴミ箱に入れられないべき (page_trash:writeが必要)")
		}
	})

	t.Run("page_trash:deleteだけでは入れられない", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageTrashDelete}, nil)

		if p.CanTrashPage() {
			t.Error("page_trash:deleteだけではページをゴミ箱に入れられないべき (page_trash:writeが必要)")
		}
	})

	t.Run("トピックスコープのpage_trash:writeでも入れられる", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(nil, []model.Scope{model.ScopePageTrashWrite})

		if !p.CanTrashPage() {
			t.Error("トピックスコープでpage_trash:writeを持つメンバーもページをゴミ箱に入れられるべき")
		}
	})
}

func TestMemberPolicy_CanShowDraftPage(t *testing.T) {
	t.Parallel()

	t.Run("所有者はdraft_page:readで閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageRead}, nil)

		if !p.CanShowDraftPage(true) {
			t.Error("所有者かつdraft_page:readを持つメンバーは閲覧可能であるべき")
		}
	})

	t.Run("非所有者はdraft_page:readだけでは閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageRead}, nil)

		if p.CanShowDraftPage(false) {
			t.Error("非所有者はdraft_page:readだけでは閲覧できないべき")
		}
	})

	t.Run("非所有者でもspace:adminなら閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanShowDraftPage(false) {
			t.Error("space:adminは非所有者でも閲覧可能であるべき")
		}
	})

	t.Run("draft_page:readなしでは所有者でも閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageRead}, nil)

		if p.CanShowDraftPage(true) {
			t.Error("draft_page:readを持たないメンバーは所有者でも閲覧できないべき")
		}
	})
}

func TestMemberPolicy_CanUpdateDraftPage(t *testing.T) {
	t.Parallel()

	t.Run("所有者はdraft_page:writeで編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageWrite}, nil)

		if !p.CanUpdateDraftPage(true) {
			t.Error("所有者かつdraft_page:writeを持つメンバーは編集可能であるべき")
		}
	})

	t.Run("非所有者はdraft_page:writeだけでは編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageWrite}, nil)

		if p.CanUpdateDraftPage(false) {
			t.Error("非所有者はdraft_page:writeだけでは編集できないべき")
		}
	})

	t.Run("非所有者でもspace:adminなら編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanUpdateDraftPage(false) {
			t.Error("space:adminは非所有者でも編集可能であるべき")
		}
	})

	t.Run("draft_page:writeなしでは所有者でも編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopePageWrite}, nil)

		if p.CanUpdateDraftPage(true) {
			t.Error("draft_page:writeを持たないメンバーは所有者でも編集できないべき")
		}
	})
}

func TestMemberPolicy_CanDeleteDraftPage(t *testing.T) {
	t.Parallel()

	t.Run("draft_page:deleteで削除可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageDelete}, nil)

		if !p.CanDeleteDraftPage() {
			t.Error("draft_page:deleteを持つメンバーは削除可能であるべき")
		}
	})

	t.Run("space:adminで削除可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanDeleteDraftPage() {
			t.Error("space:adminはdraft_page:deleteを含意展開するため削除可能であるべき")
		}
	})

	t.Run("draft_page:writeだけでは削除不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageWrite}, nil)

		if p.CanDeleteDraftPage() {
			t.Error("draft_page:writeだけでは削除できないべき (draft_page:deleteが必要)")
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
			t.Error("suggestion:writeを持つメンバーは公開トピックに編集提案を作成可能であるべき")
		}
	})

	t.Run("非公開トピックにsuggestion:write+topic:readで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite, model.ScopeTopicRead}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if !p.CanCreateSuggestion(topic) {
			t.Error("suggestion:write+topic:readを持つメンバーは非公開トピックに編集提案を作成可能であるべき")
		}
	})

	t.Run("非公開トピックにsuggestion:writeのみでは作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if p.CanCreateSuggestion(topic) {
			t.Error("topic:readなしでは非公開トピックに編集提案を作成できないべき")
		}
	})

	t.Run("suggestion:writeなしでは作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicRead}, nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPublic}

		if p.CanCreateSuggestion(topic) {
			t.Error("suggestion:writeを持たないメンバーは編集提案を作成できないべき")
		}
	})
}

func TestMemberPolicy_CanApplySuggestion(t *testing.T) {
	t.Parallel()

	t.Run("suggestion_application:writeで反映可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionApplicationWrite}, nil)

		if !p.CanApplySuggestion() {
			t.Error("suggestion_application:writeを持つメンバーは反映可能であるべき")
		}
	})

	t.Run("suggestion_application:writeなしで反映不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)

		if p.CanApplySuggestion() {
			t.Error("suggestion_application:writeを持たないメンバーは反映できないべき")
		}
	})
}

func TestMemberPolicy_CanCloseSuggestion(t *testing.T) {
	t.Parallel()

	t.Run("suggestion_closure:writeで他人の提案もクローズ可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionClosureWrite}, nil)

		if !p.CanCloseSuggestion(false) {
			t.Error("suggestion_closure:writeを持つメンバーは他人の提案もクローズ可能であるべき")
		}
	})

	t.Run("作成者は自分の提案をクローズ可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)

		if !p.CanCloseSuggestion(true) {
			t.Error("作成者は自分の提案をクローズ可能であるべき")
		}
	})

	t.Run("suggestion_closure:writeなしで他人の提案はクローズ不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)

		if p.CanCloseSuggestion(false) {
			t.Error("suggestion_closure:writeを持たない非作成者はクローズできないべき")
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
			t.Error("suggestion:writeを持つメンバーはオープンな編集提案を編集可能であるべき")
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
			t.Error("反映済みの編集提案は編集できないべき")
		}
	})

	t.Run("suggestion:writeなしで編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionRead}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if p.CanUpdateSuggestion(suggestion) {
			t.Error("suggestion:writeを持たないメンバーは編集できないべき")
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
			t.Error("suggestion:writeを持つメンバーはオープンな提案にページを追加可能であるべき")
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
			t.Error("suggestion:writeを持つメンバーはオープンな提案からページを削除可能であるべき")
		}
	})

	t.Run("クローズ済み提案からはページ削除不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusClosed}

		if p.CanRemoveSuggestionPage(suggestion) {
			t.Error("クローズ済みの提案からはページを削除できないべき")
		}
	})
}

func TestMemberPolicy_CanEditSuggestionPage(t *testing.T) {
	t.Parallel()

	t.Run("suggestion:writeで編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)

		if !p.CanEditSuggestionPage() {
			t.Error("suggestion:writeを持つメンバーは編集提案ページを編集可能であるべき")
		}
	})

	t.Run("suggestion:writeなしで編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionRead}, nil)

		if p.CanEditSuggestionPage() {
			t.Error("suggestion:writeを持たないメンバーは編集提案ページを編集できないべき")
		}
	})
}

func TestMemberPolicy_CanCreateSuggestionComment(t *testing.T) {
	t.Parallel()

	t.Run("suggestion_comment:writeで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionCommentWrite}, nil)

		if !p.CanCreateSuggestionComment() {
			t.Error("suggestion_comment:writeを持つメンバーはコメントを作成可能であるべき")
		}
	})

	t.Run("suggestion_comment:writeなしで作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionCommentRead}, nil)

		if p.CanCreateSuggestionComment() {
			t.Error("suggestion_comment:writeを持たないメンバーはコメントを作成できないべき")
		}
	})
}

func TestMemberPolicy_CanUpdateSuggestionComment(t *testing.T) {
	t.Parallel()

	t.Run("suggestion_comment:writeでオープン提案のコメントを編集可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionCommentWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if !p.CanUpdateSuggestionComment(suggestion) {
			t.Error("suggestion_comment:writeを持つメンバーはオープンな提案のコメントを編集可能であるべき")
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

	t.Run("suggestion_comment:writeなしで編集不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSuggestionWrite}, nil)
		suggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("suggestion_comment:writeを持たないメンバーはコメントを編集できないべき")
		}
	})
}

func TestMemberPolicy_CanCreateTopic(t *testing.T) {
	t.Parallel()

	t.Run("topic:writeで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicWrite}, nil)

		if !p.CanCreateTopic() {
			t.Error("topic:writeを持つメンバーはトピックを作成可能であるべき")
		}
	})

	t.Run("space:adminで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanCreateTopic() {
			t.Error("space:adminはtopic:writeを含意展開するためトピックを作成可能であるべき")
		}
	})

	t.Run("topic:writeなしで作成不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicRead}, nil)

		if p.CanCreateTopic() {
			t.Error("topic:writeを持たないメンバーはトピックを作成できないべき")
		}
	})
}

// TestMemberPolicy_AuthorizerはMemberPolicyがAuthorizerインターフェースを満たすことを検証する
func TestMemberPolicy_Authorizer(t *testing.T) {
	t.Parallel()

	var _ Authorizer = NewMemberPolicy(nil, nil)
}

func TestMemberPolicy_CanExportSpace(t *testing.T) {
	t.Parallel()

	t.Run("space:writeでエクスポート可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceWrite}, nil)

		if !p.CanExportSpace() {
			t.Error("space:writeを持つメンバーはスペースをエクスポート可能であるべき")
		}
	})

	t.Run("space:adminでエクスポート可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if !p.CanExportSpace() {
			t.Error("space:adminはspace:writeを含意展開するためスペースをエクスポート可能であるべき")
		}
	})

	t.Run("読み取り権限だけではエクスポート不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceRead, model.ScopePageRead}, nil)

		if p.CanExportSpace() {
			t.Error("space:writeを持たないメンバーはスペースをエクスポートできないべき")
		}
	})
}

func TestMemberPolicy_ScopeBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                                                                               string
		scopes                                                                             []model.Scope
		showTrash, trashPage, updatePage, editSuggestion, applySuggestion, closeSuggestion bool
	}{
		{name: "ゴミ箱閲覧", scopes: []model.Scope{model.ScopePageTrashRead}, showTrash: true},
		{name: "ゴミ箱移動", scopes: []model.Scope{model.ScopePageTrashWrite}, showTrash: true, trashPage: true},
		{name: "旧ゴミ箱移動", scopes: []model.Scope{model.ScopePageTrash}, showTrash: true, trashPage: true},
		{name: "ゴミ箱復元", scopes: []model.Scope{model.ScopePageTrashDelete}},
		{name: "旧ゴミ箱復元", scopes: []model.Scope{model.ScopePageRestore}},
		{name: "ページ編集", scopes: []model.Scope{model.ScopePageWrite}, updatePage: true},
		{name: "編集提案編集", scopes: []model.Scope{model.ScopeSuggestionWrite}, editSuggestion: true},
		{name: "編集提案反映", scopes: []model.Scope{model.ScopeSuggestionApplicationWrite}, applySuggestion: true},
		{name: "旧編集提案反映", scopes: []model.Scope{model.ScopeSuggestionApply}, applySuggestion: true},
		{name: "編集提案クローズ", scopes: []model.Scope{model.ScopeSuggestionClosureWrite}, closeSuggestion: true},
		{name: "旧編集提案クローズ", scopes: []model.Scope{model.ScopeSuggestionClose}, closeSuggestion: true},
		{name: "新旧混在", scopes: []model.Scope{model.ScopePageTrash, model.ScopePageTrashWrite, model.ScopeSuggestionClose, model.ScopeSuggestionClosureWrite}, showTrash: true, trashPage: true, closeSuggestion: true},
		{name: "管理者", scopes: []model.Scope{model.ScopeSpaceAdmin}, showTrash: true, trashPage: true, updatePage: true, editSuggestion: true, applySuggestion: true, closeSuggestion: true},
		{name: "未知のスコープ", scopes: []model.Scope{"page:trash:write", "page:restore_extra", "suggestion:apply_extra", "suggestion:close_extra", "space:admin_extra"}},
		{name: "スコープなし"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			for _, source := range []struct {
				name         string
				space, topic []model.Scope
			}{
				{"スペース", tt.scopes, nil},
				{"トピック", nil, tt.scopes},
				{"両方", tt.scopes, tt.scopes},
			} {
				t.Run(source.name, func(t *testing.T) {
					t.Parallel()

					p := NewMemberPolicy(source.space, source.topic)
					for _, check := range []struct {
						name      string
						got, want bool
					}{
						{"CanShowTrash", p.CanShowTrash(), tt.showTrash},
						{"CanTrashPage", p.CanTrashPage(), tt.trashPage},
						{"CanUpdatePage", p.CanUpdatePage(), tt.updatePage},
						{"CanEditSuggestionPage", p.CanEditSuggestionPage(), tt.editSuggestion},
						{"CanApplySuggestion", p.CanApplySuggestion(), tt.applySuggestion},
						{"CanCloseSuggestion", p.CanCloseSuggestion(false), tt.closeSuggestion},
						{"作成者のCanCloseSuggestion", p.CanCloseSuggestion(true), true},
					} {
						if check.got != check.want {
							t.Errorf("%s = %v、期待値 = %v", check.name, check.got, check.want)
						}
					}
				})
			}
		})
	}
}
