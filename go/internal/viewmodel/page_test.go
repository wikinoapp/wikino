package viewmodel_test

import (
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewPageForEdit(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name      string
		page      *model.Page
		draftPage *model.DraftPage
		wantTitle string
		wantBody  string
		wantNum   int32
	}{
		{
			name: "公開ページ（タイトルあり、下書きなし）",
			page: &model.Page{
				Number: 1,
				Title:  strPtr("公開タイトル"),
				Body:   "公開本文",
			},
			draftPage: nil,
			wantTitle: "公開タイトル",
			wantBody:  "公開本文",
			wantNum:   1,
		},
		{
			name: "公開ページ（タイトルなし、下書きなし）",
			page: &model.Page{
				Number: 2,
				Title:  nil,
				Body:   "タイトルなし本文",
			},
			draftPage: nil,
			wantTitle: "",
			wantBody:  "タイトルなし本文",
			wantNum:   2,
		},
		{
			name: "下書きあり（下書きのタイトル/本文が使われる）",
			page: &model.Page{
				Number: 3,
				Title:  strPtr("公開タイトル"),
				Body:   "公開本文",
			},
			draftPage: &model.DraftPage{
				Title: strPtr("下書きタイトル"),
				Body:  "下書き本文",
			},
			wantTitle: "下書きタイトル",
			wantBody:  "下書き本文",
			wantNum:   3,
		},
		{
			name: "下書きあり（下書きのタイトルがnil）",
			page: &model.Page{
				Number: 4,
				Title:  strPtr("公開タイトル"),
				Body:   "公開本文",
			},
			draftPage: &model.DraftPage{
				Title: nil,
				Body:  "下書き本文のみ",
			},
			wantTitle: "",
			wantBody:  "下書き本文のみ",
			wantNum:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewPageForEdit(tt.page, tt.draftPage)

			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}

			if got.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", got.Body, tt.wantBody)
			}

			if got.Number != tt.wantNum {
				t.Errorf("Number = %d, want %d", got.Number, tt.wantNum)
			}
		})
	}
}

func TestNewPageForShow(t *testing.T) {
	t.Parallel()

	title := "Page title"
	page := &model.Page{
		Title:    &title,
		BodyHTML: "<p>Page body</p>",
		Number:   42,
	}
	got := viewmodel.NewPageForShow(page, nil)
	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	if got.DisplayTitle(ctx) != title {
		t.Errorf("DisplayTitle() = %q, want %q", got.DisplayTitle(ctx), title)
	}
	if got.BodyHTML != page.BodyHTML {
		t.Errorf("BodyHTML = %q, want %q", got.BodyHTML, page.BodyHTML)
	}
	if got.Number != int32(page.Number) {
		t.Errorf("Number = %d, want %d", got.Number, page.Number)
	}
}

