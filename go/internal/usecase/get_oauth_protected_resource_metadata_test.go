package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestGetOAuthProtectedResourceMetadataUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("スペースを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("goprm-found").Build()

		output, err := NewGetOAuthProtectedResourceMetadataUsecase(repository.NewSpaceRepository(q)).Execute(context.Background(),
			GetOAuthProtectedResourceMetadataInput{SpaceIdentifier: "goprm-found"})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space.ID != spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, spaceID)
		}
	})

	tests := []struct {
		name      string
		discarded bool
	}{
		{name: "存在しないスペースは未存在", discarded: false},
		{name: "削除済みのスペースは未存在", discarded: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			if tt.discarded {
				testutil.NewSpaceBuilder(t, tx).WithIdentifier("goprm-missing").WithDiscarded().Build()
			}

			_, err := NewGetOAuthProtectedResourceMetadataUsecase(repository.NewSpaceRepository(q)).Execute(context.Background(),
				GetOAuthProtectedResourceMetadataInput{SpaceIdentifier: "goprm-missing"})
			if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
				t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
			}
		})
	}
}
