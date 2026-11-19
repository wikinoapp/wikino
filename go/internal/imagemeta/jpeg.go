package imagemeta

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// JPEGのマーカー (ITU-T T.81の表B.1)
const (
	jpegMarkerTEM   = 0x01
	jpegMarkerSOF0  = 0xC0
	jpegMarkerDHT   = 0xC4
	jpegMarkerJPG   = 0xC8
	jpegMarkerDAC   = 0xCC
	jpegMarkerSOF15 = 0xCF
	jpegMarkerRST0  = 0xD0
	jpegMarkerRST7  = 0xD7
	jpegMarkerSOI   = 0xD8
	jpegMarkerEOI   = 0xD9
	jpegMarkerSOS   = 0xDA
	jpegMarkerAPP0  = 0xE0
	jpegMarkerAPP1  = 0xE1
	jpegMarkerAPP2  = 0xE2
	jpegMarkerAPP14 = 0xEE
	jpegMarkerAPP15 = 0xEF
	jpegMarkerCOM   = 0xFE
)

// jpegSOIは、JPEGの画像の始まり
var jpegSOI = []byte{0xFF, jpegMarkerSOI}

// アプリケーションのセグメントのうち、残すものとEXIFを見分ける識別子
var (
	jfifIdentifier  = []byte("JFIF\x00")
	iccIdentifier   = []byte("ICC_PROFILE\x00")
	adobeIdentifier = []byte("Adobe")
	exifIdentifier  = []byte("Exif\x00\x00")
)

// stripJPEGは、JPEGのセグメントを順に読み、メタデータのセグメントと画像の終わりより後ろのデータを落とす。
// 最初のEXIFが既定でない向きを持つときは、その位置に向きだけを持つEXIFを書く
func stripJPEG(data []byte) ([]byte, error) {
	out := make([]byte, 0, len(data))
	out = append(out, jpegSOI...)
	exifSeen := false
	frameSeen := false
	scanSeen := false
	pos := len(jpegSOI)

	for {
		if pos >= len(data) || data[pos] != 0xFF {
			return nil, fmt.Errorf("%w: JPEGのマーカーがありません (位置%d)", ErrMalformed, pos)
		}
		markerStart := pos
		// マーカーの前には、詰め物の0xFFが続くことがある
		for pos < len(data) && data[pos] == 0xFF {
			pos++
		}
		if pos >= len(data) {
			return nil, fmt.Errorf("%w: JPEGの画像の終わりがありません", ErrMalformed)
		}
		marker := data[pos]
		pos++

		switch {
		case marker == jpegMarkerEOI:
			if !frameSeen || !scanSeen {
				return nil, fmt.Errorf("%w: JPEGにフレームと画像のスキャンがありません", ErrMalformed)
			}
			return append(out, data[markerStart:pos]...), nil
		case marker == jpegMarkerTEM || (marker >= jpegMarkerRST0 && marker <= jpegMarkerRST7):
			out = append(out, data[markerStart:pos]...)
			continue
		case marker == 0x00 || marker == jpegMarkerSOI:
			return nil, fmt.Errorf("%w: JPEGに置けないマーカー0x%02Xがあります", ErrMalformed, marker)
		}

		if pos+2 > len(data) {
			return nil, fmt.Errorf("%w: JPEGのセグメントの長さがありません", ErrMalformed)
		}
		length := int(binary.BigEndian.Uint16(data[pos:]))
		if length < 2 || pos+length > len(data) {
			return nil, fmt.Errorf("%w: JPEGのセグメントの長さ%dが範囲を超えています", ErrMalformed, length)
		}
		payload := data[pos+2 : pos+length]

		if isJPEGFrameMarker(marker) {
			if len(payload) < 6 || payload[5] == 0 || len(payload) != 6+3*int(payload[5]) {
				return nil, fmt.Errorf("%w: JPEGのフレームヘッダーが不正です", ErrMalformed)
			}
			frameSeen = true
		}
		if marker == jpegMarkerSOS {
			if !frameSeen || len(payload) < 4 || payload[0] == 0 || len(payload) != 4+2*int(payload[0]) {
				return nil, fmt.Errorf("%w: JPEGのスキャンヘッダーが不正です", ErrMalformed)
			}
		}

		switch {
		case keepJPEGSegment(marker, payload):
			out = append(out, data[markerStart:pos+length]...)
		case marker == jpegMarkerAPP1 && bytes.HasPrefix(payload, exifIdentifier) && !exifSeen:
			// ブラウザが向きに使うのは最初のEXIFのため、2つ目以降の向きは見ない
			exifSeen = true
			if v := parseOrientation(payload[len(exifIdentifier):]); hasNonDefaultOrientation(v) {
				out = appendJPEGOrientation(out, v)
			}
		}
		pos += length

		if marker == jpegMarkerSOS {
			end, err := jpegScanEnd(data, pos)
			if err != nil {
				return nil, err
			}
			if end == pos {
				return nil, fmt.Errorf("%w: JPEGのスキャンに画像のデータがありません", ErrMalformed)
			}
			scanSeen = true
			out = append(out, data[pos:end]...)
			pos = end
		}
	}
}

