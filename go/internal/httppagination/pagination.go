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
// The second return value reports whether the upper-bound SQL offset (page-1)*limit fits the int32
// parameter the paginated queries take. Uniform listings pass their page size; listings with
// variable page sizes pass their largest page size, so their resolved offset cannot exceed this bound. Callers
// reject false at the HTTP boundary before invoking the usecase.
//
// A positive value too large for the offset is rejected rather than clamped, whatever its
// magnitude: it names a page that does not exist, which is the same thing a value just inside the
// range but past the last page names, and those get a 404 from the callers that know the total.
//
// [Ja] ParsePageParam はクエリ文字列からオフセットページネーションのパラメータを読む。値が無い場合と
// 正の整数として読めない場合は 1 ページ目とし、壊れたクエリでも 1 ページ目を描画できるようにする。
//
// 2 つ目の返り値は、上限として計算した SQL offset (page-1)*limit がページネーション
// クエリの int32 パラメータに収まるかを表す。固定件数の一覧はその件数、ページごとに件数が変わる一覧は
// 最大件数を渡すため、実際の offset はこの上限を超えない。呼び出し元は UseCase を呼ぶ前に HTTP 境界で
// false を拒否する。
//
// offset に収まらない正の値は、桁数に関わらず丸めずに拒否する。それが指すのは存在しないページであり、
// 範囲内だが最終ページより後ろの値が指すものと同じだからである。後者は総ページ数を知る呼び出し元が
// 404 にしている。
func ParsePageParam(r *http.Request, limit int32) (int32, bool) {
	return ParseNamedPageParam(r, "page", limit)
}

// ParseOptionalNumberParam reads an optional positive int32 from the query string, such as the
// number identifying which listing a paginated related-page state applies to. An absent parameter
// yields zero, which callers read as "not selected".
//
// Unlike ParsePageParam, a present but unusable value is rejected rather than ignored: it names a
// selection the caller cannot honor, and silently acting on a different one would hide the mistake.
//
// [Ja] ParseOptionalNumberParam はクエリ文字列から任意の正の int32 を読む。ページネーション状態を
// どの一覧に適用するかを指す番号などが対象。パラメータが無い場合は 0 を返し、呼び出し元は
// 「選択されていない」と解釈する。
//
// ParsePageParam と違い、値があって解釈できない場合は無視せず拒否する。呼び出し元が満たせない選択を
// 指しており、黙って別のものを選ぶと誤りが隠れてしまうためである。
func ParseOptionalNumberParam(r *http.Request, name string) (int32, bool) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return 0, true
	}

	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil || number <= 0 {
		return 0, false
	}

	return int32(number), true
}

// ParseNamedPageParam applies the same validation as ParsePageParam to a named query parameter.
// Full-page fallbacks use distinct parameters for the independent related-page listings, while
// fragment endpoints keep using the conventional "page" parameter.
//
// [Ja] ParseNamedPageParam は、指定した名前のクエリパラメータに ParsePageParam と同じ検証を
// 適用する。フルページのフォールバックでは独立した関連ページ一覧ごとに別のパラメータ名を使い、
// フラグメントエンドポイントでは従来どおり "page" を使う。
func ParseNamedPageParam(r *http.Request, name string, limit int32) (int32, bool) {
	pageStr := r.URL.Query().Get(name)
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
