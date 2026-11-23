package i18n

import (
	"context"
	"math"
	"strconv"
	"strings"
)

// byteSizeUnitKeysは、FormatByteSizeが1024の累乗ごとに使う単位の翻訳キー
var byteSizeUnitKeys = []string{
	"byte_size_bytes",
	"byte_size_kilobytes",
	"byte_size_megabytes",
	"byte_size_gigabytes",
	"byte_size_terabytes",
}

// FormatByteSizeはバイト数を、1024の累乗の単位で、有効数字3桁までの読みやすい文字列にする
// (例: 1536 → 日本語は "1.5KB"、英語は "1.5 KB")。末尾の0は書かない
func FormatByteSize(ctx context.Context, byteSize int64) string {
	if byteSize < 1024 {
		return T(ctx, byteSizeUnitKeys[0], map[string]any{"Size": strconv.FormatInt(byteSize, 10), "Count": int(byteSize)})
	}

	value := float64(byteSize)
	unit := 0
	for value >= 1024 && unit < len(byteSizeUnitKeys)-1 {
		value /= 1024
		unit++
	}

	// 有効数字で丸めてから小数点以下の桁数を指定し、丸めで桁が増えても指数表記を使わない。
	const precision = 3
	decimalPlaces := precision - 1 - int(math.Floor(math.Log10(value)))
	scale := math.Pow10(decimalPlaces)
	value = math.Round(value*scale) / scale
	size := strconv.FormatFloat(value, 'f', max(decimalPlaces, 0), 64)
	if decimalPlaces > 0 {
		size = strings.TrimRight(strings.TrimRight(size, "0"), ".")
	}
	return T(ctx, byteSizeUnitKeys[unit], map[string]any{"Size": size})
}
