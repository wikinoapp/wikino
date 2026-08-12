package policy

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestGuestPolicy_CanShowTopic(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックは閲覧可能", func(t *testing.T) {
		t.Parallel()

		p := NewGuestPolicy()
		topic := &model.Topic{Visibility: model.TopicVisibilityPublic}

		if !p.CanShowTopic(topic) {
			t.Error("ゲストは公開トピックを閲覧可能であるべき")
		}
	})

	t.Run("非公開トピックは閲覧不可", func(t *testing.T) {
		t.Parallel()

		p := NewGuestPolicy()
		topic := &model.Topic{Visibility: model.TopicVisibilityPrivate}

		if p.CanShowTopic(topic) {
			t.Error("ゲストは非公開トピックを閲覧できないべき")
		}
	})
}

func TestGuestPolicy_AllRestrictedMethodsReturnFalse(t *testing.T) {
	t.Parallel()

	p := NewGuestPolicy()
	publicTopic := &model.Topic{Visibility: model.TopicVisibilityPublic}
	openSuggestion := &model.Suggestion{Status: model.SuggestionStatusOpen}

	tests := []struct {
		name   string
		result bool
	}{
		{"CanCreatePage", p.CanCreatePage()},
		{"CanUpdatePage", p.CanUpdatePage()},
		{"CanShowTrash", p.CanShowTrash()},
		{"CanShowDraftPage(owner)", p.CanShowDraftPage(true)},
		{"CanShowDraftPage(non-owner)", p.CanShowDraftPage(false)},
		{"CanUpdateDraftPage(owner)", p.CanUpdateDraftPage(true)},
		{"CanUpdateDraftPage(non-owner)", p.CanUpdateDraftPage(false)},
		{"CanDeleteDraftPage", p.CanDeleteDraftPage()},
		{"CanCreateSuggestion(public)", p.CanCreateSuggestion(publicTopic)},
		{"CanApplySuggestion", p.CanApplySuggestion()},
		{"CanCloseSuggestion(creator)", p.CanCloseSuggestion(true)},
		{"CanCloseSuggestion(non-creator)", p.CanCloseSuggestion(false)},
		{"CanUpdateSuggestion", p.CanUpdateSuggestion(openSuggestion)},
		{"CanAddSuggestionPage", p.CanAddSuggestionPage(openSuggestion)},
		{"CanRemoveSuggestionPage", p.CanRemoveSuggestionPage(openSuggestion)},
		{"CanEditSuggestionPage", p.CanEditSuggestionPage()},
		{"CanCreateSuggestionComment", p.CanCreateSuggestionComment()},
		{"CanUpdateSuggestionComment", p.CanUpdateSuggestionComment(openSuggestion)},
		{"CanCreateTopic", p.CanCreateTopic()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.result {
				t.Errorf("ゲストの %s は false であるべき", tt.name)
			}
		})
	}
}

// TestGuestPolicy_Authorizer は GuestPolicy が Authorizer インターフェースを満たすことを検証する
func TestGuestPolicy_Authorizer(t *testing.T) {
	t.Parallel()

	var _ Authorizer = NewGuestPolicy()
}
