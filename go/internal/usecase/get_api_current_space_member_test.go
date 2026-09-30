package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestGetAPICurrentSpaceMemberUsecase_Execute(t *testing.T) {
	t.Parallel()

	user := &model.User{ID: "user-1", Atname: "alice", Name: "Alice"}
	spaceMember := &model.SpaceMember{ID: "space-member-1", SpaceID: "space-1", UserID: "user-1"}
	principal := &model.APIPrincipal{
		User:        user,
		Space:       &model.Space{ID: "space-1", Identifier: "bound-space"},
		SpaceMember: spaceMember,
	}

	t.Run("トークンの持ち主のメンバーとユーザーを返す", func(t *testing.T) {
		t.Parallel()

		output, err := NewGetAPICurrentSpaceMemberUsecase().Execute(context.Background(), GetAPICurrentSpaceMemberInput{
			Principal:       principal,
			SpaceIdentifier: "bound-space",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.SpaceMember != spaceMember {
			t.Errorf("SpaceMember = %+v、期待値 = %+v", output.SpaceMember, spaceMember)
		}
		if output.User != user {
			t.Errorf("User = %+v、期待値 = %+v", output.User, user)
		}
	})

	notFoundTests := []struct {
		name       string
		principal  *model.APIPrincipal
		identifier model.SpaceIdentifier
	}{
		{name: "束縛先と異なるスペースは未存在として扱う", principal: principal, identifier: "other-space"},
		{name: "主体が無い場合は未存在として扱う", principal: nil, identifier: "bound-space"},
	}
	for _, tt := range notFoundTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := NewGetAPICurrentSpaceMemberUsecase().Execute(context.Background(), GetAPICurrentSpaceMemberInput{
				Principal:       tt.principal,
				SpaceIdentifier: tt.identifier,
			})
			if output != nil {
				t.Errorf("output = %+v、期待値 = nil", output)
			}
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
				t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
			}
		})
	}
}
