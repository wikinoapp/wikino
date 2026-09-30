// Package apihandlerは公開Web APIのハンドラーをまとめる。
// oapi-codegenが生成する `apigen.StrictServerInterface` は全operationを1つのインターフェースにまとめるため、
// リソースごとのディレクトリに置いたHandlerを埋め込んだServerでこのインターフェースを満たす。
package apihandler

import (
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/current_space_member"
	"github.com/wikinoapp/wikino/go/internal/apihandler/openapi_description"
	"github.com/wikinoapp/wikino/go/internal/apihandler/page"
	"github.com/wikinoapp/wikino/go/internal/apihandler/space"
	"github.com/wikinoapp/wikino/go/internal/apihandler/topic"
)

// リソースごとのHandlerはどれも `Handler` という名前のため、そのまま埋め込むとフィールド名が重複する。
// 別名を付けて埋め込み、別名をフィールド名にする
type (
	currentSpaceMemberHandler = current_space_member.Handler
	openAPIDescriptionHandler = openapi_description.Handler
	pageHandler               = page.Handler
	spaceHandler              = space.Handler
	topicHandler              = topic.Handler
)

// Serverはリソースごとのハンドラーを埋め込み、`apigen.StrictServerInterface` を満たす
type Server struct {
	*currentSpaceMemberHandler
	*openAPIDescriptionHandler
	*pageHandler
	*spaceHandler
	*topicHandler
}

var _ apigen.StrictServerInterface = (*Server)(nil)

// NewServerは新しいServerを作成する
func NewServer(
	currentSpaceMember *current_space_member.Handler,
	openAPIDescription *openapi_description.Handler,
	page *page.Handler,
	space *space.Handler,
	topic *topic.Handler,
) *Server {
	return &Server{
		currentSpaceMemberHandler: currentSpaceMember,
		openAPIDescriptionHandler: openAPIDescription,
		pageHandler:               page,
		spaceHandler:              space,
		topicHandler:              topic,
	}
}
