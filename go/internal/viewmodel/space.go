package viewmodel

import (
	"hash/fnv"
	"unicode"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// Spaceはテンプレートで表示するスペース情報です
type Space struct {
	Name       string
	Identifier SpaceIdentifier
}

// spaceIconPaletteはSpaceIconの背景色として使用するパレットです。
// コントラスト確保のため白文字を載せやすい落ち着いた濃いめのトーンを選んでいます。
// パレットは決定論的に選択されるため、同じidentifierには常に同じ色が割り当てられます。
var spaceIconPalette = [...]string{
	"#B26836", // くすんだオレンジ
	"#2D5F4C", // 深緑
	"#8B3A3A", // 赤茶
	"#4A5D7E", // スレートブルー
	"#5D4A7E", // くすんだ紫
	"#2C7373", // ティール
	"#2C3E5D", // 紺
	"#6B4F3A", // 茶色
	"#8B7A2C", // マスタード
	"#6B2C3E", // ボルドー
	"#3F4F4F", // ダークスレートグレー
	"#2C2C2C", // ほぼ黒
}

// spaceIconPaletteLenはspaceIconPaletteの要素数をuint32で保持します。
// FNV-1aのハッシュ値 (uint32) と剰余演算するための前計算値。len() をuint32にキャストすると
// gosecのinteger overflow検知に引っかかるため、配列リテラルから固定値として導出する。
const spaceIconPaletteLen = uint32(len(spaceIconPalette))

// NewSpaceはモデルからSpaceを生成します
func NewSpace(space *model.Space) Space {
	return Space{
		Name:       space.Name,
		Identifier: NewSpaceIdentifier(space.Identifier),
	}
}

// NewSpacesはモデルのスライスからSpaceのスライスを生成します
func NewSpaces(spaces []*model.Space) []Space {
	result := make([]Space, len(spaces))
	for i, space := range spaces {
		result[i] = NewSpace(space)
	}
	return result
}

// IconBackgroundColorはidentifierから決定的に選ばれたアイコン背景色 (hex) を返します。
// identifierをFNV-1aでハッシュし、パレット長で剰余を取って色を選択します。
func (s Space) IconBackgroundColor() string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s.Identifier))
	return spaceIconPalette[h.Sum32()%spaceIconPaletteLen]
}

// IconLabelはidentifierの先頭1文字を大文字化した1文字のラベルを返します。
// identifierが空のときは空文字列を返します。ASCII範囲外の文字はToUpperの対象外なのでそのまま返されます。
func (s Space) IconLabel() string {
	for _, r := range string(s.Identifier) {
		return string(unicode.ToUpper(r))
	}
	return ""
}
