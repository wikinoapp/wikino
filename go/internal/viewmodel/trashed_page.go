package viewmodel

import (
	"context"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// TrashedPageはゴミ箱の一覧の1行に出すページです
type TrashedPage struct {
	// IDは一括復元のフォームで選んだページを送る値です
	ID        string
	Number    PageNumber
	Topic     Topic
	TrashedAt time.Time
	title     string
}

// DisplayTitleは表示用のタイトルを返します。タイトルが無いページは「無題」と表示します。
func (p TrashedPage) DisplayTitle(ctx context.Context) string {
	if p.title != "" {
		return p.title
	}
	return i18n.T(ctx, "trash_show_untitled")
}

// NewTrashedPagesは、ゴミ箱に入れたページとそのトピックから一覧の行を生成します。
// topicMapに無いトピックのページは、閲覧者がゴミ箱を見られないページのため行にしません。
func NewTrashedPages(pages []*model.Page, topicMap map[model.TopicID]*model.Topic) []TrashedPage {
	rows := make([]TrashedPage, 0, len(pages))
	for _, pg := range pages {
		topic, ok := topicMap[pg.TopicID]
		if !ok || pg.TrashedAt == nil {
			continue
		}

		var title string
		if pg.Title != nil {
			title = *pg.Title
		}

		rows = append(rows, TrashedPage{
			ID:        string(pg.ID),
			Number:    PageNumber(pg.Number),
			Topic:     NewTopic(topic),
			TrashedAt: *pg.TrashedAt,
			title:     title,
		})
	}
	return rows
}
