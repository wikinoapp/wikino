package policy

import (
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func newSpaceMember(spaceID model.SpaceID, role model.SpaceMemberRole, active bool) *model.SpaceMember {
	return &model.SpaceMember{
		ID:       "sm-1",
		SpaceID:  spaceID,
		UserID:   "user-1",
		Role:     role,
		JoinedAt: time.Now(),
		Active:   active,
	}
}

func newPage(spaceID model.SpaceID, topicID model.TopicID) *model.Page {
	return &model.Page{
		ID:         "page-1",
		SpaceID:    spaceID,
		TopicID:    topicID,
		Number:     1,
		Body:       "",
		BodyHTML:   "",
		ModifiedAt: time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func newDraftPage(spaceID model.SpaceID, topicID model.TopicID) *model.DraftPage {
	return &model.DraftPage{
		ID:            "draft-1",
		SpaceID:       spaceID,
		PageID:        "page-1",
		SpaceMemberID: "sm-1",
		TopicID:       topicID,
		Body:          "",
		BodyHTML:      "",
		ModifiedAt:    time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

func newSuggestion(spaceID model.SpaceID, topicID model.TopicID) *model.Suggestion {
	return &model.Suggestion{
		ID:                   "suggestion-1",
		SpaceID:              spaceID,
		TopicID:              topicID,
		CreatedSpaceMemberID: "sm-creator",
		Status:               model.SuggestionStatusOpen,
	}
}

func newTopic(spaceID model.SpaceID, topicID model.TopicID) *model.Topic {
	return &model.Topic{
		ID:    topicID,
		Space: &model.Space{ID: spaceID},
		Name:  "test-topic",
	}
}

func TestNewTopicPolicy_SpaceOwner(t *testing.T) {
	t.Parallel()

	t.Run("同じスペースのトピックにページを作成可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if !p.CanCreatePage(newTopic("space-1", "topic-1")) {
			t.Error("スペースオーナーは同じスペースのトピックにページを作成できるべき")
		}
	})

	t.Run("別スペースのトピックにはページを作成不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if p.CanCreatePage(newTopic("space-2", "topic-1")) {
			t.Error("スペースオーナーは別スペースのトピックにページを作成できないべき")
		}
	})

	t.Run("非アクティブの場合はページを作成不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, false)
		p := NewTopicPolicy(sm, nil)

		if p.CanCreatePage(newTopic("space-1", "topic-1")) {
			t.Error("非アクティブなスペースオーナーはページを作成できないべき")
		}
	})

	t.Run("トピックメンバーでない場合も同じスペースのページを編集可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if !p.CanUpdatePage(newPage("space-1", "topic-1")) {
			t.Error("スペースオーナーは同じスペースのページを編集できるべき")
		}
		if !p.CanUpdateDraftPage(newDraftPage("space-1", "topic-1")) {
			t.Error("スペースオーナーは同じスペースのドラフトページを編集できるべき")
		}
	})

	t.Run("トピックメンバーである場合も同じスペースのページを編集可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if !p.CanUpdatePage(newPage("space-1", "topic-1")) {
			t.Error("スペースオーナーは同じスペースのページを編集できるべき")
		}
	})

	t.Run("同じスペース内の別トピックのページも編集可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if !p.CanUpdatePage(newPage("space-1", "topic-2")) {
			t.Error("スペースオーナーは同じスペース内の別トピックのページも編集できるべき")
		}
	})

	t.Run("別スペースのページは編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if p.CanUpdatePage(newPage("space-2", "topic-1")) {
			t.Error("スペースオーナーは別スペースのページを編集できないべき")
		}
		if p.CanUpdateDraftPage(newDraftPage("space-2", "topic-1")) {
			t.Error("スペースオーナーは別スペースのドラフトページを編集できないべき")
		}
	})

	t.Run("非アクティブの場合は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, false)
		p := NewTopicPolicy(sm, nil)

		if p.CanUpdatePage(newPage("space-1", "topic-1")) {
			t.Error("非アクティブなスペースオーナーはページを編集できないべき")
		}
		if p.CanUpdateDraftPage(newDraftPage("space-1", "topic-1")) {
			t.Error("非アクティブなスペースオーナーはドラフトページを編集できないべき")
		}
	})

	t.Run("編集提案を反映可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if !p.CanApplySuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("スペースオーナーは編集提案を反映できるべき")
		}
	})

	t.Run("別スペースの編集提案は反映不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if p.CanApplySuggestion(newSuggestion("space-2", "topic-1")) {
			t.Error("スペースオーナーは別スペースの編集提案を反映できないべき")
		}
	})

	t.Run("非アクティブの場合は編集提案を反映不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, false)
		p := NewTopicPolicy(sm, nil)

		if p.CanApplySuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("非アクティブなスペースオーナーは編集提案を反映できないべき")
		}
	})

	t.Run("編集提案をクローズ可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if !p.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("スペースオーナーは編集提案をクローズできるべき")
		}
	})

	t.Run("別スペースの編集提案はクローズ不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		if p.CanCloseSuggestion(newSuggestion("space-2", "topic-1")) {
			t.Error("スペースオーナーは別スペースの編集提案をクローズできないべき")
		}
	})

	t.Run("非アクティブの場合は編集提案をクローズ不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, false)
		p := NewTopicPolicy(sm, nil)

		if p.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("非アクティブなスペースオーナーは編集提案をクローズできないべき")
		}
	})

	t.Run("オープン状態の編集提案を編集可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		suggestion := newSuggestion("space-1", "topic-1")
		if !p.CanUpdateSuggestion(suggestion) {
			t.Error("スペースオーナーはオープン状態の編集提案を編集できるべき")
		}
		if !p.CanUpdateSuggestionComment(suggestion) {
			t.Error("スペースオーナーはオープン状態の編集提案のコメントを編集できるべき")
		}
	})

	t.Run("クローズ状態の編集提案は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		suggestion := newSuggestion("space-1", "topic-1")
		suggestion.Status = model.SuggestionStatusClosed
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("スペースオーナーはクローズ状態の編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("スペースオーナーはクローズ状態の編集提案のコメントを編集できないべき")
		}
	})

	t.Run("反映済みの編集提案は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		suggestion := newSuggestion("space-1", "topic-1")
		suggestion.Status = model.SuggestionStatusApplied
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("スペースオーナーは反映済みの編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("スペースオーナーは反映済みの編集提案のコメントを編集できないべき")
		}
	})

	t.Run("別スペースの編集提案は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, true)
		p := NewTopicPolicy(sm, nil)

		suggestion := newSuggestion("space-2", "topic-1")
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("スペースオーナーは別スペースの編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("スペースオーナーは別スペースの編集提案のコメントを編集できないべき")
		}
	})

	t.Run("非アクティブの場合は編集提案を編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleOwner, false)
		p := NewTopicPolicy(sm, nil)

		suggestion := newSuggestion("space-1", "topic-1")
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("非アクティブなスペースオーナーは編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("非アクティブなスペースオーナーは編集提案のコメントを編集できないべき")
		}
	})
}

