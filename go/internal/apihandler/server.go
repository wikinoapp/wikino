// Package apihandlerは公開Web APIのハンドラーをまとめる。
// oapi-codegenが生成する `apigen.StrictServerInterface` は全operationを1つのインターフェースにまとめるため、
// リソースごとのディレクトリに置いたHandlerを埋め込んだServerでこのインターフェースを満たす。
package apihandler

import (
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/attachment"
	"github.com/wikinoapp/wikino/go/internal/apihandler/current_space_member"
	"github.com/wikinoapp/wikino/go/internal/apihandler/current_token"
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page"
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_body_edit"
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_publication"
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_revision"
	"github.com/wikinoapp/wikino/go/internal/apihandler/openapi_description"
	"github.com/wikinoapp/wikino/go/internal/apihandler/page"
	"github.com/wikinoapp/wikino/go/internal/apihandler/page_backlink"
	"github.com/wikinoapp/wikino/go/internal/apihandler/page_body_edit"
	"github.com/wikinoapp/wikino/go/internal/apihandler/page_link"
	"github.com/wikinoapp/wikino/go/internal/apihandler/space"
	"github.com/wikinoapp/wikino/go/internal/apihandler/topic"
	"github.com/wikinoapp/wikino/go/internal/apihandler/wikilink_preview"
)

// リソースごとのHandlerはどれも `Handler` という名前のため、そのまま埋め込むとフィールド名が重複する。
// 別名を付けて埋め込み、別名をフィールド名にする
type (
	attachmentHandler           = attachment.Handler
	currentSpaceMemberHandler   = current_space_member.Handler
	currentTokenHandler         = current_token.Handler
	draftPageHandler            = draft_page.Handler
	draftPageBodyEditHandler    = draft_page_body_edit.Handler
	draftPagePublicationHandler = draft_page_publication.Handler
	draftPageRevisionHandler    = draft_page_revision.Handler
	openAPIDescriptionHandler   = openapi_description.Handler
	pageHandler                 = page.Handler
	pageBacklinkHandler         = page_backlink.Handler
	pageBodyEditHandler         = page_body_edit.Handler
	pageLinkHandler             = page_link.Handler
	spaceHandler                = space.Handler
	topicHandler                = topic.Handler
	wikilinkPreviewHandler      = wikilink_preview.Handler
)

// Serverはリソースごとのハンドラーを埋め込み、`apigen.StrictServerInterface` を満たす
type Server struct {
	*attachmentHandler
	*currentSpaceMemberHandler
	*currentTokenHandler
	*draftPageHandler
	*draftPageBodyEditHandler
	*draftPagePublicationHandler
	*draftPageRevisionHandler
	*openAPIDescriptionHandler
	*pageHandler
	*pageBacklinkHandler
	*pageBodyEditHandler
	*pageLinkHandler
	*spaceHandler
	*topicHandler
	*wikilinkPreviewHandler
}

var _ apigen.StrictServerInterface = (*Server)(nil)

// NewServerは新しいServerを作成する
func NewServer(
	attachment *attachment.Handler,
	currentSpaceMember *current_space_member.Handler,
	currentToken *current_token.Handler,
	draftPage *draft_page.Handler,
	draftPageBodyEdit *draft_page_body_edit.Handler,
	draftPagePublication *draft_page_publication.Handler,
	draftPageRevision *draft_page_revision.Handler,
	openAPIDescription *openapi_description.Handler,
	page *page.Handler,
	pageBacklink *page_backlink.Handler,
	pageBodyEdit *page_body_edit.Handler,
	pageLink *page_link.Handler,
	space *space.Handler,
	topic *topic.Handler,
	wikilinkPreview *wikilink_preview.Handler,
) *Server {
	return &Server{
		attachmentHandler:           attachment,
		currentSpaceMemberHandler:   currentSpaceMember,
		currentTokenHandler:         currentToken,
		draftPageHandler:            draftPage,
		draftPageBodyEditHandler:    draftPageBodyEdit,
		draftPagePublicationHandler: draftPagePublication,
		draftPageRevisionHandler:    draftPageRevision,
		openAPIDescriptionHandler:   openAPIDescription,
		pageHandler:                 page,
		pageBacklinkHandler:         pageBacklink,
		pageBodyEditHandler:         pageBodyEdit,
		pageLinkHandler:             pageLink,
		spaceHandler:                space,
		topicHandler:                topic,
		wikilinkPreviewHandler:      wikilinkPreview,
	}
}
