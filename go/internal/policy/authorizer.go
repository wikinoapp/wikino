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
