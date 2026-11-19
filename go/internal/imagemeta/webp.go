package imagemeta

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
)

// VP8Xのチャンクの、メタデータとアニメーションの有無を表すフラグ (WebPのコンテナの仕様)
const (
	webpFlagEXIF      = 0x08
	webpFlagXMP       = 0x04
	webpFlagAnimation = 0x02
)

// webpKeptChunksは、そのまま残す画像・アルファ・カラープロファイルのチャンク。
// ANIMはヘッダーを検査し、ANMFはフレーム内の不要なチャンクを除いてから残す
var webpKeptChunks = []string{"ICCP", "ALPH", "VP8 ", "VP8L"}

// webpHeaderSizeは、RIFFのヘッダー ("RIFF"・大きさ・"WEBP") のバイト数
const webpHeaderSize = 12

// isWebPは、dataがWebPのRIFFのヘッダーで始まるかどうかを返す
func isWebP(data []byte) bool {
	return len(data) >= webpHeaderSize && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
}

// webpChunkはWebPのチャンク
type webpChunk struct {
	fourCC string
	body   []byte
	// rawは、ヘッダーと詰め物を含むチャンクのバイト列
	raw []byte
}

// stripWebPは、WebPのチャンクを読み、拡張形式ならEXIF・XMPと知らないチャンクを落として、VP8Xのフラグを直す。
// RIFFの大きさより後ろのデータも落とす。単純な形式は画像チャンクだけであることを確かめてから返す
func stripWebP(data []byte) ([]byte, error) {
	riffSize := uint64(binary.LittleEndian.Uint32(data[4:8]))
	if riffSize < 4 || riffSize+8 > uint64(len(data)) {
		return nil, fmt.Errorf("%w: WebPのRIFFの大きさ%dが範囲を超えています", ErrMalformed, riffSize)
	}
	riff := data[:riffSize+8]
	chunks, err := parseWebPChunks(riff[webpHeaderSize:])
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("%w: WebPにチャンクがありません", ErrMalformed)
	}
	if chunks[0].fourCC != "VP8X" {
		if len(chunks) != 1 || (chunks[0].fourCC != "VP8 " && chunks[0].fourCC != "VP8L") {
			return nil, fmt.Errorf("%w: 単純なWebPに画像以外のチャンクがあります", ErrMalformed)
		}
		if err := validateWebPImageChunks(chunks); err != nil {
			return nil, err
		}
		return riff, nil
	}
	if len(chunks[0].body) != 10 {
		return nil, fmt.Errorf("%w: WebPのVP8Xチャンクの長さが不正です", ErrMalformed)
	}
	if err := validateWebPLayout(chunks[1:], chunks[0].body[0]&webpFlagAnimation != 0); err != nil {
		return nil, err
	}

	var body []byte
	exifSeen := false
	exifWritten := false
	for _, c := range chunks[1:] {
		switch {
		case c.fourCC == "ANMF":
			frame, err := stripWebPFrame(c.body)
			if err != nil {
				return nil, err
			}
			body = appendWebPChunk(body, "ANMF", frame)
		case c.fourCC == "ANIM":
			body = append(body, c.raw...)
		case c.fourCC == "EXIF":
			if exifSeen {
				continue
			}
			exifSeen = true
			// 仕様ではTIFFの構造から始まるが、JPEGと同じ識別子を前に付けて書く実装もある
			tiff := bytes.TrimPrefix(c.body, exifIdentifier)
			if v := parseOrientation(tiff); hasNonDefaultOrientation(v) {
				body = appendWebPChunk(body, "EXIF", orientationTIFF(v))
				exifWritten = true
			}
		case slices.Contains(webpKeptChunks, c.fourCC):
			body = append(body, c.raw...)
		}
	}

	vp8x := bytes.Clone(chunks[0].body)
	vp8x[0] &^= webpFlagEXIF | webpFlagXMP
	if exifWritten {
		vp8x[0] |= webpFlagEXIF
	}

	out := make([]byte, 0, len(riff))
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, 0)
	out = append(out, "WEBP"...)
	out = appendWebPChunk(out, "VP8X", vp8x)
	out = append(out, body...)
	binary.LittleEndian.PutUint32(out[4:8], chunkLength(out[8:]))
	return out, nil
}

// validateWebPLayoutは、静止画とアニメーションそれぞれの必須の画像構造を検査する。
// アニメーションの画像データは、ANIMの後に続くANMFの中に置く
func validateWebPLayout(chunks []webpChunk, animated bool) error {
	animationSeen := false
	frames := 0
	for _, c := range chunks {
		switch c.fourCC {
		case "ANIM":
			if !animated || animationSeen || len(c.body) != 6 {
				return fmt.Errorf("%w: WebPのANIMチャンクが不正です", ErrMalformed)
			}
			animationSeen = true
		case "ANMF":
			if !animated || !animationSeen {
				return fmt.Errorf("%w: WebPのアニメーションにANIMがありません", ErrMalformed)
			}
			frames++
		case "ALPH", "VP8 ", "VP8L":
			if animated {
				return fmt.Errorf("%w: WebPのアニメーションの画像データがANMFの外にあります", ErrMalformed)
			}
		}
	}
	if animated {
		if !animationSeen || frames == 0 {
			return fmt.Errorf("%w: WebPのアニメーションにフレームがありません", ErrMalformed)
		}
		return nil
	}
	return validateWebPImageChunks(chunks)
}

