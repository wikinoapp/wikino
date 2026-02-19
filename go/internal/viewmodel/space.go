package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// Space はテンプレートで表示するスペース情報です
type Space struct {
	Name       string
	Identifier string
}

// NewSpace はモデルからSpaceを生成します
func NewSpace(space *model.Space) Space {
	return Space{
		Name:       space.Name,
		Identifier: space.Identifier,
	}
}
