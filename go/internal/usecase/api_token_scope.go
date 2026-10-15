package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// ineffectiveAPITokenScopesは、candidatesのうちスペースメンバーがスペースのロールでもどの
// トピックのロールでも持たないスコープを、candidatesの順に返す。トークンの発行画面や同意画面で、
// 付与しても今の権限では効かないスコープに注記を添えるために使う
func ineffectiveAPITokenScopes(
	ctx context.Context,
	topicMemberRepo *repository.TopicMemberRepository,
	spaceMember *model.SpaceMember,
	candidates []model.Scope,
) ([]model.Scope, error) {
	topicMembers, err := topicMemberRepo.ListWithRoleBySpaceMember(ctx, spaceMember.SpaceID, spaceMember.ID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	held := policy.HeldAPITokenScopes(spaceMember, topicMembers)
	var ineffective []model.Scope
	for _, s := range candidates {
		if !model.HasScope(held, s) {
			ineffective = append(ineffective, s)
		}
	}
	return ineffective, nil
}