func TestNewTopicPolicy_TopicAdmin(t *testing.T) {
	t.Parallel()

	t.Run("所属トピックにページを作成可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if !p.CanCreatePage(newTopic("space-1", "topic-1")) {
			t.Error("トピックAdminは所属トピックにページを作成できるべき")
		}
	})

	t.Run("別トピックにはページを作成不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanCreatePage(newTopic("space-1", "topic-2")) {
			t.Error("トピックAdminは別トピックにページを作成できないべき")
		}
	})

	t.Run("所属トピックのページを編集可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if !p.CanUpdatePage(newPage("space-1", "topic-1")) {
			t.Error("トピックAdminは所属トピックのページを編集できるべき")
		}
		if !p.CanUpdateDraftPage(newDraftPage("space-1", "topic-1")) {
			t.Error("トピックAdminは所属トピックのドラフトページを編集できるべき")
		}
	})

	t.Run("別トピックのページは編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanUpdatePage(newPage("space-1", "topic-2")) {
			t.Error("トピックAdminは別トピックのページを編集できないべき")
		}
		if p.CanUpdateDraftPage(newDraftPage("space-1", "topic-2")) {
			t.Error("トピックAdminは別トピックのドラフトページを編集できないべき")
		}
	})

	t.Run("非アクティブの場合は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, false)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanUpdatePage(newPage("space-1", "topic-1")) {
			t.Error("非アクティブなトピックAdminはページを編集できないべき")
		}
		if p.CanUpdateDraftPage(newDraftPage("space-1", "topic-1")) {
			t.Error("非アクティブなトピックAdminはドラフトページを編集できないべき")
		}
	})

	t.Run("所属トピックの編集提案を反映可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if !p.CanApplySuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("トピックAdminは所属トピックの編集提案を反映できるべき")
		}
	})

	t.Run("別トピックの編集提案は反映不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanApplySuggestion(newSuggestion("space-1", "topic-2")) {
			t.Error("トピックAdminは別トピックの編集提案を反映できないべき")
		}
	})

	t.Run("非アクティブの場合は編集提案を反映不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, false)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanApplySuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("非アクティブなトピックAdminは編集提案を反映できないべき")
		}
	})

	t.Run("所属トピックの編集提案をクローズ可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if !p.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("トピックAdminは所属トピックの編集提案をクローズできるべき")
		}
	})

	t.Run("別トピックの編集提案はクローズ不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanCloseSuggestion(newSuggestion("space-1", "topic-2")) {
			t.Error("トピックAdminは別トピックの編集提案をクローズできないべき")
		}
	})

	t.Run("非アクティブの場合は編集提案をクローズ不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, false)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("非アクティブなトピックAdminは編集提案をクローズできないべき")
		}
	})

	t.Run("オープン状態の編集提案を編集可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		suggestion := newSuggestion("space-1", "topic-1")
		if !p.CanUpdateSuggestion(suggestion) {
			t.Error("トピックAdminはオープン状態の編集提案を編集できるべき")
		}
		if !p.CanUpdateSuggestionComment(suggestion) {
			t.Error("トピックAdminはオープン状態の編集提案のコメントを編集できるべき")
		}
	})

	t.Run("クローズ状態の編集提案は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		suggestion := newSuggestion("space-1", "topic-1")
		suggestion.Status = model.SuggestionStatusClosed
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("トピックAdminはクローズ状態の編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("トピックAdminはクローズ状態の編集提案のコメントを編集できないべき")
		}
	})

	t.Run("別トピックの編集提案は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		suggestion := newSuggestion("space-1", "topic-2")
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("トピックAdminは別トピックの編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("トピックAdminは別トピックの編集提案のコメントを編集できないべき")
		}
	})

	t.Run("非アクティブの場合は編集提案を編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, false)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleAdmin,
		}
		p := NewTopicPolicy(sm, topicMember)

		suggestion := newSuggestion("space-1", "topic-1")
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("非アクティブなトピックAdminは編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("非アクティブなトピックAdminは編集提案のコメントを編集できないべき")
		}
	})
}

