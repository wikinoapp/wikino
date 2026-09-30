package current_space_member

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetCurrentSpaceMemberはトークンの持ち主の、束縛先のスペースでのメンバーを返す (GET /api/v1/spaces/{space_identifier}/members/me)
func (h *Handler) GetCurrentSpaceMember(ctx context.Context, request apigen.GetCurrentSpaceMemberRequestObject) (apigen.GetCurrentSpaceMemberResponseObject, error) {
	output, err := h.getAPICurrentSpaceMemberUC.Execute(ctx, usecase.GetAPICurrentSpaceMemberInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
	})
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(string(output.SpaceMember.ID))
	if err != nil {
		return nil, fmt.Errorf("メンバーIDをUUIDとして解釈できません: %w", err)
	}

	return apigen.GetCurrentSpaceMember200JSONResponse{
		Body: apigen.SpaceMember{
			Id:     id,
			Atname: output.User.Atname,
			Name:   output.User.Name,
		},
	}, nil
}
