package viewmodel_test

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

func TestNewDraftPageGroupsForIndex_Empty(t *testing.T) {
	t.Parallel()

	groups := viewmodel.NewDraftPageGroupsForIndex(nil)
	if groups != nil {
		t.Errorf("実測値 = %v、期待値 = nil", groups)
	}
}

func TestNewDraftPageGroupsForIndex_Grouping(t *testing.T) {
	t.Parallel()

	now := time.Now()
	drafts := []*model.DraftPage{
		{
			ID:         "dp1",
			Title:      strPtr("ページ1"),
			ModifiedAt: now,
			Page:       &model.Page{ID: "p1", Number: 1},
			Topic: &model.Topic{
				ID:         "t1",
				Name:       "トピックA",
				Visibility: model.TopicVisibilityPublic,
				Space:      &model.Space{ID: "s1", Identifier: "space-a", Name: "スペースA"},
			},
		},
		{
			ID:         "dp2",
			Title:      strPtr("ページ2"),
			ModifiedAt: now,
			Page:       &model.Page{ID: "p2", Number: 2},
			Topic: &model.Topic{
				ID:         "t1",
				Name:       "トピックA",
				Visibility: model.TopicVisibilityPublic,
				Space:      &model.Space{ID: "s1", Identifier: "space-a", Name: "スペースA"},
			},
		},
		{
			ID:         "dp3",
			Title:      strPtr("ページ3"),
			ModifiedAt: now,
			Page:       &model.Page{ID: "p3", Number: 3},
			Topic: &model.Topic{
				ID:         "t2",
				Name:       "トピックB",
				Visibility: model.TopicVisibilityPrivate,
				Space:      &model.Space{ID: "s1", Identifier: "space-a", Name: "スペースA"},
			},
		},
	}

	groups := viewmodel.NewDraftPageGroupsForIndex(drafts)

	if len(groups) != 2 {
		t.Fatalf("グループの件数 = %d、期待値 = 2", len(groups))
	}

	if groups[0].SpaceName != "スペースA" {
		t.Errorf("スペース名 = %s、期待値 = スペースA", groups[0].SpaceName)
	}
	if groups[0].TopicName != "トピックA" {
		t.Errorf("トピック名 = %s、期待値 = トピックA", groups[0].TopicName)
	}
	if len(groups[0].DraftPages) != 2 {
		t.Errorf("グループ1の下書きの件数 = %d、期待値 = 2", len(groups[0].DraftPages))
	}

	if groups[1].TopicName != "トピックB" {
		t.Errorf("トピック名 = %s、期待値 = トピックB", groups[1].TopicName)
	}
	if len(groups[1].DraftPages) != 1 {
		t.Errorf("グループ2の下書きの件数 = %d、期待値 = 1", len(groups[1].DraftPages))
	}
	if groups[1].TopicIconName != "lock-regular" {
		t.Errorf("非公開トピックのアイコン = %s、期待値 = lock-regular", groups[1].TopicIconName)
	}
}

func TestDraftPageForIndex_DisplayTitle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, "ja")

	now := time.Now()
	drafts := []*model.DraftPage{
		{
			ID:         "dp1",
			Title:      strPtr("カスタムタイトル"),
			ModifiedAt: now,
			Page:       &model.Page{ID: "p1", Number: 1},
			Topic: &model.Topic{
				Name:       "T",
				Visibility: model.TopicVisibilityPublic,
				Space:      &model.Space{Identifier: "s", Name: "S"},
			},
		},
		{
			ID:         "dp2",
			Title:      nil,
			ModifiedAt: now,
			Page:       &model.Page{ID: "p2", Number: 2, Title: strPtr("公開タイトル")},
			Topic: &model.Topic{
				Name:       "T",
				Visibility: model.TopicVisibilityPublic,
				Space:      &model.Space{Identifier: "s", Name: "S"},
			},
		},
		{
			ID:         "dp3",
			Title:      nil,
			ModifiedAt: now,
			Page:       &model.Page{ID: "p3", Number: 3},
			Topic: &model.Topic{
				Name:       "T",
				Visibility: model.TopicVisibilityPublic,
				Space:      &model.Space{Identifier: "s", Name: "S"},
			},
		},
	}

	groups := viewmodel.NewDraftPageGroupsForIndex(drafts)
	if len(groups) != 1 {
		t.Fatalf("グループの件数 = %d、期待値 = 1", len(groups))
	}

	pages := groups[0].DraftPages

	if got := pages[0].DisplayTitle(ctx); got != "カスタムタイトル" {
		t.Errorf("実測値 = %s、期待値 = カスタムタイトル", got)
	}

	if got := pages[1].DisplayTitle(ctx); got != "公開タイトル" {
		t.Errorf("実測値 = %s、期待値 = 公開タイトル", got)
	}

	if got := pages[2].DisplayTitle(ctx); got != "無題" {
		t.Errorf("実測値 = %s、期待値 = 無題", got)
	}
}

func strPtr(s string) *string {
	return &s
}
