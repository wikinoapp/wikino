package viewmodel

import "github.com/wikinoapp/wikino/go/internal/model"

// SearchResultPageは検索の結果の1件です。結果はスペースをまたぐため、カードのリンク先を組み立てる
// スペースの識別子をページごとに持ちます。
type SearchResultPage struct {
	Card            CardLinkPage
	SpaceIdentifier SpaceIdentifier
}

// NewSearchResultPagesは、検索で見つけたページとそのトピック (スペースを添えたもの) から結果を生成します。
// topicMapに無いトピックのページは、リンク先を組み立てられないため結果にしません。
func NewSearchResultPages(pages []*model.Page, topicMap map[model.TopicID]*model.Topic) []SearchResultPage {
	results := make([]SearchResultPage, 0, len(pages))
	for _, pg := range pages {
		topic, ok := topicMap[pg.TopicID]
		if !ok || topic.Space == nil {
			continue
		}

		card := NewCardLinkPage(pg, topicMap)
		card.SpaceName = topic.Space.Name
		results = append(results, SearchResultPage{
			Card:            card,
			SpaceIdentifier: NewSpaceIdentifier(topic.Space.Identifier),
		})
	}
	return results
}
