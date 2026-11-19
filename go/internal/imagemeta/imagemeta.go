// Package imagemetaは、利用者がアップロードした画像から、撮影位置などのメタデータを取り除く。
//
// 画素のデータは復号も再エンコードもせず、JPEG・PNG・WebPの入れ物の構造だけを読んで、表示に要らない
// 部分を落として書き直す。画質を変えずに済み、cgoや外部のサービスにも依存しない。
// 表示の向きと色を保つため、向きの情報とカラープロファイルは残す。
package imagemeta

import (
	"bytes"
	"errors"
	"math"
)

var (
	// ErrUnsupportedFormatは、内容がJPEG・PNG・WebPのどれでもないことを表す
	ErrUnsupportedFormat = errors.New("メタデータを取り除けない形式の画像です")
	// ErrMalformedは、画像の構造を読めないことを表す。同じ内容で何度試しても結果は変わらない
	ErrMalformed = errors.New("画像の構造を読めません")
)

// Stripは、画像dataからメタデータを取り除いたバイト列を返す。形式は、申告された種類ではなく内容の先頭から
// 判別する。取り除くものが無ければdataをそのまま返し、結果をもう一度渡しても同じバイト列が返る。
// 形式を判別できなければErrUnsupportedFormatを、構造を読めなければErrMalformedを包んだエラーを返す
func Strip(data []byte) ([]byte, error) {
	var (
		out []byte
		err error
	)
	switch {
	case bytes.HasPrefix(data, jpegSOI):
		out, err = stripJPEG(data)
	case bytes.HasPrefix(data, pngSignature):
		out, err = stripPNG(data)
	case isWebP(data):
		out, err = stripWebP(data)
	default:
		return nil, ErrUnsupportedFormat
	}
	if err != nil {
		return nil, err
	}
	if bytes.Equal(out, data) {
		return data, nil
	}
	return out, nil
}

// chunkLengthは、チャンクやファイルの大きさを書く4バイトの欄に入れる、bの長さを返す。
// 書き出すのは読んだ4バイトの大きさの範囲に収まるデータだけのため、超えるのは実装の誤りである
func chunkLength(b []byte) uint32 {
	n := len(b)
	if n > math.MaxUint32 {
		panic("imagemeta: 4バイトの欄に収まらない長さです")
	}
	return uint32(n)
}
