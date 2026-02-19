// Package page_location はページロケーション関連のHTTPハンドラーを提供します
package page_location

import (
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// Handler はページロケーションハンドラー
type Handler struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
}

// NewHandler は新しいページロケーションハンドラーを作成します
func NewHandler(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
) *Handler {
	return &Handler{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
	}
}
