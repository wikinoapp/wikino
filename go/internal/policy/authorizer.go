package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// Authorizerはリソースに対する権限を判定するインターフェース。
// MemberPolicy (スペースメンバー用) とGuestPolicy (非メンバー用) が実装する。
type Authorizer interface {
	// トピック
	CanShowTopic(topic *model.Topic) bool
	CanUpdateTopic() bool

	// ページ
	CanCreatePage() bool
	CanUpdatePage() bool

	// ゴミ箱

	// CanShowTrashはpage_trash:readスコープで判定する。
	// これにより、読み取り専用メンバーからゴミ箱内の内容を隠しつつ、ゴミ箱を
	// 開ける権限を持つメンバーは内容を確認できる。
	CanShowTrash() bool

	// CanTrashPageはpage_trash:writeスコープで判定し、ページをゴミ箱へ入れる操作とその後ゴミ箱を
	// 覗く操作もpage_trash:readの含意によって許可する。page:writeでは意図的に足りないものとする。ページを書き換えて
	// よい編集者が、そのページをスペースの可視な内容から外してよいとは限らないためである。
	CanTrashPage() bool

	// 下書きページ (所有者チェックパターン)
	CanShowDraftPage(isOwner bool) bool
	CanUpdateDraftPage(isOwner bool) bool
	// CanDeleteDraftPageはdraft_page:deleteスコープを持つかどうかのみで判定する。
	// 所有者チェックはUseCase側で「本人の下書きしか取得しない」ことで担保する。
	// adminが他メンバーの下書きを操作する経路は将来別UseCaseで実装する想定。
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

	// スペース
	CanCreateTopic() bool

	// CanExportSpaceはspace:writeスコープで判定し、Rails版が要求するものと揃える。
	// エクスポートはスペースの全ページを1つのアーカイブに収めるため、読める全員ではなく
	// スペースを変更してよいと認められたメンバーに開く。
	CanExportSpace() bool
}
