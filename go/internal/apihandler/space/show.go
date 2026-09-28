package space

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetSpaceはトークンを束縛したスペースを返す (GET /api/v1/spaces/{space_identifier})
func (h *Handler) GetSpace(ctx context.Context, request apigen.GetSpaceRequestObject) (apigen.GetSpaceResponseObject, error) {
	output, err := h.getAPISpaceUC.Execute(ctx, usecase.GetAPISpaceInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
	})
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(string(output.Space.ID))
	if err != nil {
		return nil, fmt.Errorf("スペースIDをUUIDとして解釈できません: %w", err)
	}

	return apigen.GetSpace200JSONResponse{
		Body: apigen.Space{
			Id:         id,
			Identifier: string(output.Space.Identifier),
			Name:       output.Space.Name,
		},
	}, nil
}
