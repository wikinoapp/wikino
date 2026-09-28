package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestGetAPISpaceUsecase_Execute(t *testing.T) {
	t.Parallel()

	space := &model.Space{ID: "space-1", Identifier: "bound-space", Name: "束縛先のスペース"}
	principal := &model.APIPrincipal{
		User:  &model.User{ID: "user-1"},
		Space: space,
	}

	t.Run("束縛先のスペースを返す", func(t *testing.T) {
		t.Parallel()

		output, err := NewGetAPISpaceUsecase().Execute(context.Background(), GetAPISpaceInput{
			Principal:       principal,
			SpaceIdentifier: "bound-space",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output.Space != space {
			t.Errorf("Space = %+v、期待値 = %+v", output.Space, space)
		}
	})

	notFoundTests := []struct {
		name       string
		principal  *model.APIPrincipal
		identifier model.SpaceIdentifier
	}{
		{name: "束縛先と異なるスペースは未存在として扱う", principal: principal, identifier: "other-space"},
		{name: "識別子の大文字小文字の違いも別のスペースとして扱う", principal: principal, identifier: "Bound-Space"},
		{name: "主体が無い場合は未存在として扱う", principal: nil, identifier: "bound-space"},
	}
	for _, tt := range notFoundTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output, err := NewGetAPISpaceUsecase().Execute(context.Background(), GetAPISpaceInput{
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
