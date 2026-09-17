package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// SpaceIdentifierはスペース識別子を表すPresentation層の型です
type SpaceIdentifier model.SpaceIdentifier

// NewSpaceIdentifierはmodel.SpaceIdentifierからviewmodel.SpaceIdentifierを生成します
func NewSpaceIdentifier(id model.SpaceIdentifier) SpaceIdentifier {
	return SpaceIdentifier(id)
}

// StringはSpaceIdentifierを文字列に変換します
func (s SpaceIdentifier) String() string {
	return string(s)
}
