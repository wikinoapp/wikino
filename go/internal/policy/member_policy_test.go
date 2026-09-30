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

	t.Run("管理者のロールで定義の表の全スコープが有効になる", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(model.SpaceRoleAdmin.Scopes(), nil)

		for _, d := range model.ScopeDefinitions {
			if !p.effectiveScopes[d.Scope] {
				t.Errorf("管理者のロールで%sが含まれるべき", d.Scope)
			}
		}
	})

	t.Run("space:adminは権限を生まない", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeSpaceAdmin}, nil)

		if p.CanUpdateSpace() || p.CanCreatePage() {
			t.Error("space:adminはRails版向けの値であり、Go版の判定で権限を生むべきでない")
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

func TestNewAPIMemberPolicy(t *testing.T) {
	t.Parallel()

	t.Run("メンバーが持ちトークンが持たないスコープは有効にならない", func(t *testing.T) {
		t.Parallel()

		p := NewAPIMemberPolicy(
			[]model.Scope{model.ScopePageWrite, model.ScopeTopicRead},
			nil,
			[]model.Scope{model.ScopePageRead},
		)

		if !p.effectiveScopes[model.ScopePageRead] {
			t.Error("両方が持つpage:readは有効であるべき")
		}
		if p.CanCreatePage() {
			t.Error("トークンが持たないpage:writeでページを作成できるべきでない")
		}
		if p.CanShowTopic(&model.Topic{Visibility: model.TopicVisibilityPrivate}) {
			t.Error("トークンが持たないtopic:readで非公開トピックを閲覧できるべきでない")
		}
	})

	t.Run("トークンが持ちメンバーが持たないスコープは有効にならない", func(t *testing.T) {
		t.Parallel()

		p := NewAPIMemberPolicy(
			[]model.Scope{model.ScopePageRead},
			nil,
			[]model.Scope{model.ScopePageWrite, model.ScopeTopicRead},
		)

		if !p.effectiveScopes[model.ScopePageRead] {
			t.Error("メンバーのpage:readとトークンのpage:write (含意でpage:read) の積でpage:readは有効であるべき")
		}
		if p.CanCreatePage() {
			t.Error("メンバーが持たないpage:writeでページを作成できるべきでない")
		}
		if p.CanShowTopic(&model.Topic{Visibility: model.TopicVisibilityPrivate}) {
			t.Error("メンバーが持たないtopic:readで非公開トピックを閲覧できるべきでない")
		}
	})

	t.Run("トピックスコープもメンバー側に含めてから積を取る", func(t *testing.T) {
		t.Parallel()

		p := NewAPIMemberPolicy(
			nil,
			[]model.Scope{model.ScopeTopicWrite},
			[]model.Scope{model.ScopeTopicRead},
		)

		if !p.CanShowTopic(&model.Topic{Visibility: model.TopicVisibilityPrivate}) {
			t.Error("トピックスコープのtopic:write (含意でtopic:read) とトークンのtopic:readの積で非公開トピックを閲覧できるべき")
		}
		if p.CanUpdateTopic() {
			t.Error("トークンが持たないtopic:writeでトピックを更新できるべきでない")
		}
	})

	t.Run("管理者のロールを持つメンバーでもトークンのスコープに限られる", func(t *testing.T) {
		t.Parallel()

		p := NewAPIMemberPolicy(
			model.SpaceRoleAdmin.Scopes(),
			nil,
			[]model.Scope{model.ScopePageWrite, model.ScopeTopicRead},
		)

		want := map[model.Scope]bool{
			model.ScopePageRead:  true,
			model.ScopePageWrite: true,
			model.ScopeTopicRead: true,
		}
		if len(p.effectiveScopes) != len(want) {
			t.Errorf("effectiveScopes = %v、期待値 = %v", p.effectiveScopes, want)
		}
		for s := range want {
			if !p.effectiveScopes[s] {
				t.Errorf("%sが含まれるべき", s)
			}
		}
		if p.CanUpdateSpace() {
			t.Error("管理者のロールのメンバーでもトークン経由でスペースを更新できるべきでない")
		}
		if p.CanCreatePersonalAccessToken() {
			t.Error("管理者のロールのメンバーでもトークン経由で個人アクセストークンを発行できるべきでない")
		}
	})

	t.Run("トークンに付与できないスコープはトークン側に保存されていても有効にならない", func(t *testing.T) {
		t.Parallel()

		p := NewAPIMemberPolicy(
			model.SpaceRoleAdmin.Scopes(),
			nil,
			[]model.Scope{model.ScopeSpaceAdmin},
		)

		if len(p.effectiveScopes) != 0 {
			t.Errorf("effectiveScopesの件数 = %d、期待値 = 0", len(p.effectiveScopes))
		}
	})

	t.Run("公開トピックはどちらのスコープにも関係なく閲覧できる", func(t *testing.T) {
		t.Parallel()

		p := NewAPIMemberPolicy(nil, nil, nil)

		if !p.CanShowTopic(&model.Topic{Visibility: model.TopicVisibilityPublic}) {
			t.Error("公開トピックは閲覧可能であるべき")
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

	t.Run("管理者のロールは非公開トピックを閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(model.SpaceRoleAdmin.Scopes(), nil)
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if !p.CanShowTopic(topic) {
			t.Error("管理者のロールは非公開トピックを閲覧可能であるべき")
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

	t.Run("管理者のロールで閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(model.SpaceRoleAdmin.Scopes(), nil)

		if !p.CanShowTrash() {
			t.Error("管理者のロールはpage_trash:writeを持つためゴミ箱を閲覧可能であるべき")
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

	t.Run("管理者のロールでゴミ箱に入れられる", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(model.SpaceRoleAdmin.Scopes(), nil)

		if !p.CanTrashPage() {
			t.Error("管理者のロールはpage_trash:writeを持つためページをゴミ箱に入れられるべき")
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

func TestMemberPolicy_CanDeleteDraftPage(t *testing.T) {
	t.Parallel()

	t.Run("draft_page:deleteで削除可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeDraftPageDelete}, nil)

		if !p.CanDeleteDraftPage() {
			t.Error("draft_page:deleteを持つメンバーは削除可能であるべき")
		}
	})

	t.Run("管理者のロールで削除可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(model.SpaceRoleAdmin.Scopes(), nil)

		if !p.CanDeleteDraftPage() {
			t.Error("管理者のロールはdraft_page:deleteを持つため削除可能であるべき")
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

	t.Run("管理者のロールで作成可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(model.SpaceRoleAdmin.Scopes(), nil)

		if !p.CanCreateTopic() {
			t.Error("管理者のロールはtopic:writeを持つためトピックを作成可能であるべき")
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

func TestMemberPolicy_CanUpdateTopicVisibility(t *testing.T) {
	t.Parallel()

	t.Run("topic_visibility:writeで変更可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicVisibilityWrite}, nil)

		if !p.CanUpdateTopicVisibility() {
			t.Error("topic_visibility:writeを持つメンバーは公開範囲を変更可能であるべき")
		}
	})

	t.Run("トピックメンバーのtopic_visibility:writeで変更可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(nil, []model.Scope{model.ScopeTopicVisibilityWrite})

		if !p.CanUpdateTopicVisibility() {
			t.Error("トピックでtopic_visibility:writeを持つメンバーは公開範囲を変更可能であるべき")
		}
	})

	t.Run("管理者のロールで変更可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(model.SpaceRoleAdmin.Scopes(), nil)

		if !p.CanUpdateTopicVisibility() {
			t.Error("管理者のロールはtopic_visibility:writeを持つため公開範囲を変更可能であるべき")
		}
	})

	t.Run("topic:writeだけでは変更不可", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy([]model.Scope{model.ScopeTopicWrite}, nil)

		if p.CanUpdateTopicVisibility() {
			t.Error("topic:writeだけを持つメンバーは公開範囲を変更できないべき")
		}
		if !p.CanUpdateTopic() {
			t.Error("topic:writeを持つメンバーはトピックを更新可能であるべき")
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

	t.Run("管理者のロールでエクスポート可能", func(t *testing.T) {
		t.Parallel()

		p := NewMemberPolicy(model.SpaceRoleAdmin.Scopes(), nil)

		if !p.CanExportSpace() {
			t.Error("管理者のロールはspace:writeを持つためスペースをエクスポート可能であるべき")
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

func TestMemberPolicy_CanUpdateSpace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []model.Scope
		want   bool
	}{
		{name: "space:write", scopes: []model.Scope{model.ScopeSpaceWrite}, want: true},
		{name: "管理者のロールはspace:writeを持つ", scopes: model.SpaceRoleAdmin.Scopes(), want: true},
		{name: "読み取り権限だけ", scopes: []model.Scope{model.ScopeSpaceRead, model.ScopePageRead}, want: false},
		{name: "トークン管理の権限だけ", scopes: []model.Scope{model.ScopePersonalAccessTokenWrite, model.ScopeOAuthGrantWrite}, want: false},
		{name: "OAuthアプリの管理の権限だけ", scopes: []model.Scope{model.ScopeOAuthApplicationWrite, model.ScopeOAuthApplicationDelete}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewMemberPolicy(tt.scopes, nil).CanUpdateSpace(); got != tt.want {
				t.Errorf("CanUpdateSpace() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
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
		{name: "ゴミ箱復元", scopes: []model.Scope{model.ScopePageTrashDelete}},
		{name: "ページ編集", scopes: []model.Scope{model.ScopePageWrite}, updatePage: true},
		{name: "編集提案編集", scopes: []model.Scope{model.ScopeSuggestionWrite}, editSuggestion: true},
		{name: "編集提案反映", scopes: []model.Scope{model.ScopeSuggestionApplicationWrite}, applySuggestion: true},
		{name: "編集提案クローズ", scopes: []model.Scope{model.ScopeSuggestionClosureWrite}, closeSuggestion: true},
		{name: "管理者", scopes: model.SpaceRoleAdmin.Scopes(), showTrash: true, trashPage: true, updatePage: true, editSuggestion: true, applySuggestion: true, closeSuggestion: true},
		{name: "未知のスコープ", scopes: []model.Scope{"unknown:write", "page_trash:admin", "suggestion_application:read", "suggestion_closure:delete", "space:admin_extra"}},
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

func TestMemberPolicy_TokenManagement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                                               string
		scopes                                             []model.Scope
		showPAT, createPAT, deletePAT                      bool
		showOAuthGrant, createOAuthGrant, deleteOAuthGrant bool
	}{
		{name: "個人アクセストークンの閲覧", scopes: []model.Scope{model.ScopePersonalAccessTokenRead}, showPAT: true},
		{name: "個人アクセストークンの発行", scopes: []model.Scope{model.ScopePersonalAccessTokenWrite}, showPAT: true, createPAT: true},
		{name: "個人アクセストークンの失効", scopes: []model.Scope{model.ScopePersonalAccessTokenDelete}, showPAT: true, deletePAT: true},
		{name: "OAuthの連携の閲覧", scopes: []model.Scope{model.ScopeOAuthGrantRead}, showOAuthGrant: true},
		{name: "OAuthの連携の許可", scopes: []model.Scope{model.ScopeOAuthGrantWrite}, showOAuthGrant: true, createOAuthGrant: true},
		{name: "OAuthの連携の解除", scopes: []model.Scope{model.ScopeOAuthGrantDelete}, showOAuthGrant: true, deleteOAuthGrant: true},
		{name: "管理者", scopes: model.SpaceRoleAdmin.Scopes(), showPAT: true, createPAT: true, deletePAT: true, showOAuthGrant: true, createOAuthGrant: true, deleteOAuthGrant: true},
		// スペースを変更できるメンバーでも、トークンを管理するスコープは別に要る
		{name: "スペースの編集", scopes: []model.Scope{model.ScopeSpaceWrite, model.ScopeSpaceDelete, model.ScopePageWrite}},
		{name: "スコープなし"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewMemberPolicy(tt.scopes, nil)
			for _, check := range []struct {
				name      string
				got, want bool
			}{
				{"CanShowPersonalAccessTokens", p.CanShowPersonalAccessTokens(), tt.showPAT},
				{"CanCreatePersonalAccessToken", p.CanCreatePersonalAccessToken(), tt.createPAT},
				{"CanDeletePersonalAccessToken", p.CanDeletePersonalAccessToken(), tt.deletePAT},
				{"CanShowOAuthGrants", p.CanShowOAuthGrants(), tt.showOAuthGrant},
				{"CanCreateOAuthGrant", p.CanCreateOAuthGrant(), tt.createOAuthGrant},
				{"CanDeleteOAuthGrant", p.CanDeleteOAuthGrant(), tt.deleteOAuthGrant},
			} {
				if check.got != check.want {
					t.Errorf("%s = %v、期待値 = %v", check.name, check.got, check.want)
				}
			}
		})
	}
}

func TestMemberPolicy_OAuthApplication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                         string
		scopes                       []model.Scope
		show, create, update, delete bool
	}{
		{name: "閲覧", scopes: []model.Scope{model.ScopeOAuthApplicationRead}, show: true},
		{name: "登録・編集", scopes: []model.Scope{model.ScopeOAuthApplicationWrite}, show: true, create: true, update: true},
		{name: "削除", scopes: []model.Scope{model.ScopeOAuthApplicationDelete}, show: true, delete: true},
		{name: "管理者", scopes: model.SpaceRoleAdmin.Scopes(), show: true, create: true, update: true, delete: true},
		// スペースを変更できるメンバーや、連携を許可できるメンバーでも、アプリを管理するスコープは別に要る
		{name: "スペースの編集", scopes: []model.Scope{model.ScopeSpaceWrite, model.ScopeSpaceDelete}},
		{name: "OAuthの連携", scopes: []model.Scope{model.ScopeOAuthGrantWrite, model.ScopeOAuthGrantDelete}},
		{name: "スコープなし"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := NewMemberPolicy(tt.scopes, nil)
			for _, check := range []struct {
				name      string
				got, want bool
			}{
				{"CanShowOAuthApplications", p.CanShowOAuthApplications(), tt.show},
				{"CanCreateOAuthApplication", p.CanCreateOAuthApplication(), tt.create},
				{"CanUpdateOAuthApplication", p.CanUpdateOAuthApplication(), tt.update},
				{"CanDeleteOAuthApplication", p.CanDeleteOAuthApplication(), tt.delete},
			} {
				if check.got != check.want {
					t.Errorf("%s = %v、期待値 = %v", check.name, check.got, check.want)
				}
			}
		})
	}
}
