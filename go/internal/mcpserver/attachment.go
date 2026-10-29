package mcpserver

import (
	"context"
	"encoding/json"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// attachmentToolNameは、画像の中身も返す添付ファイルのツール (operationId)
const attachmentToolName = "getAttachment"

// attachmentImageContentTypesは、ツールの結果に中身を入れる画像の種類。主なMCPクライアントが画像として
// モデルに渡せる種類に限る
var attachmentImageContentTypes = []string{"image/png", "image/jpeg", "image/gif", "image/webp"}

// attachmentImageMaxBytesは、ツールの結果に中身を入れる画像の大きさの上限 (512KiB)。
// Base64化で画像が約4/3倍になることと、ファイルの情報も返すことを考慮し、結果の送信量を抑えるために設ける。
// 上限を超える画像は署名付きURLだけを返す
const attachmentImageMaxBytes = 512 << 10

// attachmentContentReaderは、添付ファイルの中身を読むユースケース。
// usecase.GetAPIAttachmentContentUsecaseがこのインターフェースを満たす
type attachmentContentReader interface {
	Execute(ctx context.Context, input usecase.GetAPIAttachmentContentInput) (*usecase.GetAPIAttachmentContentOutput, error)
}

// addAttachmentImageは、添付ファイルのツールの成功の結果resultに、画像で上限以下なら中身を画像として加える。
// 結果は、画像 (`ImageContent`) と、公開APIの応答のファイルの情報のテキストの順に並べる。
// 画像でないファイル・上限を超える画像・中身を読めない場合は、公開APIの応答の結果のまま返す
func (s *Server) addAttachmentImage(ctx context.Context, principal *model.APIPrincipal, arguments json.RawMessage, result *mcp.CallToolResult) {
	// この処理もSDKのgoroutineで動くため、画像の取得のpanicはここで回復する。
	// 画像は取得できた後にだけ結果へ加え、回復した場合も元の公開APIの結果を保つ
	defer func() {
		if rvr := recover(); rvr != nil {
			s.logger.ErrorContext(ctx, "MCPのツールで添付ファイルの画像の取得中にpanicしました",
				"operation_id", attachmentToolName, "panic", rvr, "stack", string(debug.Stack()))
		}
	}()

	var args struct {
		AttachmentID string `json:"attachment_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil || args.AttachmentID == "" {
		return
	}

	output, err := s.attachmentContents.Execute(ctx, usecase.GetAPIAttachmentContentInput{
		Principal:       principal,
		SpaceIdentifier: principal.Space.Identifier,
		AttachmentID:    model.AttachmentID(args.AttachmentID),
		ContentTypes:    attachmentImageContentTypes,
		MaxBytes:        attachmentImageMaxBytes,
	})
	if err != nil {
		// 公開APIの応答 (署名付きURL) は返せているため、中身を読めなくてもツールの呼び出しは失敗させない。
		// 見つからない・見る権限が無い (*model.AppError) のは、公開APIを呼んだ後に権限が変わった場合などの
		// 想定内の結果のため、障害としてログに残さない
		if model.AsAppError(err) == nil {
			s.logger.ErrorContext(ctx, "MCPのツールで添付ファイルの中身の取得に失敗しました", "error", err, "attachment_id", args.AttachmentID)
		}
		return
	}
	if output.Content == nil {
		return
	}

	image := &mcp.ImageContent{Data: output.Content, MIMEType: output.ContentType}
	result.Content = append([]mcp.Content{image}, result.Content...)
}
