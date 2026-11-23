package imagemeta

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"slices"
)

// pngSignatureは、PNGの先頭の8バイト
var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// pngKeptAncillaryChunksは、補助チャンクのうち、画像の表示 (色・透過・画素の縦横比・アニメーション) に使うため残すもの。
// それ以外の補助チャンク (テキスト・EXIF・更新日時・そのほかのアプリケーションのデータ) は落とす
var pngKeptAncillaryChunks = []string{
	"tRNS", "cHRM", "gAMA", "iCCP", "sBIT", "sRGB", "cICP", "mDCV", "cLLI", "bKGD", "pHYs",
	"acTL", "fcTL", "fdAT",
}

// pngChunkOverheadは、チャンクの中身の前後に付く長さ・種類・CRCのバイト数
const pngChunkOverhead = 12

// stripPNGは、PNGのチャンクを順に読み、メタデータのチャンクとIENDより後ろのデータを落とす。
// 最初のeXIfが既定でない向きを持つときは、その位置に向きだけを持つeXIfを書く
func stripPNG(data []byte) ([]byte, error) {
	out := make([]byte, 0, len(data))
	out = append(out, pngSignature...)
	exifSeen := false
	imageDataSeen := false
	pos := len(pngSignature)

	for first := true; ; first = false {
		if pos+pngChunkOverhead > len(data) {
			return nil, fmt.Errorf("%w: PNGのIENDがありません", ErrMalformed)
		}
		length := binary.BigEndian.Uint32(data[pos:])
		if int(length) > len(data)-pos-pngChunkOverhead {
			return nil, fmt.Errorf("%w: PNGのチャンクの長さ%dが範囲を超えています", ErrMalformed, length)
		}
		end := pos + pngChunkOverhead + int(length)
		typ := string(data[pos+4 : pos+8])
		body := data[pos+8 : end-4]
		if !isPNGChunkType(typ) {
			return nil, fmt.Errorf("%w: PNGのチャンクの種類が不正です", ErrMalformed)
		}
		if crc32.ChecksumIEEE(data[pos+4:end-4]) != binary.BigEndian.Uint32(data[end-4:]) {
			return nil, fmt.Errorf("%w: PNGの%sチャンクのCRCが一致しません", ErrMalformed, typ)
		}
		if first && typ != "IHDR" {
			return nil, fmt.Errorf("%w: PNGの最初のチャンクがIHDRではありません", ErrMalformed)
		}
		switch typ {
		case "IHDR":
			if !first || len(body) != 13 {
				return nil, fmt.Errorf("%w: PNGのIHDRが重複しているか、長さが不正です", ErrMalformed)
			}
		case "IDAT":
			imageDataSeen = imageDataSeen || len(body) > 0
		case "IEND":
			if len(body) != 0 || !imageDataSeen {
				return nil, fmt.Errorf("%w: PNGに画像のデータが無いか、IENDの長さが不正です", ErrMalformed)
			}
		}

		switch {
		case typ == "eXIf":
			if !exifSeen {
				exifSeen = true
				if v := parseOrientation(body); hasNonDefaultOrientation(v) {
					out = appendPNGChunk(out, "eXIf", orientationTIFF(v))
				}
			}
		case isPNGCriticalChunk(typ) || slices.Contains(pngKeptAncillaryChunks, typ):
			out = append(out, data[pos:end]...)
		}
		pos = end

		if typ == "IEND" {
			return out, nil
		}
	}
}

// isPNGChunkTypeは、typがチャンクの種類として使えるASCIIの英字4文字かどうかを返す
func isPNGChunkType(typ string) bool {
	for i := range len(typ) {
		c := typ[i]
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
}

// isPNGCriticalChunkは、種類typが復号に欠かせない必須チャンクかどうかを返す (1文字目が大文字)。
// 必須チャンクは、知らない種類でも落とすと画像が壊れるため残す
func isPNGCriticalChunk(typ string) bool {
	return typ[0] >= 'A' && typ[0] <= 'Z'
}

// appendPNGChunkは、種類typと中身bodyのチャンクをoutに足す
func appendPNGChunk(out []byte, typ string, body []byte) []byte {
	out = binary.BigEndian.AppendUint32(out, chunkLength(body))
	start := len(out)
	out = append(out, typ...)
	out = append(out, body...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(out[start:]))
}
