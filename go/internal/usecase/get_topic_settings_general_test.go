package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// newGetTopicSettingsGeneralUsecase builds the UseCase against the fixture transaction.
//
// [Ja] newGetTopicSettingsGeneralUsecase はフィクスチャのトランザクションを使う UseCase を
// 組み立てる。
func newGetTopicSettingsGeneralUsecase(f topicSettingsGeneralFixture) *GetTopicSettingsGeneralUsecase {
	return NewGetTopicSettingsGeneralUsecase(
		repository.NewSpaceRepository(f.queries),
		repository.NewSpaceMemberRepository(f.queries),
		repository.NewTopicRepository(f.queries),
		repository.NewTopicMemberRepository(f.queries),
	)
}

func TestGetTopicSettingsGeneralUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupTopicSettingsGeneralFixture(t, "get-success", []model.Scope{model.ScopeSpaceAdmin})
	uc := newGetTopicSettingsGeneralUsecase(f)

	output, err := uc.Execute(ctx, GetTopicSettingsGeneralInput{
		SpaceIdentifier: f.identifier,
		TopicNumber:     1,
		UserID:          f.userID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Space.ID != f.spaceID {
		t.Errorf("Space.ID = %v, want %v", output.Space.ID, f.spaceID)
	}
	if output.Topic.ID != f.topicID {
		t.Errorf("Topic.ID = %v, want %v", output.Topic.ID, f.topicID)
	}
}

// TestGetTopicSettingsGeneralUsecase_ExecuteRefused covers viewers who may not update a topic and
// topic numbers that name no topic.
//
// [Ja] TestGetTopicSettingsGeneralUsecase_ExecuteRefused は、トピックを更新できない閲覧者と、
// どのトピックも指さないトピック番号を扱う。
func TestGetTopicSettingsGeneralUsecase_ExecuteRefused(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name        string
		suffix      string
		scopes      []model.Scope
		topicNumber int32
		wantCode    model.AppErrorCode
	}{
		{
			name:        "トピック更新権限がない場合は拒否する",
			suffix:      "get-reader",
			scopes:      []model.Scope{model.ScopePageRead},
			topicNumber: 1,
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "存在しないトピックでは拒否する",
			suffix:      "get-notopic",
			scopes:      []model.Scope{model.ScopeSpaceAdmin},
			topicNumber: 999,
			wantCode:    model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupTopicSettingsGeneralFixture(t, tt.suffix, tt.scopes)
			uc := newGetTopicSettingsGeneralUsecase(f)

			_, err := uc.Execute(ctx, GetTopicSettingsGeneralInput{
				SpaceIdentifier: f.identifier,
				TopicNumber:     tt.topicNumber,
				UserID:          f.userID,
			})
			if err == nil {
				t.Fatal("expected error but got nil")
			}

			ae := model.AsAppError(err)
			if ae == nil {
				t.Fatalf("expected AppError but got %v", err)
			}
			if ae.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", ae.Code, tt.wantCode)
			}
		})
	}
}
