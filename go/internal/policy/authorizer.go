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
	// CanUpdateTopicVisibilityはtopic_visibility:writeスコープで判定する。
	// 非公開トピックを公開すると内容がスペースの外へ出るため、トピックの名前や説明を
	// 変えられるtopic:writeとは分けている。
	CanUpdateTopicVisibility() bool
	// CanDeleteTopicはtopic:deleteスコープで判定する。トピックの削除はページごと消して
	// 元に戻せないため、トピックの名前や説明を変えられるtopic:writeとは分けている。
	CanDeleteTopic() bool

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

	// CanRestorePageはpage_trash:deleteスコープで判定し、ゴミ箱からページを戻す操作を許可する。
	CanRestorePage() bool

	// ページのピン留め

	// CanPinPageはpage_pin:writeスコープで、CanUnpinPageはpage_pin:deleteスコープで判定する。
	// ピン留めはトピックの一覧での見せ方を変える操作で、ページの内容を書く権限 (page:write) とは
	// 別の境界として扱う。ピンの状態はpage:readで見えるため、読み取りのスコープは持たない。
	CanPinPage() bool
	CanUnpinPage() bool

	// 下書きページ (所有者チェックパターン)
	// 下書きの参照・保存はページの公開 (page:write) と分けてdraft_page:read / draft_page:writeで判定する。
	// 下書きは書けるが公開はできないトークンを、エージェントに渡せるようにするためである。
	CanShowDraftPage() bool
	CanSaveDraftPage() bool
	// CanDeleteDraftPageはdraft_page:deleteスコープを持つかどうかのみで判定する。
	// 所有者チェックはUseCase側で「本人の下書きしか取得しない」ことで担保する。
	// adminが他メンバーの下書きを操作する経路は将来別UseCaseで実装する想定。
	CanDeleteDraftPage() bool

	// 添付ファイル

	// CanShowAttachmentは、attachment:readスコープを持つか、
	// 添付ファイルを参照するページがすべて公開トピックにある (allReferencingPagesPublic) かで判定する。
	// attachment:readはスペースにしか付けられないスコープで、判定でトピックを見ない。
	// そのため非公開トピックのページだけが参照する添付ファイルも、attachment:readがあれば閲覧できる。
	CanShowAttachment(allReferencingPagesPublic bool) bool
	// CanCreateAttachmentはattachment:writeスコープで判定する。
	// 添付ファイルはスペースに属し、ページを書く前にアップロードするため、判定でトピックを見ない。
	CanCreateAttachment() bool

	// CanDeleteAttachmentはattachment:deleteスコープで判定する。
	// 削除は取り消せず、ほかのメンバーがページに貼った添付ファイルも消すため、管理者のロールだけが持つ。
	CanDeleteAttachment() bool

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

	// CanExportSpaceはspace:writeスコープで判定する。
	// エクスポートはスペースの全ページを1つのアーカイブに収めるため、読める全員ではなく
	// スペースを変更してよいと認められたメンバーに開く。
	CanExportSpace() bool

	// CanUpdateSpaceはspace:writeスコープで判定する。
	// スペース設定の既存の項目 (一般・エクスポート・添付ファイル) はこの権限を持つメンバーにだけ出す。
	CanUpdateSpace() bool

	// CanDeleteSpaceはspace:deleteスコープで判定する。スペースの削除はトピック・ページ・添付ファイルごと
	// 消して元に戻せないため、スペースの名前や識別子を変えられるspace:writeとは分けている。
	CanDeleteSpace() bool

	// 個人用アクセストークン (所有者チェックパターン)
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
