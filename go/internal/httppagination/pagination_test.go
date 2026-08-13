package httppagination_test

import (
	"net/http/httptest"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/httppagination"
)

// TestParsePageParam pins both halves of the contract: how a value that is not a page number is
// coerced, and where the offset stops fitting the int32 parameter the paginated queries take.
//
// With a limit of 15, page 143165577 is the last one whose offset fits (143165576 * 15 =
// 2147483640), and page 143165578 is the first one that does not (143165577 * 15 = 2147483655).
//
// [Ja] TestParsePageParam は契約の両面を固定する。不正な値をどう丸めるかと、offset が
// ページネーションクエリの int32 パラメータに収まらなくなる境界である。
//
// 上限 15 件のとき、offset が収まる最後のページは 143165577 (143165576 * 15 = 2147483640) で、
// 収まらない最初のページは 143165578 (143165577 * 15 = 2147483655) になる。
func TestParsePageParam(t *testing.T) {
	t.Parallel()

	const limit int32 = 15

	tests := []struct {
		name     string
		query    string
		wantPage int32
		wantOK   bool
	}{
		{name: "パラメータが無ければ 1 ページ目", query: "", wantPage: 1, wantOK: true},
		{name: "空文字列は 1 ページ目", query: "?page=", wantPage: 1, wantOK: true},
		{name: "数値でない値は 1 ページ目", query: "?page=abc", wantPage: 1, wantOK: true},
		{name: "0 は 1 ページ目", query: "?page=0", wantPage: 1, wantOK: true},
		{name: "負の値は 1 ページ目", query: "?page=-3", wantPage: 1, wantOK: true},
		// A positive value past the offset limit names a page that does not exist, so it is rejected
		// however many digits it has rather than falling back to the first page.
		//
		// [Ja] offset の上限を超える正の値が指すのは存在しないページのため、桁数に関わらず 1 ページ目へ
		// フォールバックせず拒否する。
		{name: "int32 を超える値は拒否する", query: "?page=2147483648", wantPage: 0, wantOK: false},
		{name: "int64 も超える桁数の値は拒否する", query: "?page=99999999999999999999", wantPage: 0, wantOK: false},
		// A negative value is not a page number at all, so it keeps falling back to the first page
		// even when its magnitude overflows int64.
		//
		// [Ja] 負の値はそもそもページ番号ではないため、桁数が int64 を超えても 1 ページ目への
		// フォールバックを保つ。
		{name: "int64 を超える負の値は 1 ページ目", query: "?page=-99999999999999999999", wantPage: 1, wantOK: true},
		{name: "正の値はそのまま返す", query: "?page=2", wantPage: 2, wantOK: true},
		{name: "offset が int32 に収まる最大のページ", query: "?page=143165577", wantPage: 143165577, wantOK: true},
		{name: "offset が int32 に収まらないページは拒否する", query: "?page=143165578", wantPage: 0, wantOK: false},
		{name: "int32 の上限値は拒否する", query: "?page=2147483647", wantPage: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest("GET", "/"+tt.query, nil)

			page, ok := httppagination.ParsePageParam(r, limit)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if page != tt.wantPage {
				t.Errorf("page = %d, want %d", page, tt.wantPage)
			}
		})
	}
}

// The offsets a caller is allowed through must survive the int32 arithmetic the repository uses, or
// they wrap to a negative offset and PostgreSQL rejects the query.
//
// [Ja] 通過を許すページの offset は、Repository が使う int32 演算を通っても壊れてはならない。
// 壊れると負の offset に回り込み、PostgreSQL がクエリを拒否する。
func TestParsePageParam_AcceptedOffsetsDoNotOverflow(t *testing.T) {
	t.Parallel()

	for _, limit := range []int32{13, 14, 15, 100} {
		for _, query := range []string{"?page=1", "?page=2", "?page=143165577", "?page=2147483647"} {
			r := httptest.NewRequest("GET", "/"+query, nil)

			page, ok := httppagination.ParsePageParam(r, limit)
			if !ok {
				continue
			}

			if offset := (page - 1) * limit; offset < 0 {
				t.Errorf("limit=%d %s: offset = %d, want >= 0", limit, query, offset)
			}
		}
	}
}

// TestParseNamedPageParam verifies that an independent listing can use its own query parameter
// without changing the validation contract.
//
// [Ja] TestParseNamedPageParam は、独立した一覧が検証契約を変えずに固有のクエリパラメータを
// 使えることを確認する。
func TestParseNamedPageParam(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest("GET", "/?links_page=3&page=9", nil)
	page, ok := httppagination.ParseNamedPageParam(r, "links_page", 15)
	if !ok {
		t.Fatal("ParseNamedPageParam() ok = false, want true")
	}
	if page != 3 {
		t.Errorf("ParseNamedPageParam() page = %d, want 3", page)
	}
}