// isJPEGFrameMarkerは、markerがフレームのヘッダー (SOFn) かどうかを返す。
// 同じ範囲に割り当てられたテーブルと拡張用のマーカーを除く
func isJPEGFrameMarker(marker byte) bool {
	return marker >= jpegMarkerSOF0 && marker <= jpegMarkerSOF15 &&
		marker != jpegMarkerDHT && marker != jpegMarkerJPG && marker != jpegMarkerDAC
}

// keepJPEGSegmentは、マーカーmarkerと中身payloadのセグメントを残すかどうかを返す。
// アプリケーションのセグメントは、表示に使うJFIF・カラープロファイル・Adobe (色の変換の指定) だけを残し、
// コメントは落とす。それ以外 (量子化表・ハフマン表・フレームとスキャンのヘッダーなど) は復号に要るため残す
func keepJPEGSegment(marker byte, payload []byte) bool {
	switch {
	case marker == jpegMarkerAPP0:
		return bytes.HasPrefix(payload, jfifIdentifier)
	case marker == jpegMarkerAPP2:
		return bytes.HasPrefix(payload, iccIdentifier)
	case marker == jpegMarkerAPP14:
		return bytes.HasPrefix(payload, adobeIdentifier)
	case marker >= jpegMarkerAPP0 && marker <= jpegMarkerAPP15:
		return false
	case marker == jpegMarkerCOM:
		return false
	default:
		return true
	}
}

// jpegScanEndは、位置startから始まる符号化されたデータの終わり (次のマーカーの位置) を返す。
// データの中の0xFFは、0x00 (バイトの詰め物) かリスタートのマーカーを伴う。0xFFが続くときはマーカーの前の
// 詰め物のため読み飛ばし、0x00・リスタートのマーカー・0xFF以外が続く0xFFの位置で終わる
func jpegScanEnd(data []byte, start int) (int, error) {
	p := start
	for {
		i := bytes.IndexByte(data[p:], 0xFF)
		if i < 0 || p+i+1 >= len(data) {
			return 0, fmt.Errorf("%w: JPEGの画像の終わりがありません", ErrMalformed)
		}
		p += i
		next := data[p+1]
		switch {
		case next == 0x00 || (next >= jpegMarkerRST0 && next <= jpegMarkerRST7):
			p += 2
		case next == 0xFF:
			p++
		default:
			return p, nil
		}
	}
}

// jpegOrientationSegmentLengthは、向きだけを持つEXIFのセグメントの長さ (長さの2バイト・識別子・TIFFの構造)
const jpegOrientationSegmentLength = 2 + 6 + orientationTIFFSize

// appendJPEGOrientationは、向きvだけを持つEXIFのセグメントをoutに足す
func appendJPEGOrientation(out []byte, v uint16) []byte {
	tiff := orientationTIFF(v)
	out = append(out, 0xFF, jpegMarkerAPP1)
	out = binary.BigEndian.AppendUint16(out, jpegOrientationSegmentLength)
	out = append(out, exifIdentifier...)
	return append(out, tiff...)
}
