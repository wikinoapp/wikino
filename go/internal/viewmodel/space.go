package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// Space はテンプレートで表示するスペース情報です
type Space struct {
	Name       string
	Identifier SpaceIdentifier
}

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
