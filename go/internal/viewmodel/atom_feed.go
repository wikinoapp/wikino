package viewmodel

import (
	"encoding/xml"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// atomTagDateはフィードとエントリーのid (tag URI。RFC 4151) に入れる日付。
// idはフィードリーダーがエントリーを見分けるのに使うため、Rails版のフィードと同じ値から変えない
const atomTagDate = "2025"

// AtomFeedはAtom (RFC 4287) のフィードの文書。ハンドラーが `encoding/xml` で書き出す
type AtomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	ID      string      `xml:"id"`
	Title   string      `xml:"title"`
	Links   []AtomLink  `xml:"link"`
	Updated string      `xml:"updated"`
	Author  AtomPerson  `xml:"author"`
	Entries []AtomEntry `xml:"entry"`
}

// AtomEntryはフィードのエントリー (1つのページ)
type AtomEntry struct {
	ID        string   `xml:"id"`
	Title     string   `xml:"title"`
	Link      AtomLink `xml:"link"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
	Content   AtomText `xml:"content"`
}

// AtomLinkはAtomのlink要素
type AtomLink struct {
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr,omitempty"`
	Href string `xml:"href,attr"`
}

// AtomPersonはAtomのauthor要素
type AtomPerson struct {
	Name string `xml:"name"`
}

// AtomTextはtype属性を持つAtomのテキスト。Bodyは書き出すときにエスケープされる
type AtomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

// AtomFeedSourceはフィードを組み立てる材料
type AtomFeedSource struct {
	Space *model.Space
	// Pagesはエントリーにするページ。並び順のままエントリーにする
	Pages []*model.Page
	// ContentHTMLByPageIDは各ページの本文を絶対URLで描画したHTML
	ContentHTMLByPageID map[model.PageID]string
	// AlternatePathはフィードの内容を表示する画面のパス、SelfPathはフィード自身のパス
	AlternatePath string
	SelfPath      string
}

// NewSpaceAtomFeedはスペースのフィードを組み立てる
func NewSpaceAtomFeed(cfg *config.Config, src AtomFeedSource) AtomFeed {
	return newAtomFeed(cfg, src.Space.Name+" | Wikino", src)
}

// NewTopicAtomFeedはトピックのフィードを組み立てる。src.Spaceはトピックの属するスペースを渡す
func NewTopicAtomFeed(cfg *config.Config, topic *model.Topic, src AtomFeedSource) AtomFeed {
	return newAtomFeed(cfg, topic.Name+" | "+src.Space.Name+" | Wikino", src)
}

func newAtomFeed(cfg *config.Config, title string, src AtomFeedSource) AtomFeed {
	appURL := cfg.AppURL()

	entries := make([]AtomEntry, len(src.Pages))
	// RFC 4287はフィードのupdatedを必須にしている。エントリーが無いときもフィードの内容が
	// 変わらない限り同じ値になるよう、現在時刻ではなくスペースの作成日時を使う (ETagを保つため)
	updated := src.Space.JoinedAt
	for i, pg := range src.Pages {
		entries[i] = newAtomEntry(appURL, src.Space, pg, src.ContentHTMLByPageID[pg.ID])
		if i == 0 || pg.ModifiedAt.After(updated) {
			updated = pg.ModifiedAt
		}
	}

	return AtomFeed{
		ID:    "tag:" + cfg.Domain + "," + atomTagDate + ":" + src.SelfPath,
		Title: title,
		Links: []AtomLink{
			{Rel: "alternate", Type: "text/html", Href: appURL + src.AlternatePath},
			{Rel: "self", Type: "application/atom+xml", Href: appURL + src.SelfPath},
		},
		Updated: atomTime(updated),
		Author:  AtomPerson{Name: src.Space.Name},
		Entries: entries,
	}
}

func newAtomEntry(appURL string, space *model.Space, pg *model.Page, contentHTML string) AtomEntry {
	var title string
	if pg.Title != nil {
		title = *pg.Title
	}

	// フィードに載せるのは公開済みのページだけだが、公開日時が無いときは更新日時で代える
	published := pg.ModifiedAt
	if pg.PublishedAt != nil {
		published = *pg.PublishedAt
	}

	return AtomEntry{
		ID:        "tag:Wikino," + atomTagDate + ":Page/" + pg.ID.String(),
		Title:     title,
		Link:      AtomLink{Rel: "alternate", Type: "text/html", Href: model.PageWebURL(appURL, space, pg.Number)},
		Published: atomTime(published),
		Updated:   atomTime(pg.ModifiedAt),
		Content:   AtomText{Type: "html", Body: contentHTML},
	}
}

// atomTimeは日時をUTCのRFC 3339の形式にする
func atomTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