func TestPageForShow_DisplayTitle(t *testing.T) {
	t.Parallel()

	title := "Page title"
	emptyTitle := ""
	tests := []struct {
		name   string
		title  *string
		locale string
		want   string
	}{
		{name: "タイトルあり (日本語)", title: &title, locale: i18n.LangJa, want: title},
		{name: "タイトルあり (英語)", title: &title, locale: i18n.LangEn, want: title},
		{name: "タイトルが nil (日本語)", title: nil, locale: i18n.LangJa, want: "無題"},
		{name: "タイトルが nil (英語)", title: nil, locale: i18n.LangEn, want: "Untitled"},
		{name: "タイトルが空 (日本語)", title: &emptyTitle, locale: i18n.LangJa, want: "無題"},
		{name: "タイトルが空 (英語)", title: &emptyTitle, locale: i18n.LangEn, want: "Untitled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			page := viewmodel.NewPageForShow(&model.Page{Title: tt.title}, nil)
			ctx := i18n.SetLocale(t.Context(), tt.locale)

			if got := page.DisplayTitle(ctx); got != tt.want {
				t.Errorf("DisplayTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPageForShow_MetaDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bodyHTML string
		want     string
	}{
		{
			name:     "本文からプレーンテキストを取り出す",
			bodyHTML: "<p>ページの本文です。</p><p>2 つ目の段落。</p>",
			want:     "ページの本文です。 2 つ目の段落。",
		},
		{
			name:     "本文が空の場合は空文字列を返す",
			bodyHTML: "",
			want:     "",
		},
		{
			name:     "テキストを持たない本文は空文字列を返す",
			bodyHTML: `<p><img src="/attachments/1" alt="図"></p>`,
			want:     "",
		},
		{
			name:     "上限ちょうどの本文は切り詰めない",
			bodyHTML: "<p>" + strings.Repeat("あ", 120) + "</p>",
			want:     strings.Repeat("あ", 120),
		},
		{
			name:     "上限を超える本文は切り詰めて省略記号を付ける",
			bodyHTML: "<p>" + strings.Repeat("あ", 121) + "</p>",
			want:     strings.Repeat("あ", 119) + "…",
		},
		{
			// The 119th rune is the space markup.PlainText emits between the two blocks, so the cut
			// lands on it and no space may remain in front of the ellipsis.
			//
			// [Ja] 119 文字目は markup.PlainText が 2 つのブロックの間に出す半角スペースであり、
			// 切り取り位置がそこに重なる。省略記号の直前に空白が残ってはいけない。
			name:     "ブロック境界で切れる本文は省略記号の直前に空白を残さない",
			bodyHTML: "<p>" + strings.Repeat("あ", 118) + "</p><p>" + strings.Repeat("い", 5) + "</p>",
			want:     strings.Repeat("あ", 118) + "…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			page := viewmodel.NewPageForShow(&model.Page{BodyHTML: tt.bodyHTML}, nil)

			if got := page.MetaDescription(); got != tt.want {
				t.Errorf("MetaDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPageForShow_OGImageAttachmentID fixes which cover images the og:image tag may point at. The
// empty cases are the ones that must leave the caller on the site-wide default OGP image.
//
// [Ja] TestPageForShow_OGImageAttachmentID は og:image タグが指してよいアイキャッチ画像を固定する。
// 空文字列を返すケースは、呼び出し元がサイト共通の既定 OGP 画像を保たなければならないケースにあたる。
func TestPageForShow_OGImageAttachmentID(t *testing.T) {
	t.Parallel()

	attachment := func(filename string) *model.Attachment {
		return &model.Attachment{
			ID:       model.AttachmentID("550e8400-e29b-41d4-a716-446655440000"),
			Filename: filename,
		}
	}

	tests := []struct {
		name       string
		attachment *model.Attachment
		want       string
	}{
		{
			name:       "アイキャッチ画像を持つページは添付ファイルの ID を返す",
			attachment: attachment("cover.png"),
			want:       "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:       "アイキャッチ画像を持たないページは空文字列を返す",
			attachment: nil,
			want:       "",
		},
		{
			name:       "GIF のアイキャッチ画像は空文字列を返す",
			attachment: attachment("animation.gif"),
			want:       "",
		},
		{
			// Rails compares the downcased filename, so an uppercase extension is a GIF as well.
			//
			// [Ja] Rails 版は小文字化したファイル名で比較するため、大文字の拡張子も GIF として扱う。
			name:       "拡張子が大文字の GIF も空文字列を返す",
			attachment: attachment("ANIMATION.GIF"),
			want:       "",
		},
		{
			// Only the extension makes it a GIF: a name that merely contains "gif" does not.
			//
			// [Ja] GIF と判定するのは拡張子だけで、名前に "gif" を含むだけのファイルは判定しない。
			name:       "ファイル名に gif を含むだけの画像は ID を返す",
			attachment: attachment("gift.png"),
			want:       "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			// The filename is not populated on every fetch path, and an unknown format must not be
			// treated as a GIF.
			//
			// [Ja] ファイル名はすべての取得経路で populate されるわけではなく、形式が分からない場合を
			// GIF として扱ってはならない。
			name:       "ファイル名が空の添付ファイルは ID を返す",
			attachment: attachment(""),
			want:       "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			page := viewmodel.NewPageForShow(&model.Page{}, tt.attachment)

			if got := page.OGImageAttachmentID(); got != tt.want {
				t.Errorf("OGImageAttachmentID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewCardLinkPage(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }
	attachmentIDPtr := func(s string) *model.AttachmentID {
		id := model.AttachmentID(s)
		return &id
	}

	topicID := model.TopicID("topic-1")
	topicMap := map[model.TopicID]*model.Topic{
		topicID: {
			ID:         topicID,
			Name:       "テストトピック",
			Visibility: model.TopicVisibilityPublic,
		},
	}

	tests := []struct {
		name             string
		page             *model.Page
		topicMap         map[model.TopicID]*model.Topic
		wantTitle        string
		wantNumber       int32
		wantCardImageURL string
		wantPinned       bool
		wantTopicName    string
		wantTopicIcon    viewmodel.IconName
		wantTopicNil     bool
	}{
		{
			name: "アイキャッチ画像ありのページ",
			page: &model.Page{
				Number:                    1,
				Title:                     strPtr("画像付きページ"),
				TopicID:                   topicID,
				FeaturedImageAttachmentID: attachmentIDPtr("550e8400-e29b-41d4-a716-446655440000"),
			},
			topicMap:         topicMap,
			wantTitle:        "画像付きページ",
			wantNumber:       1,
			wantCardImageURL: "/attachments/550e8400-e29b-41d4-a716-446655440000",
			wantPinned:       false,
			wantTopicName:    "テストトピック",
			wantTopicIcon:    "globe-regular",
		},
		{
			name: "アイキャッチ画像なしのページ",
			page: &model.Page{
				Number:                    2,
				Title:                     strPtr("画像なしページ"),
				TopicID:                   topicID,
				FeaturedImageAttachmentID: nil,
			},
			topicMap:         topicMap,
			wantTitle:        "画像なしページ",
			wantNumber:       2,
			wantCardImageURL: "",
			wantPinned:       false,
			wantTopicName:    "テストトピック",
			wantTopicIcon:    "globe-regular",
		},
		{
			name: "タイトルがnilの場合は空文字になる",
			page: &model.Page{
				Number:  3,
				Title:   nil,
				TopicID: topicID,
			},
			topicMap:         topicMap,
			wantTitle:        "",
			wantNumber:       3,
			wantCardImageURL: "",
			wantPinned:       false,
			wantTopicName:    "テストトピック",
			wantTopicIcon:    "globe-regular",
		},
		{
			name: "topicMapがnilの場合はTopicがnilになる",
			page: &model.Page{
				Number:  4,
				Title:   strPtr("トピックなし"),
				TopicID: topicID,
			},
			topicMap:     nil,
			wantTitle:    "トピックなし",
			wantNumber:   4,
			wantTopicNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewCardLinkPage(tt.page, tt.topicMap)

			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if got.Number != tt.wantNumber {
				t.Errorf("Number = %d, want %d", got.Number, tt.wantNumber)
			}
			if got.CardImageURL != tt.wantCardImageURL {
				t.Errorf("CardImageURL = %q, want %q", got.CardImageURL, tt.wantCardImageURL)
			}
			if got.Pinned != tt.wantPinned {
				t.Errorf("Pinned = %v, want %v", got.Pinned, tt.wantPinned)
			}
			if tt.wantTopicNil {
				if got.Topic != nil {
					t.Errorf("Topic = %v, want nil", got.Topic)
				}
			} else {
				if got.Topic == nil {
					t.Fatal("Topic is nil, want non-nil")
				}
				if got.Topic.Name != tt.wantTopicName {
					t.Errorf("Topic.Name = %q, want %q", got.Topic.Name, tt.wantTopicName)
				}
				if got.Topic.IconName != tt.wantTopicIcon {
					t.Errorf("Topic.IconName = %q, want %q", got.Topic.IconName, tt.wantTopicIcon)
				}
			}
		})
	}
}

func TestPage_AutofocusTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		page viewmodel.Page
		want bool
	}{
		{
			name: "タイトルあり → false",
			page: viewmodel.Page{Title: "タイトル"},
			want: false,
		},
		{
			name: "タイトルなし → true",
			page: viewmodel.Page{Title: ""},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.page.AutofocusTitle(); got != tt.want {
				t.Errorf("AutofocusTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}
