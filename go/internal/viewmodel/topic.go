package viewmodel

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TopicVisibilityIconName はトピックの公開範囲に対応するアイコン名を返します
func TopicVisibilityIconName(v model.TopicVisibility) string {
	if v == model.TopicVisibilityPublic {
		return "globe-regular"
	}
	return "lock-regular"
}
