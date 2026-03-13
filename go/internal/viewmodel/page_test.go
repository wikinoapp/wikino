package viewmodel_test

import (
	"testing"

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
