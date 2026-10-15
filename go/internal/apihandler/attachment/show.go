package attachment

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetAttachmentはトークンが属するスペースの添付ファイルの情報と、中身をダウンロードする署名付きURLを返す
// (GET /api/v1/spaces/{space_identifier}/attachments/{attachment_id})
func (h *Handler) GetAttachment(ctx context.Context, request apigen.GetAttachmentRequestObject) (apigen.GetAttachmentResponseObject, error) {
	output, err := h.getAPIAttachmentUC.Execute(ctx, usecase.GetAPIAttachmentInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		AttachmentID:    model.AttachmentID(request.AttachmentId.String()),
	})
	if err != nil {
		return nil, err
	}

	attachment, err := toAPIAttachment(output)
	if err != nil {
		return nil, err
	}
	return apigen.GetAttachment200JSONResponse{Body: attachment}, nil
}
