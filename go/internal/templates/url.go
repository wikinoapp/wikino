package templates

import (
	"fmt"

	"github.com/a-h/templ"
)

// SpaceURL はスペースのURLを生成します
func SpaceURL(identifier string) templ.SafeURL {
	return templ.SafeURL("/s/" + identifier)
}

// TopicURL はトピックのURLを生成します
func TopicURL(spaceIdentifier string, topicNumber int32) string {
	return fmt.Sprintf("/s/%s/topics/%d", spaceIdentifier, topicNumber)
}
