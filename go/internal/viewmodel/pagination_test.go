package viewmodel_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewPagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		current     int
		totalCount  int64
		limit       int
		wantTotal   int
		wantHasNext bool
		wantHasPrev bool
	}{
		{
			name:        "1ページ目・15件中15件表示",
			current:     1,
			totalCount:  15,
			limit:       15,
			wantTotal:   1,
			wantHasNext: false,
			wantHasPrev: false,
		},
		{
			name:        "1ページ目・16件中15件表示 (次ページあり)",
			current:     1,
			totalCount:  16,
			limit:       15,
			wantTotal:   2,
			wantHasNext: true,
			wantHasPrev: false,
		},
		{
			name:        "2ページ目・16件中15件表示 (前ページあり)",
			current:     2,
			totalCount:  16,
			limit:       15,
			wantTotal:   2,
			wantHasNext: false,
			wantHasPrev: true,
		},
		{
			name:        "2ページ目・45件中15件表示 (前後ページあり)",
			current:     2,
			totalCount:  45,
			limit:       15,
			wantTotal:   3,
			wantHasNext: true,
			wantHasPrev: true,
		},
		{
			name:        "0件の場合はトータル1ページ",
			current:     1,
			totalCount:  0,
			limit:       15,
			wantTotal:   1,
			wantHasNext: false,
			wantHasPrev: false,
		},
		{
			name:        "30件ちょうどは2ページ",
			current:     1,
			totalCount:  30,
			limit:       15,
			wantTotal:   2,
			wantHasNext: true,
			wantHasPrev: false,
		},
		{
			name:        "14件で14件/ページは1ページ",
			current:     1,
			totalCount:  14,
			limit:       14,
			wantTotal:   1,
			wantHasNext: false,
			wantHasPrev: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := viewmodel.NewPagination(tt.current, tt.totalCount, tt.limit)

			if p.Current != tt.current {
				t.Errorf("Current = %d、期待値 = %d", p.Current, tt.current)
			}
			if p.Total != tt.wantTotal {
				t.Errorf("Total = %d、期待値 = %d", p.Total, tt.wantTotal)
			}
			if p.HasNext != tt.wantHasNext {
				t.Errorf("HasNext = %v、期待値 = %v", p.HasNext, tt.wantHasNext)
			}
			if p.HasPrevious != tt.wantHasPrev {
				t.Errorf("HasPrevious = %v、期待値 = %v", p.HasPrevious, tt.wantHasPrev)
			}
		})
	}
}

func TestNewRelatedPagePagination_CumulativeLimit(t *testing.T) {
	t.Parallel()

	const cumulativePageLimit int32 = 10
	totalCount := int64((cumulativePageLimit + 1) * viewmodel.LinkLimit)
	editorState := viewmodel.PageLinkState{Context: viewmodel.PageLinkContextEdit}
	publicState := viewmodel.PageLinkState{Context: viewmodel.PageLinkContextShow}
	paginatedEditorState := viewmodel.PageLinkState{Context: viewmodel.PageLinkContextEditPaginated}

	editorPagination, editorCapped := viewmodel.NewRelatedPagePagination(
		cumulativePageLimit,
		totalCount,
		viewmodel.LinkLimit,
		editorState,
		cumulativePageLimit,
	)
	if editorPagination.HasNext {
		t.Error("エディタのページネーションが累積取得の上限で止まっていない")
	}
	if !editorCapped {
		t.Error("エディタのページネーションが次のページを保留していることを示していない")
	}
	if editorPagination.Total != int(cumulativePageLimit+1) {
		t.Errorf("エディタのページネーションの合計 = %d、期待値 = %d", editorPagination.Total, cumulativePageLimit+1)
	}

	publicPagination, publicCapped := viewmodel.NewRelatedPagePagination(
		cumulativePageLimit,
		totalCount,
		viewmodel.LinkLimit,
		publicState,
		cumulativePageLimit,
	)
	if !publicPagination.HasNext {
		t.Error("公開ページのページネーションがエディタ専用の累積取得の上限を超えて使えない")
	}
	if publicCapped {
		t.Error("公開ページのページネーションが次のページの保留を示している")
	}

	paginatedEditorPagination, paginatedEditorCapped := viewmodel.NewRelatedPagePagination(
		cumulativePageLimit,
		totalCount,
		viewmodel.LinkLimit,
		paginatedEditorState,
		cumulativePageLimit,
	)
	if !paginatedEditorPagination.HasNext {
		t.Error("1ページずつのエディタのページネーションが累積取得の上限を超えて続いていない")
	}
	if paginatedEditorCapped {
		t.Error("1ページずつのエディタのページネーションが次のページの保留を示している")
	}

	// 上限に達しても後ろに何も残っていない場合は打ち切りではなく通常の終端のため、一覧が
	// 続きがあるかのように示してはならない。
	exactPagination, exactCapped := viewmodel.NewRelatedPagePagination(
		cumulativePageLimit,
		int64(viewmodel.LinkLimit+(cumulativePageLimit-1)*viewmodel.RelatedPageFollowingLimit),
		viewmodel.LinkLimit,
		editorState,
		cumulativePageLimit,
	)
	if exactPagination.HasNext {
		t.Error("ちょうど上限で終わる一覧に次のページがある")
	}
	if exactCapped {
		t.Error("ちょうど上限で終わる一覧が残りを保留している")
	}
}

