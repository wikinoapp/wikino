package i18n_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
)

func TestFormatByteSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locale   string
		byteSize int64
		want     string
	}{
		{name: "日本語で1024未満はバイトで書く", locale: i18n.LangJa, byteSize: 512, want: "512バイト"},
		{name: "英語で1024未満はBytesで書く", locale: i18n.LangEn, byteSize: 0, want: "0 Bytes"},
		{name: "英語で1バイトは単数形で書く", locale: i18n.LangEn, byteSize: 1, want: "1 Byte"},
		{name: "英語で2バイトは複数形で書く", locale: i18n.LangEn, byteSize: 2, want: "2 Bytes"},
		{name: "日本語で1バイトを書く", locale: i18n.LangJa, byteSize: 1, want: "1バイト"},
		{name: "KB未満のバイト数には桁区切りを付けない", locale: i18n.LangEn, byteSize: 1023, want: "1023 Bytes"},
		{name: "ちょうど1024はKBの1で書く", locale: i18n.LangJa, byteSize: 1024, want: "1KB"},
		{name: "小数第3位の5を切り上げる", locale: i18n.LangEn, byteSize: 1152, want: "1.13 KB"},
		{name: "小数第2位の5を切り上げる", locale: i18n.LangEn, byteSize: 10496, want: "10.3 KB"},
		{name: "小数第1位の5を切り上げる", locale: i18n.LangEn, byteSize: 102912, want: "101 KB"},
		{name: "有効数字3桁に丸める", locale: i18n.LangJa, byteSize: 1500, want: "1.46KB"},
		{name: "末尾の0は書かない", locale: i18n.LangEn, byteSize: 1536, want: "1.5 KB"},
		{name: "MBで書く", locale: i18n.LangJa, byteSize: 12345678, want: "11.8MB"},
		{name: "丸めて1000になる値も指数表記にしない", locale: i18n.LangEn, byteSize: 1023999, want: "1000 KB"},
		{name: "1000以上でも有効数字3桁に丸める", locale: i18n.LangJa, byteSize: 1023 * 1024, want: "1020KB"},
		{name: "MBの直前はKBの有効数字3桁で書く", locale: i18n.LangEn, byteSize: 1048575, want: "1020 KB"},
		{name: "ちょうど1MBで単位を切り替える", locale: i18n.LangEn, byteSize: 1048576, want: "1 MB"},
		{name: "GBの直前はMBの有効数字3桁で書く", locale: i18n.LangEn, byteSize: 1073741823, want: "1020 MB"},
		{name: "ちょうど1GBで単位を切り替える", locale: i18n.LangEn, byteSize: 1073741824, want: "1 GB"},
		{name: "GBで書く", locale: i18n.LangJa, byteSize: 3 * 1024 * 1024 * 1024, want: "3GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), tt.locale)
			if got := i18n.FormatByteSize(ctx, tt.byteSize); got != tt.want {
				t.Errorf("FormatByteSize(%d) = %q、期待値 = %q", tt.byteSize, got, tt.want)
			}
		})
	}
}
