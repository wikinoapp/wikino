package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
)

// GetUserはトークンの持ち主のユーザーを返す (GET /api/v1/user)。
// 返すのはトークン認証で取得済みの主体のユーザーで、業務上の判定もデータの取得も無いため、UseCaseを経由しない
func (h *Handler) GetUser(ctx context.Context, _ apigen.GetUserRequestObject) (apigen.GetUserResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	if principal == nil {
		// operationのsecurityによりリクエスト検証でトークンの無い要求は401になるため、ここには来ない
		return nil, errors.New("公開APIの呼び出し主体がコンテキストにありません")
	}

	id, err := uuid.Parse(string(principal.User.ID))
	if err != nil {
		return nil, fmt.Errorf("ユーザーIDをUUIDとして解釈できません: %w", err)
	}

	return apigen.GetUser200JSONResponse{
		Body: apigen.User{
			Id:          id,
			Atname:      principal.User.Atname,
			Name:        principal.User.Name,
			Description: principal.User.Description,
		},
	}, nil
}
