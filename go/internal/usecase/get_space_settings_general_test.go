package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

func TestGetSpaceSettingsGeneralUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	t.Run("管理者にはスペースを返す", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsGeneralFixture(t, "get-space-general", model.SpaceRoleAdmin)
		uc := NewGetSpaceSettingsGeneralUsecase(repository.NewSpaceRepository(f.queries), repository.NewSpaceMemberRepository(f.queries))
		output, err := uc.Execute(ctx, GetSpaceSettingsGeneralInput{SpaceIdentifier: f.identifier, UserID: f.userID})
		if err != nil {
			t.Fatalf("予期しないエラー: %v", err)
		}
		if output.Space.ID != f.spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
		}
	})

	t.Run("スペースを変更できないメンバーは拒否する", func(t *testing.T) {
		t.Parallel()

		f := setupSpaceSettingsGeneralFixture(t, "get-space-general-ed", model.SpaceRoleEditor)
		uc := NewGetSpaceSettingsGeneralUsecase(repository.NewSpaceRepository(f.queries), repository.NewSpaceMemberRepository(f.queries))
		_, err := uc.Execute(ctx, GetSpaceSettingsGeneralInput{SpaceIdentifier: f.identifier, UserID: f.userID})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeForbidden {
			t.Errorf("エラー = %v、ForbiddenのAppErrorを期待", err)
		}
	})
}
