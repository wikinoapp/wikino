package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// GuestPolicy は非ログイン・非スペースメンバー用の権限判定を行う。
// 公開トピック・ページの閲覧のみ許可し、それ以外はすべて拒否する。
type GuestPolicy struct{}

// NewGuestPolicy は GuestPolicy を生成する
func NewGuestPolicy() *GuestPolicy {
	return &GuestPolicy{}
}

func (p *GuestPolicy) CanShowTopic(topic *model.Topic) bool {
	return topic.Visibility == model.TopicVisibilityPublic
}

func (p *GuestPolicy) CanUpdateTopic() bool                                { return false }
func (p *GuestPolicy) CanCreatePage() bool                                 { return false }
func (p *GuestPolicy) CanUpdatePage() bool                                 { return false }
func (p *GuestPolicy) CanShowDraftPage(_ bool) bool                        { return false }
func (p *GuestPolicy) CanUpdateDraftPage(_ bool) bool                      { return false }
func (p *GuestPolicy) CanCreateSuggestion(_ *model.Topic) bool             { return false }
func (p *GuestPolicy) CanApplySuggestion() bool                            { return false }
func (p *GuestPolicy) CanCloseSuggestion(_ bool) bool                      { return false }
func (p *GuestPolicy) CanUpdateSuggestion(_ *model.Suggestion) bool        { return false }
func (p *GuestPolicy) CanAddSuggestionPage(_ *model.Suggestion) bool       { return false }
func (p *GuestPolicy) CanRemoveSuggestionPage(_ *model.Suggestion) bool    { return false }
func (p *GuestPolicy) CanEditSuggestionPage() bool                         { return false }
func (p *GuestPolicy) CanCreateSuggestionComment() bool                    { return false }
func (p *GuestPolicy) CanUpdateSuggestionComment(_ *model.Suggestion) bool { return false }
