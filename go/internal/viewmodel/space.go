package viewmodel

import (
	"hash/fnv"
	"unicode"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// Space はテンプレートで表示するスペース情報です
type Space struct {
	Name       string
	Identifier SpaceIdentifier
}

// spaceIconPalette is the palette used for SpaceIcon background colors.
// Darker, muted tones are chosen so white text remains legible.
// Selected deterministically: the same identifier always maps to the same color.
//
// [Ja] spaceIconPalette は SpaceIcon の背景色として使用するパレットです。
// コントラスト確保のため白文字を載せやすい落ち着いた濃いめのトーンを選んでいます。
// パレットは決定論的に選択されるため、同じ identifier には常に同じ色が割り当てられます。
var spaceIconPalette = [...]string{
	"#B26836", // muted orange. [Ja] くすんだオレンジ
	"#2D5F4C", // deep green. [Ja] 深緑
	"#8B3A3A", // red brown. [Ja] 赤茶
	"#4A5D7E", // slate blue. [Ja] スレートブルー
	"#5D4A7E", // dusty purple. [Ja] くすんだ紫
	"#2C7373", // teal. [Ja] ティール
	"#2C3E5D", // navy. [Ja] 紺
	"#6B4F3A", // brown. [Ja] 茶色
	"#8B7A2C", // mustard. [Ja] マスタード
	"#6B2C3E", // bordeaux. [Ja] ボルドー
	"#3F4F4F", // dark slate gray. [Ja] ダークスレートグレー
	"#2C2C2C", // near black. [Ja] ほぼ黒
}

// spaceIconPaletteLen holds the element count of spaceIconPalette as uint32.
// Pre-computed for taking modulo against an FNV-1a hash value (uint32). Casting len() to uint32
// trips gosec's integer overflow detector, so we derive it from the array literal as a fixed constant.
//
// [Ja] spaceIconPaletteLen は spaceIconPalette の要素数を uint32 で保持します。
// FNV-1a のハッシュ値 (uint32) と剰余演算するための前計算値。len() を uint32 にキャストすると
// gosec の integer overflow 検知に引っかかるため、配列リテラルから固定値として導出する。
const spaceIconPaletteLen = uint32(len(spaceIconPalette))

// NewSpace はモデルからSpaceを生成します
func NewSpace(space *model.Space) Space {
	return Space{
		Name:       space.Name,
		Identifier: NewSpaceIdentifier(space.Identifier),
	}
}

// NewSpaces はモデルのスライスからSpaceのスライスを生成します
func NewSpaces(spaces []*model.Space) []Space {
	result := make([]Space, len(spaces))
	for i, space := range spaces {
		result[i] = NewSpace(space)
	}
	return result
}

// IconBackgroundColor returns the icon background color (hex) deterministically chosen from the identifier.
// It hashes the identifier with FNV-1a and picks a color by taking the modulo against the palette length.
//
// [Ja] IconBackgroundColor は identifier から決定的に選ばれたアイコン背景色 (hex) を返します。
// identifier を FNV-1a でハッシュし、パレット長で剰余を取って色を選択します。
func (s Space) IconBackgroundColor() string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s.Identifier))
	return spaceIconPalette[h.Sum32()%spaceIconPaletteLen]
}

// IconLabel returns a one-character label, the first character of the identifier uppercased.
// Returns an empty string when the identifier is empty. Characters outside the ASCII range are not
// affected by ToUpper, so they are returned as-is.
//
// [Ja] IconLabel は identifier の先頭 1 文字を大文字化した 1 文字のラベルを返します。
// identifier が空のときは空文字列を返します。ASCII 範囲外の文字は ToUpper の対象外なのでそのまま返されます。
func (s Space) IconLabel() string {
	for _, r := range string(s.Identifier) {
		return string(unicode.ToUpper(r))
	}
	return ""
}
