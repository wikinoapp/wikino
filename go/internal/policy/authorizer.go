package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Authorizer はリソースに対する権限を判定するインターフェース。
// MemberPolicy（スペースメンバー用）と GuestPolicy（非メンバー用）が実装する。
type Authorizer interface {
	// トピック
	CanShowTopic(topic *model.Topic) bool
	CanUpdateTopic() bool

	// ページ
	CanCreatePage() bool
	CanUpdatePage() bool

	// 下書きページ（所有者チェックパターン）
	CanShowDraftPage(isOwner bool) bool
	CanUpdateDraftPage(isOwner bool) bool
	// CanDeleteDraftPage は draft_page:delete スコープを持つかどうかのみで判定する。
	// 所有者チェックは UseCase 側で「本人の下書きしか取得しない」ことで担保する。
	// admin が他メンバーの下書きを操作する経路は将来別 UseCase で実装する想定。
	CanDeleteDraftPage() bool

	// 編集提案
	CanCreateSuggestion(topic *model.Topic) bool
	CanApplySuggestion() bool
	CanCloseSuggestion(isCreator bool) bool
	CanUpdateSuggestion(suggestion *model.Suggestion) bool
	CanAddSuggestionPage(suggestion *model.Suggestion) bool
	CanRemoveSuggestionPage(suggestion *model.Suggestion) bool
	CanEditSuggestionPage() bool

	// 編集提案コメント
	CanCreateSuggestionComment() bool
	CanUpdateSuggestionComment(suggestion *model.Suggestion) bool
}
