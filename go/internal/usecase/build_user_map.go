package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// buildUserMapBySpaceMemberIDs はSpaceMemberIDのスライスからSpaceMemberID → Userのマップを構築する
func buildUserMapBySpaceMemberIDs(ctx context.Context, spaceMemberRepo *repository.SpaceMemberRepository, userRepo *repository.UserRepository, memberIDs []model.SpaceMemberID, spaceID model.SpaceID) (map[model.SpaceMemberID]*model.User, error) {
	if len(memberIDs) == 0 {
		return map[model.SpaceMemberID]*model.User{}, nil
	}

	// SpaceMemberを一括取得
	members, err := spaceMemberRepo.FindByIDs(ctx, memberIDs, spaceID)
	if err != nil {
		return nil, err
	}

	// UserIDを収集
	userIDs := make([]model.UserID, 0, len(members))
	memberToUser := make(map[model.SpaceMemberID]model.UserID, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
		memberToUser[m.ID] = m.UserID
	}

	// Userを一括取得
	users, err := userRepo.FindByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	userByID := make(map[model.UserID]*model.User, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}

	// SpaceMemberID → User のマップを構築
	result := make(map[model.SpaceMemberID]*model.User, len(memberIDs))
	for memberID, userID := range memberToUser {
		if u, ok := userByID[userID]; ok {
			result[memberID] = u
		}
	}

	return result, nil
}
