package imagemeta

import "encoding/binary"

// orientationTagは、EXIFで画像の向きを表すタグ
const orientationTag = 0x0112

// tiffTypeShortは、TIFFの値の型のうち、2バイトの符号なし整数
const tiffTypeShort = 3

// parseOrientationは、EXIFのTIFFの構造tiffから、最初のIFDにある向きを返す。
// 向きが無いか、1から8の範囲に無いか、構造を読めないときは0を返す。ブラウザもこうしたEXIFの向きは
// 使わないため、読めない向きを捨てても表示は変わらない
func parseOrientation(tiff []byte) uint16 {
	if len(tiff) < 8 {
		return 0
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return 0
	}
	offset := uint64(order.Uint32(tiff[4:8]))
	if offset+2 > uint64(len(tiff)) {
		return 0
	}
	count := uint64(order.Uint16(tiff[offset:]))
	entries := offset + 2
	for i := range count {
		start := entries + i*12
		if start+12 > uint64(len(tiff)) {
			return 0
		}
		entry := tiff[start : start+12]
		if order.Uint16(entry[0:2]) != orientationTag {
			continue
		}
		if order.Uint16(entry[2:4]) != tiffTypeShort || order.Uint32(entry[4:8]) != 1 {
			return 0
		}
		if v := order.Uint16(entry[8:10]); v >= 1 && v <= 8 {
			return v
		}
		return 0
	}
	return 0
}

// orientationTIFFSizeは、orientationTIFFが返すバイト数 (ヘッダー・項目数・1つの項目・次のIFDの位置)
const orientationTIFFSize = 8 + 2 + 12 + 4

// orientationTIFFは、向きvだけを持つ最小のEXIFのTIFFの構造を返す。
// バイト順をビッグエンディアンに固定し、同じ向きからは常に同じバイト列を作る
func orientationTIFF(v uint16) []byte {
	b := make([]byte, 0, orientationTIFFSize)
	b = append(b, 'M', 'M', 0, 42)
	b = binary.BigEndian.AppendUint32(b, 8)
	b = binary.BigEndian.AppendUint16(b, 1)
	b = binary.BigEndian.AppendUint16(b, orientationTag)
	b = binary.BigEndian.AppendUint16(b, tiffTypeShort)
	b = binary.BigEndian.AppendUint32(b, 1)
	b = binary.BigEndian.AppendUint16(b, v)
	b = append(b, 0, 0)
	return binary.BigEndian.AppendUint32(b, 0)
}

// hasNonDefaultOrientationは、向きvが既定 (回転も反転も無し) 以外かどうかを返す
func hasNonDefaultOrientation(v uint16) bool {
	return v >= 2 && v <= 8
}
