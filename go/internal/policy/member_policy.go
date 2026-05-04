package policy

import (
	"github.com/wikinoapp/wikino/go/internal/model"
)

// MemberPolicy はスペース���ンバー用の権限判定を行う。
// スペーススコープとトピックスコープの和集合を含意展開し、有効スコープの集合として保持する。
type MemberPolicy struct {
	effectiveScopes map[model.Scope]bool
}

// NewMemberPolicy はスペーススコープとトピックスコープから MemberPolicy を生成する。
// 両スコープの和集合を取り、含意ルールで展開する。
func NewMemberPolicy(spaceScopes, topicScopes []model.Scope) *MemberPolicy {
	merged := make([]model.Scope, 0, len(spaceScopes)+len(topicScopes))
	merged = append(merged, spaceScopes...)
	merged = append(merged, topicScopes...)
	expanded := expandScopes(merged)
	m := make(map[model.Scope]bool, len(expanded))
	for _, s := range expanded {
		m[s] = true
	}
	return &MemberPolicy{effectiveScopes: m}
}

func (p *MemberPolicy) CanShowTopic(topic *model.Topic) bool {
	if topic.Visibility == model.TopicVisibilityPublic {
		return true
	}
	return p.effectiveScopes[model.ScopeTopicRead]
}

func (p *MemberPolicy) CanUpdateTopic() bool {
	return p.effectiveScopes[model.ScopeTopicWrite]
}

func (p *MemberPolicy) CanCreatePage() bool {
	return p.effectiveScopes[model.ScopePageWrite]
}

func (p *MemberPolicy) CanUpdatePage() bool {
	return p.effectiveScopes[model.ScopePageWrite]
}

func (p *MemberPolicy) CanShowDraftPage(isOwner bool) bool {
	if !p.effectiveScopes[model.ScopeDraftPageRead] {
		return false
	}
	return isOwner || p.effectiveScopes[model.ScopeSpaceAdmin]
}

func (p *MemberPolicy) CanUpdateDraftPage(isOwner bool) bool {
	if !p.effectiveScopes[model.ScopeDraftPageWrite] {
		return false
	}
	return isOwner || p.effectiveScopes[model.ScopeSpaceAdmin]
}

func (p *MemberPolicy) CanDeleteDraftPage() bool {
	return p.effectiveScopes[model.ScopeDraftPageDelete]
}

func (p *MemberPolicy) CanCreateSuggestion(topic *model.Topic) bool {
	if !p.effectiveScopes[model.ScopeSuggestionWrite] {
		return false
	}
	if topic.Visibility == model.TopicVisibilityPublic {
		return true
	}
	return p.effectiveScopes[model.ScopeTopicRead]
}

func (p *MemberPolicy) CanApplySuggestion() bool {
	return p.effectiveScopes[model.ScopeSuggestionApply]
}

func (p *MemberPolicy) CanCloseSuggestion(isCreator bool) bool {
	if p.effectiveScopes[model.ScopeSuggestionClose] {
		return true
	}
	return isCreator
}

func (p *MemberPolicy) CanUpdateSuggestion(suggestion *model.Suggestion) bool {
	return p.effectiveScopes[model.ScopeSuggestionWrite] && suggestion.Status == model.SuggestionStatusOpen
}

func (p *MemberPolicy) CanAddSuggestionPage(suggestion *model.Suggestion) bool {
	return p.effectiveScopes[model.ScopeSuggestionWrite] && suggestion.Status == model.SuggestionStatusOpen
}

func (p *MemberPolicy) CanRemoveSuggestionPage(suggestion *model.Suggestion) bool {
	return p.effectiveScopes[model.ScopeSuggestionWrite] && suggestion.Status == model.SuggestionStatusOpen
}

func (p *MemberPolicy) CanEditSuggestionPage() bool {
	return p.effectiveScopes[model.ScopeSuggestionWrite]
}

func (p *MemberPolicy) CanCreateSuggestionComment() bool {
	return p.effectiveScopes[model.ScopeSuggestionCommentWrite]
}

func (p *MemberPolicy) CanUpdateSuggestionComment(suggestion *model.Suggestion) bool {
	return p.effectiveScopes[model.ScopeSuggestionCommentWrite] && suggestion.Status == model.SuggestionStatusOpen
}
