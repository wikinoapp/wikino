package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Authorizer is an interface that determines permissions for resources.
// It is implemented by MemberPolicy (for space members) and GuestPolicy (for non-members).
//
// [Ja] Authorizer はリソースに対する権限を判定するインターフェース。
// MemberPolicy (スペースメンバー用) と GuestPolicy (非メンバー用) が実装する。
type Authorizer interface {
	// Topic
	// [Ja] トピック
	CanShowTopic(topic *model.Topic) bool
	CanUpdateTopic() bool

	// Page
	// [Ja] ページ
	CanCreatePage() bool
	CanUpdatePage() bool

	// Trash
	// [Ja] ゴミ箱

	// CanShowTrash decides based on the page:trash scope, not page:read.
	// This keeps trashed content hidden from read-only members while allowing members
	// authorized to open the trash to inspect it.
	//
	// [Ja] CanShowTrash は page:read ではなく page:trash スコープで判定する。
	// これにより、読み取り専用メンバーからゴミ箱内の内容を隠しつつ、ゴミ箱を
	// 開ける権限を持つメンバーは内容を確認できる。
	CanShowTrash() bool

	// CanTrashPage decides based on the page:trash scope, so that moving a page into the trash and
	// looking into the trash afterwards stay on the same permission axis. page:write is deliberately
	// not enough: an editor who may rewrite a page is not thereby allowed to take it out of the
	// space's visible content.
	//
	// [Ja] CanTrashPage は page:trash スコープで判定し、ページをゴミ箱へ入れる操作とその後ゴミ箱を
	// 覗く操作を同じ権限軸に揃える。page:write では意図的に足りないものとする。ページを書き換えて
	// よい編集者が、そのページをスペースの可視な内容から外してよいとは限らないためである。
	CanTrashPage() bool

	// Draft page (owner-check pattern)
	// [Ja] 下書きページ (所有者チェックパターン)
	CanShowDraftPage(isOwner bool) bool
	CanUpdateDraftPage(isOwner bool) bool
	// CanDeleteDraftPage decides based only on whether the draft_page:delete scope is held.
	// The owner check is ensured on the UseCase side by only ever retrieving the user's own drafts.
	// A path for an admin to operate on other members' drafts is planned to be implemented in a separate UseCase later.
	//
	// [Ja] CanDeleteDraftPage は draft_page:delete スコープを持つかどうかのみで判定する。
	// 所有者チェックは UseCase 側で「本人の下書きしか取得しない」ことで担保する。
	// admin が他メンバーの下書きを操作する経路は将来別 UseCase で実装する想定。
	CanDeleteDraftPage() bool

	// Suggestion
	// [Ja] 編集提案
	CanCreateSuggestion(topic *model.Topic) bool
	CanApplySuggestion() bool
	CanCloseSuggestion(isCreator bool) bool
	CanUpdateSuggestion(suggestion *model.Suggestion) bool
	CanAddSuggestionPage(suggestion *model.Suggestion) bool
	CanRemoveSuggestionPage(suggestion *model.Suggestion) bool
	CanEditSuggestionPage() bool

	// Suggestion comment
	// [Ja] 編集提案コメント
	CanCreateSuggestionComment() bool
	CanUpdateSuggestionComment(suggestion *model.Suggestion) bool

	// Space
	// [Ja] スペース
	CanCreateTopic() bool
}
