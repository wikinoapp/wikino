package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

func TestGetSpaceDeletionNewUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	t.Run("管理者にはスペースを返す", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsGeneralFixture(t, "get-space-deletion", model.SpaceRoleAdmin)
		uc := NewGetSpaceDeletionNewUsecase(repository.NewSpaceRepository(f.queries), repository.NewSpaceMemberRepository(f.queries))
		output, err := uc.Execute(ctx, GetSpaceDeletionNewInput{SpaceIdentifier: f.identifier, UserID: f.userID})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space.ID != f.spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
		}
	})

	t.Run("space:deleteを持たないメンバーは拒否する", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsGeneralFixture(t, "get-space-deletion-ed", model.SpaceRoleEditor)
		uc := NewGetSpaceDeletionNewUsecase(repository.NewSpaceRepository(f.queries), repository.NewSpaceMemberRepository(f.queries))
		_, err := uc.Execute(ctx, GetSpaceDeletionNewInput{SpaceIdentifier: f.identifier, UserID: f.userID})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeForbidden {
			t.Errorf("エラー = %v、ForbiddenのAppErrorを期待", err)
		}
	})
}
