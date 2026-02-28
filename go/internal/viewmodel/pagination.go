package viewmodel

// Pagination はオフセットベースのページネーション情報です。
// フィールド名に "Page" を使わない（Wikinoのドメインモデル Page との混同を避けるため）。
type Pagination struct {
	Current     int
	Total       int
	HasNext     bool
	HasPrevious bool
}
