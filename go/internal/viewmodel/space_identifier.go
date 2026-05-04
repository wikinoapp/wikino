package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// SpaceIdentifier はスペース識別子を表すPresentation層の型です
type SpaceIdentifier model.SpaceIdentifier

// NewSpaceIdentifier は model.SpaceIdentifier から viewmodel.SpaceIdentifier を生成します
func NewSpaceIdentifier(id model.SpaceIdentifier) SpaceIdentifier {
	return SpaceIdentifier(id)
}

// String は SpaceIdentifier を文字列に変換します
func (s SpaceIdentifier) String() string {
	return string(s)
}
