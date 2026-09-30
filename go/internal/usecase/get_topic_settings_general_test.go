package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// newGetTopicSettingsGeneralUsecaseはフィクスチャのトランザクションを使うUseCaseを
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
	f := setupTopicSettingsGeneralFixture(t, "get-success", model.SpaceRoleAdmin, "")
	uc := newGetTopicSettingsGeneralUsecase(f)

	output, err := uc.Execute(ctx, GetTopicSettingsGeneralInput{
		SpaceIdentifier: f.identifier,
		TopicNumber:     1,
		UserID:          f.userID,
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if output.Space.ID != f.spaceID {
		t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
	}
	if output.Topic.ID != f.topicID {
		t.Errorf("Topic.ID = %v、期待値 = %v", output.Topic.ID, f.topicID)
	}
}

// TestGetTopicSettingsGeneralUsecase_ExecuteRefusedは、トピックを更新できない閲覧者と、
// どのトピックも指さないトピック番号を扱う。
func TestGetTopicSettingsGeneralUsecase_ExecuteRefused(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name        string
		suffix      string
		spaceRole   model.SpaceRole
		topicNumber int32
		wantCode    model.AppErrorCode
	}{
		{
			name:        "トピック更新権限がない場合は拒否する",
			suffix:      "get-reader",
			spaceRole:   model.SpaceRoleViewer,
			topicNumber: 1,
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "存在しないトピックでは拒否する",
			suffix:      "get-notopic",
			spaceRole:   model.SpaceRoleAdmin,
			topicNumber: 999,
			wantCode:    model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupTopicSettingsGeneralFixture(t, tt.suffix, tt.spaceRole, "")
			uc := newGetTopicSettingsGeneralUsecase(f)

			_, err := uc.Execute(ctx, GetTopicSettingsGeneralInput{
				SpaceIdentifier: f.identifier,
				TopicNumber:     tt.topicNumber,
				UserID:          f.userID,
			})
			if err == nil {
				t.Fatal("エラーを期待したが、nilだった")
			}

			ae := model.AsAppError(err)
			if ae == nil {
				t.Fatalf("AppErrorを期待したが、%vだった", err)
			}
			if ae.Code != tt.wantCode {
				t.Errorf("Code = %v、期待値 = %v", ae.Code, tt.wantCode)
			}
		})
	}
}

// TestGetTopicSettingsGeneralUsecase_ExecuteCanUpdateVisibilityは、画面が公開範囲の選択肢を
// 出すかどうかをtopic_visibility:writeで決めることを扱う。
func TestGetTopicSettingsGeneralUsecase_ExecuteCanUpdateVisibility(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name      string
		suffix    string
		spaceRole model.SpaceRole
		topicRole model.TopicRole
		want      bool
	}{
		{
			name:      "topic:writeだけを持つ編集者は公開範囲を変えられない",
			suffix:    "get-writer",
			spaceRole: model.SpaceRoleEditor,
			want:      false,
		},
		{
			name:      "トピック管理者のロールでtopic_visibility:writeを持つメンバーは公開範囲を変えられる",
			suffix:    "get-visibility-writer",
			spaceRole: model.SpaceRoleEditor,
			topicRole: model.TopicRoleAdmin,
			want:      true,
		},
		{
			name:      "管理者は公開範囲を変えられる",
			suffix:    "get-admin",
			spaceRole: model.SpaceRoleAdmin,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupTopicSettingsGeneralFixture(t, tt.suffix, tt.spaceRole, tt.topicRole)
			uc := newGetTopicSettingsGeneralUsecase(f)

			output, err := uc.Execute(ctx, GetTopicSettingsGeneralInput{
				SpaceIdentifier: f.identifier,
				TopicNumber:     1,
				UserID:          f.userID,
			})
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			if output.CanUpdateVisibility != tt.want {
				t.Errorf("CanUpdateVisibility = %v、期待値 = %v", output.CanUpdateVisibility, tt.want)
			}
		})
	}
}
