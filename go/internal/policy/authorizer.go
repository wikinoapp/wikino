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

	// CanUpdateSpaceはspace:writeスコープで判定し、Rails版のcan_update_space?と揃える。
	// スペース設定の既存の項目 (一般・エクスポート・添付ファイル・削除) はこの権限を持つメンバーにだけ出す。
	CanUpdateSpace() bool

	// 個人アクセストークン (所有者チェックパターン)
	// 一覧・失効の対象は自分のトークンに限る。所有者チェックはUseCase側で
	// 「本人のトークンしか取得しない」ことで担保する。
	CanShowPersonalAccessTokens() bool
	// CanCreatePersonalAccessTokenはpersonal_access_token:writeスコープで判定する。
	// 発行時だけでなく、トークン認証のたびに持ち主がトークンを使い続けてよいかもこれで判定する。
	CanCreatePersonalAccessToken() bool
	CanDeletePersonalAccessToken() bool

	// OAuthの連携 (所有者チェックパターン)
	// 一覧・解除の対象は自分が許可した連携に限る。所有者チェックはUseCase側で担保する。
	CanShowOAuthGrants() bool
	// CanCreateOAuthGrantはoauth_grant:writeスコープで判定する。
	// 許可時だけでなく、トークン認証のたびに持ち主がOAuthのトークンを使い続けてよいかもこれで判定する。
	CanCreateOAuthGrant() bool
	CanDeleteOAuthGrant() bool

	// OAuthアプリ
	// トークン管理とは違い、自分が作成したものに限らずスペースのアプリ全体を対象にする。
	CanShowOAuthApplications() bool
	// CanCreateOAuthApplicationとCanUpdateOAuthApplicationはoauth_application:writeスコープで判定する。
	// シークレットの再発行も編集として扱い、CanUpdateOAuthApplicationで判定する。
	CanCreateOAuthApplication() bool
	CanUpdateOAuthApplication() bool
	CanDeleteOAuthApplication() bool
}
