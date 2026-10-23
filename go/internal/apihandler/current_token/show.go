package current_token

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// GetCurrentTokenはリクエストに使ったトークンの種類・スコープ・有効期限を返す (GET /api/v1/spaces/{space_identifier}/tokens/current)
func (h *Handler) GetCurrentToken(ctx context.Context, request apigen.GetCurrentTokenRequestObject) (apigen.GetCurrentTokenResponseObject, error) {
	output, err := h.getAPICurrentTokenUC.Execute(ctx, usecase.GetAPICurrentTokenInput{
		Principal:       middleware.APIPrincipalFromContext(ctx),
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
	})
	if err != nil {
		return nil, err
	}

	kind, err := tokenKind(output.Kind)
	if err != nil {
		return nil, err
	}
	scopes := make([]string, 0, len(output.Scopes))
	for _, s := range output.Scopes {
		scopes = append(scopes, string(s))
	}

	return apigen.GetCurrentToken200JSONResponse{
		Body: apigen.Token{
			Kind:      kind,
			Scopes:    scopes,
			ExpiresAt: output.ExpiresAt.UTC(),
		},
	}, nil
}

// tokenKindはトークンの種類をAPIの表現に変換する
func tokenKind(kind model.APITokenKind) (apigen.TokenKind, error) {
	switch kind {
	case model.APITokenKindPersonalAccessToken:
		return apigen.PersonalAccessToken, nil
	case model.APITokenKindOAuthAccessToken:
		return apigen.OauthAccessToken, nil
	default:
		return "", fmt.Errorf("未知のトークンの種類です: %s", kind)
	}
}
