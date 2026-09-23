package repository

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// PaginatedPagesはページネーション付きのページ一覧です
type PaginatedPages struct {
	Pages      []*model.Page
	TotalCount int64
}
