// Package httppaginationはHTTPページネーションパラメータを解析・検証する。
package httppagination

import (
	"errors"
	"math"
	"net/http"
	"strconv"
)

// ParsePageParamはクエリ文字列からオフセットページネーションのパラメータを読む。値が無い場合と
// 正の整数として読めない場合は1ページ目とし、壊れたクエリでも1ページ目を描画できるようにする。
//
// 2つ目の返り値は、上限として計算したSQL offset (page-1)*limitがページネーション
// クエリのint32パラメータに収まるかを表す。固定件数の一覧はその件数、ページごとに件数が変わる一覧は
// 最大件数を渡すため、実際のoffsetはこの上限を超えない。呼び出し元はUseCaseを呼ぶ前にHTTP境界で
// falseを拒否する。
//
// offsetに収まらない正の値は、桁数に関わらず丸めずに拒否する。それが指すのは存在しないページであり、
// 範囲内だが最終ページより後ろの値が指すものと同じだからである。後者は総ページ数を知る呼び出し元が
// 404にしている。
func ParsePageParam(r *http.Request, limit int32) (int32, bool) {
	return ParseNamedPageParam(r, "page", limit)
}

// ParseOptionalNumberParamはクエリ文字列から任意の正のint32を読む。ページネーション状態を
// どの一覧に適用するかを指す番号などが対象。パラメータが無い場合は0を返し、呼び出し元は
// 「選択されていない」と解釈する。
//
// ParsePageParamと違い、値があって解釈できない場合は無視せず拒否する。呼び出し元が満たせない選択を
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

// ParseNamedPageParamは、指定した名前のクエリパラメータにParsePageParamと同じ検証を
// 適用する。フルページのフォールバックでは独立した関連ページ一覧ごとに別のパラメータ名を使い、
// フラグメントエンドポイントでは従来どおり "page" を使う。
func ParseNamedPageParam(r *http.Request, name string, limit int32) (int32, bool) {
	pageStr := r.URL.Query().Get(name)
	if pageStr == "" {
		return 1, true
	}

	// オーバーフロー時、ParseIntは0ではなく丸めた境界値を返すため、int64に収まらない値も
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
