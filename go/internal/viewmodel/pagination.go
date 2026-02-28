package viewmodel

// Pagination はオフセットベースのページネーション情報です。
// フィールド名に "Page" を使わない（Wikinoのドメインモデル Page との混同を避けるため）。
type Pagination struct {
	Current     int
	Total       int
	HasNext     bool
	HasPrevious bool
}

// NewPagination は現在ページ・総件数・1ページあたりの件数からPaginationを生成する
func NewPagination(current int, totalCount int64, limit int) Pagination {
	total := int((totalCount + int64(limit) - 1) / int64(limit))
	if total < 1 {
		total = 1
	}

	return Pagination{
		Current:     current,
		Total:       total,
		HasNext:     current < total,
		HasPrevious: current > 1,
	}
}
