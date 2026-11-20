package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// newGetTopicSettingsUsecaseはフィクスチャのトランザクションを使うUseCaseを組み立てる。
func newGetTopicSettingsUsecase(f topicSettingsGeneralFixture) *GetTopicSettingsUsecase {
	return NewGetTopicSettingsUsecase(
		repository.NewSpaceRepository(f.queries),
		repository.NewSpaceMemberRepository(f.queries),
		repository.NewTopicRepository(f.queries),
		repository.NewTopicMemberRepository(f.queries),
	)
}

// TestGetTopicSettingsUsecase_Executeは、閲覧者のスペースとトピックのロールから、トピックの設定の
// どの項目を出すかを決めることを扱う。
func TestGetTopicSettingsUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name      string
		suffix    string
		spaceRole model.SpaceRole
		topicRole model.TopicRole
		want      TopicSettingsItems
	}{
		{
			name:      "スペースの管理者は一般設定と削除の項目を開ける",
			suffix:    "settings-admin",
			spaceRole: model.SpaceRoleAdmin,
			want:      TopicSettingsItems{CanUpdateTopic: true, CanDeleteTopic: true},
		},
		{
			name:      "トピックの管理者のロールを持つ編集者は一般設定と削除の項目を開ける",
			suffix:    "settings-topic-admin",
			spaceRole: model.SpaceRoleEditor,
			topicRole: model.TopicRoleAdmin,
			want:      TopicSettingsItems{CanUpdateTopic: true, CanDeleteTopic: true},
		},
		{
			name:      "スペースの編集者はトピックを作るためのtopic:writeで一般設定の項目だけを開ける",
			suffix:    "settings-editor",
			spaceRole: model.SpaceRoleEditor,
			want:      TopicSettingsItems{CanUpdateTopic: true, CanDeleteTopic: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupTopicSettingsGeneralFixture(t, tt.suffix, tt.spaceRole, tt.topicRole)
			uc := newGetTopicSettingsUsecase(f)

			output, err := uc.Execute(ctx, GetTopicSettingsInput{
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
			if output.Items != tt.want {
				t.Errorf("Items = %+v、期待値 = %+v", output.Items, tt.want)
			}
		})
	}
}

// TestGetTopicSettingsUsecase_ExecuteRefusedは、開ける項目を持たない閲覧者と、どのトピックも
// 指さないトピック番号を扱う。
func TestGetTopicSettingsUsecase_ExecuteRefused(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name        string
		suffix      string
		spaceRole   model.SpaceRole
		topicNumber int32
		otherUser   bool
		wantCode    model.AppErrorCode
	}{
		{
			name:        "開ける項目を持たない閲覧者は拒否する",
			suffix:      "settings-viewer",
			spaceRole:   model.SpaceRoleViewer,
			topicNumber: 1,
			wantCode:    model.AppErrCodeForbidden,
		},
		{
			name:        "スペースのメンバーでないユーザーは見つからないとして拒否する",
			suffix:      "settings-outsider",
			spaceRole:   model.SpaceRoleAdmin,
			topicNumber: 1,
			otherUser:   true,
			wantCode:    model.AppErrCodeResourceNotFound,
		},
		{
			name:        "存在しないトピックでは拒否する",
			suffix:      "settings-notopic",
			spaceRole:   model.SpaceRoleAdmin,
			topicNumber: 999,
			wantCode:    model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupTopicSettingsGeneralFixture(t, tt.suffix, tt.spaceRole, "")
			userID := f.userID
			if tt.otherUser {
				// 別のスペースのメンバーを用意し、このスペースには属さないユーザーとして使う
				other := setupTopicSettingsGeneralFixture(t, tt.suffix+"-other", model.SpaceRoleAdmin, "")
				userID = other.userID
			}
			uc := newGetTopicSettingsUsecase(f)

			_, err := uc.Execute(ctx, GetTopicSettingsInput{
				SpaceIdentifier: f.identifier,
				TopicNumber:     tt.topicNumber,
				UserID:          userID,
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