// stripWebPFrameは、ANMFのフレームヘッダーと画像のサブチャンクを保ち、アプリ固有データを除く。
// フレームの外側と同じ長さの検査をサブチャンクにも適用し、画像のデータがあることを確かめる
func stripWebPFrame(frame []byte) ([]byte, error) {
	const headerSize = 16
	if len(frame) < headerSize {
		return nil, fmt.Errorf("%w: WebPのフレームヘッダーが短すぎます", ErrMalformed)
	}
	chunks, err := parseWebPChunks(frame[headerSize:])
	if err != nil {
		return nil, err
	}
	if err := validateWebPImageChunks(chunks); err != nil {
		return nil, err
	}
	out := bytes.Clone(frame[:headerSize])
	for _, c := range chunks {
		if c.fourCC == "ALPH" || c.fourCC == "VP8 " || c.fourCC == "VP8L" {
			out = append(out, c.raw...)
		}
	}
	return out, nil
}

// validateWebPImageChunksは、1つの画像のビットストリームと、その前に置けるアルファの構造を検査する。
// 画素は復号せず、画像のヘッダーと、符号化されたデータがあることを確かめる
func validateWebPImageChunks(chunks []webpChunk) error {
	imageSeen := false
	alphaSeen := false
	for _, c := range chunks {
		switch c.fourCC {
		case "ALPH":
			if imageSeen || alphaSeen || len(c.body) == 0 {
				return fmt.Errorf("%w: WebPのアルファのチャンクが不正です", ErrMalformed)
			}
			alphaSeen = true
		case "VP8 ", "VP8L":
			if imageSeen || (alphaSeen && c.fourCC == "VP8L") {
				return fmt.Errorf("%w: WebPの画像のチャンクが重複しているか、アルファとの組が不正です", ErrMalformed)
			}
			if err := validateWebPImageHeader(c); err != nil {
				return err
			}
			imageSeen = true
		}
	}
	if !imageSeen {
		return fmt.Errorf("%w: WebPに画像のチャンクがありません", ErrMalformed)
	}
	return nil
}

// validateWebPImageHeaderは、VP8またはVP8Lの画像ヘッダーを検査する。
// ヘッダーしかない入力も、符号化された画素を持たないため拒否する
func validateWebPImageHeader(c webpChunk) error {
	if c.fourCC == "VP8 " {
		if len(c.body) <= 10 || c.body[0]&1 != 0 || !bytes.Equal(c.body[3:6], []byte{0x9D, 0x01, 0x2A}) {
			return fmt.Errorf("%w: WebPのVP8の画像ヘッダーかデータが不正です", ErrMalformed)
		}
		return nil
	}
	if len(c.body) <= 5 || c.body[0] != 0x2F || c.body[4]&0xE0 != 0 {
		return fmt.Errorf("%w: WebPのVP8Lの画像ヘッダーかデータが不正です", ErrMalformed)
	}
	return nil
}

// parseWebPChunksは、RIFFの中身bをチャンクに分ける
func parseWebPChunks(b []byte) ([]webpChunk, error) {
	var chunks []webpChunk
	pos := 0
	for pos < len(b) {
		if pos+8 > len(b) {
			return nil, fmt.Errorf("%w: WebPのチャンクのヘッダーが途中で切れています", ErrMalformed)
		}
		size := int(binary.LittleEndian.Uint32(b[pos+4:]))
		padded := size + size%2
		if padded > len(b)-pos-8 {
			return nil, fmt.Errorf("%w: WebPのチャンクの大きさ%dが範囲を超えています", ErrMalformed, size)
		}
		end := pos + 8 + padded
		chunks = append(chunks, webpChunk{
			fourCC: string(b[pos : pos+4]),
			body:   b[pos+8 : pos+8+size],
			raw:    b[pos:end],
		})
		pos = end
	}
	return chunks, nil
}

// appendWebPChunkは、種類fourCCと中身bodyのチャンクを、奇数の大きさなら詰め物を付けてoutに足す
func appendWebPChunk(out []byte, fourCC string, body []byte) []byte {
	out = append(out, fourCC...)
	out = binary.LittleEndian.AppendUint32(out, chunkLength(body))
	out = append(out, body...)
	if len(body)%2 == 1 {
		out = append(out, 0)
	}
	return out
}