func TestNewTopicPolicy_TopicMember(t *testing.T) {
	t.Parallel()

	t.Run("所属トピックにページを作成可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if !p.CanCreatePage(newTopic("space-1", "topic-1")) {
			t.Error("トピックMemberは所属トピックにページを作成できるべき")
		}
	})

	t.Run("別トピックにはページを作成不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanCreatePage(newTopic("space-1", "topic-2")) {
			t.Error("トピックMemberは別トピックにページを作成できないべき")
		}
	})

	t.Run("所属トピックのページを編集可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if !p.CanUpdatePage(newPage("space-1", "topic-1")) {
			t.Error("トピックMemberは所属トピックのページを編集できるべき")
		}
		if !p.CanUpdateDraftPage(newDraftPage("space-1", "topic-1")) {
			t.Error("トピックMemberは所属トピックのドラフトページを編集できるべき")
		}
	})

	t.Run("別トピックのページは編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanUpdatePage(newPage("space-1", "topic-2")) {
			t.Error("トピックMemberは別トピックのページを編集できないべき")
		}
		if p.CanUpdateDraftPage(newDraftPage("space-1", "topic-2")) {
			t.Error("トピックMemberは別トピックのドラフトページを編集できないべき")
		}
	})

	t.Run("非アクティブの場合は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, false)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanUpdatePage(newPage("space-1", "topic-1")) {
			t.Error("非アクティブなトピックMemberはページを編集できないべき")
		}
		if p.CanUpdateDraftPage(newDraftPage("space-1", "topic-1")) {
			t.Error("非アクティブなトピックMemberはドラフトページを編集できないべき")
		}
	})

	t.Run("編集提案は反映不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanApplySuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("トピックMemberは編集提案を反映できないべき")
		}
	})

	t.Run("作成者本人は編集提案をクローズ可能", func(t *testing.T) {
		t.Parallel()

		sm := &model.SpaceMember{
			ID:      "sm-creator",
			SpaceID: "space-1",
			UserID:  "user-1",
			Role:    model.SpaceMemberRoleMember,
			Active:  true,
		}
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if !p.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("作成者本人は編集提案をクローズできるべき")
		}
	})

	t.Run("作成者でないトピックMemberは編集提案をクローズ不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("作成者でないトピックMemberは編集提案をクローズできないべき")
		}
	})

	t.Run("非アクティブの場合は作成者でもクローズ不可", func(t *testing.T) {
		t.Parallel()

		sm := &model.SpaceMember{
			ID:      "sm-creator",
			SpaceID: "space-1",
			UserID:  "user-1",
			Role:    model.SpaceMemberRoleMember,
			Active:  false,
		}
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		if p.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("非アクティブな作成者は編集提案をクローズできないべき")
		}
	})

	t.Run("オープン状態の編集提案を編集可能", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		suggestion := newSuggestion("space-1", "topic-1")
		if !p.CanUpdateSuggestion(suggestion) {
			t.Error("トピックMemberはオープン状態の編集提案を編集できるべき")
		}
		if !p.CanUpdateSuggestionComment(suggestion) {
			t.Error("トピックMemberはオープン状態の編集提案のコメントを編集できるべき")
		}
	})

	t.Run("クローズ状態の編集提案は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		suggestion := newSuggestion("space-1", "topic-1")
		suggestion.Status = model.SuggestionStatusClosed
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("トピックMemberはクローズ状態の編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("トピックMemberはクローズ状態の編集提案のコメントを編集できないべき")
		}
	})

	t.Run("別トピックの編集提案は編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		suggestion := newSuggestion("space-1", "topic-2")
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("トピックMemberは別トピックの編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("トピックMemberは別トピックの編集提案のコメントを編集できないべき")
		}
	})

	t.Run("非アクティブの場合は編集提案を編集不可", func(t *testing.T) {
		t.Parallel()

		sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, false)
		topicMember := &model.TopicMember{
			TopicID: "topic-1",
			Role:    model.TopicMemberRoleMember,
		}
		p := NewTopicPolicy(sm, topicMember)

		suggestion := newSuggestion("space-1", "topic-1")
		if p.CanUpdateSuggestion(suggestion) {
			t.Error("非アクティブなトピックMemberは編集提案を編集できないべき")
		}
		if p.CanUpdateSuggestionComment(suggestion) {
			t.Error("非アクティブなトピックMemberは編集提案のコメントを編集できないべき")
		}
	})
}

