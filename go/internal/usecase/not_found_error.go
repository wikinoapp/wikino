package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// notFoundErrorは、求められたものに手が届かないときに返すエラーを組み立てる。
func notFoundError(ctx context.Context) error {
	return &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}
}
