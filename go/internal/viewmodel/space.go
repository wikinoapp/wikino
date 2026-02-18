package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// SpaceHeader はレイアウトのヘッダーに表示するスペース情報です
type SpaceHeader struct {
	Name       string
	Identifier string
}

// NewSpaceHeader はモデルからSpaceHeaderを生成します
func NewSpaceHeader(space *model.Space) SpaceHeader {
	return SpaceHeader{
		Name:       space.Name,
		Identifier: space.Identifier,
	}
}