func TestNewTopicPolicy_Guest(t *testing.T) {
	t.Parallel()

	sm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
	p := NewTopicPolicy(sm, nil)

	if p.CanCreatePage(newTopic("space-1", "topic-1")) {
		t.Error("非トピックメンバーはページを作成できないべき")
	}
	if p.CanUpdatePage(newPage("space-1", "topic-1")) {
		t.Error("非トピックメンバーはページを編集できないべき")
	}
	if p.CanUpdateDraftPage(newDraftPage("space-1", "topic-1")) {
		t.Error("非トピックメンバーはドラフトページを編集できないべき")
	}
	if p.CanApplySuggestion(newSuggestion("space-1", "topic-1")) {
		t.Error("非トピックメンバーは編集提案を反映できないべき")
	}
	if p.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
		t.Error("非トピックメンバーは作成者でなければ編集提案をクローズできないべき")
	}

	t.Run("作成者本人はトピック非メンバーでもクローズ可能", func(t *testing.T) {
		t.Parallel()

		creatorSm := &model.SpaceMember{
			ID:      "sm-creator",
			SpaceID: "space-1",
			UserID:  "user-1",
			Role:    model.SpaceMemberRoleMember,
			Active:  true,
		}
		guestPolicy := NewTopicPolicy(creatorSm, nil)

		if !guestPolicy.CanCloseSuggestion(newSuggestion("space-1", "topic-1")) {
			t.Error("作成者本人はトピック非メンバーでも編集提案をクローズできるべき")
		}
	})

	t.Run("オープン状態の編集提案を編集可能", func(t *testing.T) {
		t.Parallel()

		guestSm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		guestPolicy := NewTopicPolicy(guestSm, nil)

		suggestion := newSuggestion("space-1", "topic-1")
		if !guestPolicy.CanUpdateSuggestion(suggestion) {
			t.Error("非トピックメンバーでもオープン状態の編集提案を編集できるべき")
		}
		if !guestPolicy.CanUpdateSuggestionComment(suggestion) {
			t.Error("非トピックメンバーでもオープン状態の編集提案のコメントを編集できるべき")
		}
	})

	t.Run("クローズ状態の編集提案は編集不可", func(t *testing.T) {
		t.Parallel()

		guestSm := newSpaceMember("space-1", model.SpaceMemberRoleMember, true)
		guestPolicy := NewTopicPolicy(guestSm, nil)

		suggestion := newSuggestion("space-1", "topic-1")
		suggestion.Status = model.SuggestionStatusClosed
		if guestPolicy.CanUpdateSuggestion(suggestion) {
			t.Error("非トピックメンバーはクローズ状態の編集提案を編集できないべき")
		}
		if guestPolicy.CanUpdateSuggestionComment(suggestion) {
			t.Error("非トピックメンバーはクローズ状態の編集提案のコメントを編集できないべき")
		}
	})

	t.Run("非アクティブの場合は編集提案を編集不可", func(t *testing.T) {
		t.Parallel()

		guestSm := newSpaceMember("space-1", model.SpaceMemberRoleMember, false)
		guestPolicy := NewTopicPolicy(guestSm, nil)

		suggestion := newSuggestion("space-1", "topic-1")
		if guestPolicy.CanUpdateSuggestion(suggestion) {
			t.Error("非アクティブな非トピックメンバーは編集提案を編集できないべき")
		}
		if guestPolicy.CanUpdateSuggestionComment(suggestion) {
			t.Error("非アクティブな非トピックメンバーは編集提案のコメントを編集できないべき")
		}
	})
}