func TestPageLinkState_WithinCumulativeLimit(t *testing.T) {
	t.Parallel()
	const cumulativePageLimit int32 = 10

	tests := []struct {
		name  string
		state viewmodel.PageLinkState
		want  bool
	}{
		{
			name: "上限ちょうどのエディタ",
			state: viewmodel.PageLinkState{
				Context:            viewmodel.PageLinkContextEdit,
				LinkPage:           cumulativePageLimit,
				LinkedBacklinkPage: cumulativePageLimit,
				PageBacklinkPage:   cumulativePageLimit,
			},
			want: true,
		},
		{
			name: "上限を超えたエディタ",
			state: viewmodel.PageLinkState{
				Context:  viewmodel.PageLinkContextEdit,
				LinkPage: cumulativePageLimit + 1,
			},
			want: false,
		},
		{
			name: "累積取得の上限を超えたページ単位のエディタ",
			state: viewmodel.PageLinkState{
				Context:  viewmodel.PageLinkContextEditPaginated,
				LinkPage: cumulativePageLimit + 1,
			},
			want: true,
		},
		{
			name: "エディタの上限を超えた公開ページ",
			state: viewmodel.PageLinkState{
				Context:  viewmodel.PageLinkContextShow,
				LinkPage: cumulativePageLimit + 1,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.state.WithinCumulativeLimit(cumulativePageLimit); got != tt.want {
				t.Errorf("WithinCumulativeLimit() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

func TestPageLinkContext_PaginationModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		value                   string
		want                    viewmodel.PageLinkContext
		wantEdit                bool
		wantPrecedingPages      bool
		wantCumulativePageLimit int32
	}{
		{
			name:                    "累積取得のエディタ",
			value:                   string(viewmodel.PageLinkContextEdit),
			want:                    viewmodel.PageLinkContextEdit,
			wantEdit:                true,
			wantPrecedingPages:      true,
			wantCumulativePageLimit: 10,
		},
		{
			name:                    "1ページずつのエディタ",
			value:                   string(viewmodel.PageLinkContextEditPaginated),
			want:                    viewmodel.PageLinkContextEditPaginated,
			wantEdit:                true,
			wantPrecedingPages:      false,
			wantCumulativePageLimit: 0,
		},
		{
			name:                    "公開ページ",
			value:                   string(viewmodel.PageLinkContextShow),
			want:                    viewmodel.PageLinkContextShow,
			wantEdit:                false,
			wantPrecedingPages:      false,
			wantCumulativePageLimit: 0,
		},
		{
			name:                    "未知の値は安全側の累積取得のエディタになる",
			value:                   "unknown",
			want:                    viewmodel.PageLinkContextEdit,
			wantEdit:                true,
			wantPrecedingPages:      true,
			wantCumulativePageLimit: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			context := viewmodel.NormalizePageLinkContext(tt.value)
			if context != tt.want {
				t.Errorf("NormalizePageLinkContext() = %q、期待値 = %q", context, tt.want)
			}
			if got := context.IsEdit(); got != tt.wantEdit {
				t.Errorf("IsEdit() = %v、期待値 = %v", got, tt.wantEdit)
			}
			if got := context.IncludesPrecedingPages(); got != tt.wantPrecedingPages {
				t.Errorf("IncludesPrecedingPages() = %v、期待値 = %v", got, tt.wantPrecedingPages)
			}
			state := viewmodel.PageLinkState{Context: context}
			if got := state.CumulativePageLimit(10); got != tt.wantCumulativePageLimit {
				t.Errorf("CumulativePageLimit() = %d、期待値 = %d", got, tt.wantCumulativePageLimit)
			}
		})
	}
}

func TestRelatedPageBoundaries(t *testing.T) {
	t.Parallel()

	totalTests := []struct {
		totalCount int64
		want       int
	}{
		{totalCount: 0, want: 1},
		{totalCount: 14, want: 1},
		{totalCount: 15, want: 2},
		{totalCount: 29, want: 2},
		{totalCount: 30, want: 3},
		{totalCount: 44, want: 3},
		{totalCount: 45, want: 4},
	}
	for _, tt := range totalTests {
		if got := viewmodel.RelatedPageTotalPages(tt.totalCount, viewmodel.LinkLimit); got != tt.want {
			t.Errorf("RelatedPageTotalPages(%d) = %d、期待値 = %d", tt.totalCount, got, tt.want)
		}
	}

	indexTests := []struct {
		index int
		want  int32
	}{
		{index: 0, want: 1},
		{index: 13, want: 1},
		{index: 14, want: 2},
		{index: 28, want: 2},
		{index: 29, want: 3},
		{index: 43, want: 3},
		{index: 44, want: 4},
	}
	for _, tt := range indexTests {
		if got := viewmodel.RelatedPageNumberForIndex(tt.index, viewmodel.LinkLimit); got != tt.want {
			t.Errorf("RelatedPageNumberForIndex(%d) = %d、期待値 = %d", tt.index, got, tt.want)
		}
	}
}

// TestRelatedPageLimitsFitTheGridは、3つの関連ページ一覧の件数が満たすために選ばれている性質を
// 固定する。「もっと見る」のタイルはカードと同じ3カラムグリッドのもう1マスであり、読み込み済み
// カード数を3で割った余りが2である間だけ、埋まった行の末尾に来る。初回の件数がこれを満たし、
// 後続ページが行単位で追記することで、追記のたびに保たれる。
//
// 後続の件数は、一覧が返しうる最大のページ件数でもある必要がある。Handlerは、SQL offsetがint32に
// 収まらないページをこの件数だけを使って拒否しているためである。
func TestRelatedPageLimitsFitTheGrid(t *testing.T) {
	t.Parallel()

	const columns = 3

	for _, tt := range []struct {
		name  string
		limit int32
	}{
		{name: "LinkLimit", limit: viewmodel.LinkLimit},
		{name: "BacklinkLimit", limit: viewmodel.BacklinkLimit},
		{name: "PageBacklinkLimit", limit: viewmodel.PageBacklinkLimit},
	} {
		if tt.limit%columns != columns-1 {
			t.Errorf("%s = %dのため、さらに読み込むタイルで行が埋まらない: 期待値 = %d %% %d == %d", tt.name, tt.limit, tt.limit, columns, columns-1)
		}
		if viewmodel.RelatedPageFollowingLimit < tt.limit {
			t.Errorf("RelatedPageFollowingLimit = %dが%s = %dより小さく、オフセットの上限の検査対象となる最大のページサイズになっていない", viewmodel.RelatedPageFollowingLimit, tt.name, tt.limit)
		}
	}

	if viewmodel.RelatedPageFollowingLimit%columns != 0 {
		t.Errorf("RelatedPageFollowingLimit = %dでは行単位で追加されない: 期待値 = %d %% %d == 0", viewmodel.RelatedPageFollowingLimit, viewmodel.RelatedPageFollowingLimit, columns)
	}
}
