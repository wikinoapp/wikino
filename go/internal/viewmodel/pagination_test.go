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
			name:        "1ページ目・16件中15件表示（次ページあり）",
			current:     1,
			totalCount:  16,
			limit:       15,
			wantTotal:   2,
			wantHasNext: true,
			wantHasPrev: false,
		},
		{
			name:        "2ページ目・16件中15件表示（前ページあり）",
			current:     2,
			totalCount:  16,
			limit:       15,
			wantTotal:   2,
			wantHasNext: false,
			wantHasPrev: true,
		},
		{
			name:        "2ページ目・45件中15件表示（前後ページあり）",
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
				t.Errorf("Current = %d, want %d", p.Current, tt.current)
			}
			if p.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", p.Total, tt.wantTotal)
			}
			if p.HasNext != tt.wantHasNext {
				t.Errorf("HasNext = %v, want %v", p.HasNext, tt.wantHasNext)
			}
			if p.HasPrevious != tt.wantHasPrev {
				t.Errorf("HasPrevious = %v, want %v", p.HasPrevious, tt.wantHasPrev)
			}
		})
	}
}
