// Package httppagination parses and validates HTTP pagination parameters.
//
// [Ja] Package httppagination は HTTP ページネーションパラメータを解析・検証する。
package httppagination

import (
	"errors"
	"math"
	"net/http"
	"strconv"
)

// ParsePageParam reads the offset pagination parameter from the query string. A missing value, or
// one that is not a positive integer, yields page 1, so that a malformed query still renders the
// first page.
//
// The second return value reports whether the SQL offset for that page fits the int32 parameter the
// paginated queries take. The repository computes the offset as (page-1)*limit in int32 arithmetic,
// so a larger page silently wraps to a negative offset and PostgreSQL rejects the query. Callers
// reject false at the HTTP boundary before invoking the usecase.
//
// A positive value too large for the offset is rejected rather than clamped, whatever its
// magnitude: it names a page that does not exist, which is the same thing a value just inside the
// range but past the last page names, and those get a 404 from the callers that know the total.
//
// [Ja] ParsePageParam はクエリ文字列からオフセットページネーションのパラメータを読む。値が無い場合と
// 正の整数として読めない場合は 1 ページ目とし、壊れたクエリでも 1 ページ目を描画できるようにする。
//
// 2 つ目の返り値は、そのページの SQL offset がページネーションクエリの int32 パラメータに収まるかを
// 表す。Repository は offset を int32 演算の (page-1)*limit で求めるため、これより大きいページでは
// 黙って負値に回り込み PostgreSQL がクエリを拒否する。呼び出し元は UseCase を呼ぶ前に HTTP 境界で
// false を拒否する。
//
// offset に収まらない正の値は、桁数に関わらず丸めずに拒否する。それが指すのは存在しないページであり、
// 範囲内だが最終ページより後ろの値が指すものと同じだからである。後者は総ページ数を知る呼び出し元が
// 404 にしている。
func ParsePageParam(r *http.Request, limit int32) (int32, bool) {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		return 1, true
	}

	// On overflow ParseInt returns the bound it clamped to instead of 0, so a value too large for
	// int64 still comes back positive and is told apart from a syntax error.
	//
	// [Ja] オーバーフロー時、ParseInt は 0 ではなく丸めた境界値を返すため、int64 に収まらない値も
	// 正の数として返り、構文エラーと区別できる。
	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 1, true
	}
	if page <= 0 {
		return 1, true
	}

	if page > math.MaxInt32 || (page-1)*int64(limit) > math.MaxInt32 {
		return 0, false
	}

	return int32(page), true
}
