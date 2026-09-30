package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// topicSettingsGeneralFixtureは一般設定のUseCaseテストが対象にするスペース・メンバー・
// トピック。queriesはフィクスチャを所有するトランザクションに束縛されている。
type topicSettingsGeneralFixture struct {
	queries    *query.Queries
	identifier model.SpaceIdentifier
	spaceID    model.SpaceID
	topicID    model.TopicID
	userID     model.UserID
}

// setupTopicSettingsGeneralFixtureは、トランザクションで分離された、トピックを1つ持つ
// スペースと、渡したスコープを持つメンバー1人を作成する。並行トランザクションが開いている間も
// 一意な列が衝突しないよう、suffixで区別する。
func setupTopicSettingsGeneralFixture(t *testing.T, suffix string, scopes []model.Scope) topicSettingsGeneralFixture {
	t.Helper()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-topic-" + suffix + "@example.com").
		WithAtname("update_topic_" + suffix).
		Build()

	identifier := "update-topic-" + suffix
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		Build()

	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes(scopes).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("日報 " + suffix).
		Build()

	return topicSettingsGeneralFixture{
		queries:    queries,
		identifier: model.SpaceIdentifier(identifier),
		spaceID:    spaceID,
		topicID:    topicID,
		userID:     userID,
	}
}

// newUpdateTopicUsecaseはフィクスチャのトランザクションを使うUseCaseを組み立てる。
func newUpdateTopicUsecase(f topicSettingsGeneralFixture) *UpdateTopicUsecase {
	topicRepo := repository.NewTopicRepository(f.queries)
	return NewUpdateTopicUsecase(
		repository.NewSpaceRepository(f.queries),
		repository.NewSpaceMemberRepository(f.queries),
		topicRepo,
		repository.NewTopicMemberRepository(f.queries),
		validator.NewTopicUpdateValidator(topicRepo),
	)
}

func TestUpdateTopicUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupTopicSettingsGeneralFixture(t, "success", []model.Scope{model.ScopeSpaceAdmin})
	uc := newUpdateTopicUsecase(f)

	output, err := uc.Execute(ctx, UpdateTopicInput{
		SpaceIdentifier: f.identifier,
		TopicNumber:     1,
		UserID:          f.userID,
		Name:            "週報",
		Description:     "毎週の記録",
		Visibility:      "private",
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if output.Topic.ID != f.topicID {
		t.Errorf("ID = %v、期待値 = %v", output.Topic.ID, f.topicID)
	}
	if output.Topic.Number != 1 {
		t.Errorf("Number = %d、期待値 = 1", output.Topic.Number)
	}

	topicRepo := repository.NewTopicRepository(f.queries)
	stored, err := topicRepo.FindBySpaceAndID(ctx, f.spaceID, f.topicID)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if stored.Name != "週報" {
		t.Errorf("Name = %q、期待値 = %q", stored.Name, "週報")
	}
	if stored.Description != "毎週の記録" {
		t.Errorf("Description = %q、期待値 = %q", stored.Description, "毎週の記録")
	}
	if stored.Visibility != model.TopicVisibilityPrivate {
		t.Errorf("Visibility = %v、期待値 = %v", stored.Visibility, model.TopicVisibilityPrivate)
	}
}

// TestUpdateTopicUsecase_ExecuteKeepsOwnNameは名前をそのままにした送信を扱う。一意性チェックは
// スペースのトピックを数えるため、更新するトピック自身を見つけてはいけない。
func TestUpdateTopicUsecase_ExecuteKeepsOwnName(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupTopicSettingsGeneralFixture(t, "samename", []model.Scope{model.ScopeSpaceAdmin})
	uc := newUpdateTopicUsecase(f)

	if _, err := uc.Execute(ctx, UpdateTopicInput{
		SpaceIdentifier: f.identifier,
		TopicNumber:     1,
		UserID:          f.userID,
		Name:            "日報 samename",
		Description:     "説明だけ変える",
		Visibility:      "public",
	}); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	topicRepo := repository.NewTopicRepository(f.queries)
	stored, err := topicRepo.FindBySpaceAndID(ctx, f.spaceID, f.topicID)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if stored.Description != "説明だけ変える" {
		t.Errorf("Description = %q、期待値 = %q", stored.Description, "説明だけ変える")
	}
}

// TestUpdateTopicUsecase_ExecuteRefusedは何も変わらない送信を扱う。バリデーターが拒否する
// 入力・トピックを変更するスコープを持たないメンバー・どのトピックも指さないトピック番号である。
func TestUpdateTopicUsecase_ExecuteRefused(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name           string
		suffix         string
		scopes         []model.Scope
		topicNumber    int32
		topicName      string
		wantValidation bool
		wantAppErrCode model.AppErrorCode
	}{
		{
			name:           "名前が不正な場合は更新されない",
			suffix:         "invalid",
			scopes:         []model.Scope{model.ScopeSpaceAdmin},
			topicNumber:    1,
			topicName:      "foo/bar",
			wantValidation: true,
		},
		{
			name:           "トピック更新権限がない場合は更新されない",
			suffix:         "reader",
			scopes:         []model.Scope{model.ScopePageRead},
			topicNumber:    1,
			topicName:      "週報",
			wantAppErrCode: model.AppErrCodeForbidden,
		},
		{
			name:           "存在しないトピックでは更新されない",
			suffix:         "notopic",
			scopes:         []model.Scope{model.ScopeSpaceAdmin},
			topicNumber:    999,
			topicName:      "週報",
			wantAppErrCode: model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupTopicSettingsGeneralFixture(t, tt.suffix, tt.scopes)
			uc := newUpdateTopicUsecase(f)

			_, err := uc.Execute(ctx, UpdateTopicInput{
				SpaceIdentifier: f.identifier,
				TopicNumber:     tt.topicNumber,
				UserID:          f.userID,
				Name:            tt.topicName,
				Visibility:      "public",
			})
			if err == nil {
				t.Fatal("エラーを期待したが、nilだった")
			}

			if tt.wantValidation {
				if ve := model.AsValidationError(err); ve == nil {
					t.Fatalf("ValidationErrorを期待したが、%vだった", err)
				}
			} else {
				ae := model.AsAppError(err)
				if ae == nil {
					t.Fatalf("AppErrorを期待したが、%vだった", err)
				}
				if ae.Code != tt.wantAppErrCode {
					t.Errorf("Code = %v、期待値 = %v", ae.Code, tt.wantAppErrCode)
				}
			}

			topicRepo := repository.NewTopicRepository(f.queries)
			stored, err := topicRepo.FindBySpaceAndID(ctx, f.spaceID, f.topicID)
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if stored.Name != "日報 "+tt.suffix {
				t.Errorf("Name = %q、期待値 = %q", stored.Name, "日報 "+tt.suffix)
			}
		})
	}
}

// TestUpdateTopicUsecase_ExecuteVisibilityは、公開範囲の変更をtopic_visibility:writeで判定し、
// topic:writeだけのメンバーには名前と説明の変更だけを許すことを扱う。フィクスチャのトピックは公開である。
func TestUpdateTopicUsecase_ExecuteVisibility(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name           string
		suffix         string
		scopes         []model.Scope
		visibility     string
		wantForbidden  bool
		wantVisibility model.TopicVisibility
	}{
		{
			name:           "topic:writeだけのメンバーは公開範囲を送らなければ名前と説明を変えられる",
			suffix:         "writer-omit",
			scopes:         []model.Scope{model.ScopeTopicWrite},
			visibility:     "",
			wantVisibility: model.TopicVisibilityPublic,
		},
		{
			name:           "topic:writeだけのメンバーは今と同じ公開範囲を送っても名前と説明を変えられる",
			suffix:         "writer-same",
			scopes:         []model.Scope{model.ScopeTopicWrite},
			visibility:     "public",
			wantVisibility: model.TopicVisibilityPublic,
		},
		{
			name:           "topic:writeだけのメンバーは公開範囲を変えられない",
			suffix:         "writer-change",
			scopes:         []model.Scope{model.ScopeTopicWrite},
			visibility:     "private",
			wantForbidden:  true,
			wantVisibility: model.TopicVisibilityPublic,
		},
		{
			name:           "topic_visibility:writeを持つメンバーは公開範囲を変えられる",
			suffix:         "visibility-writer",
			scopes:         []model.Scope{model.ScopeTopicWrite, model.ScopeTopicVisibilityWrite},
			visibility:     "private",
			wantVisibility: model.TopicVisibilityPrivate,
		},
		{
			name:           "space:adminを持つメンバーは公開範囲を変えられる",
			suffix:         "visibility-admin",
			scopes:         []model.Scope{model.ScopeSpaceAdmin},
			visibility:     "private",
			wantVisibility: model.TopicVisibilityPrivate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupTopicSettingsGeneralFixture(t, tt.suffix, tt.scopes)
			uc := newUpdateTopicUsecase(f)

			_, err := uc.Execute(ctx, UpdateTopicInput{
				SpaceIdentifier: f.identifier,
				TopicNumber:     1,
				UserID:          f.userID,
				Name:            "週報 " + tt.suffix,
				Description:     "毎週の記録",
				Visibility:      tt.visibility,
			})

			wantName := "週報 " + tt.suffix
			if tt.wantForbidden {
				ae := model.AsAppError(err)
				if ae == nil {
					t.Fatalf("AppErrorを期待したが、%vだった", err)
				}
				if ae.Code != model.AppErrCodeForbidden {
					t.Errorf("Code = %v、期待値 = %v", ae.Code, model.AppErrCodeForbidden)
				}
				wantName = "日報 " + tt.suffix
			} else if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			topicRepo := repository.NewTopicRepository(f.queries)
			stored, err := topicRepo.FindBySpaceAndID(ctx, f.spaceID, f.topicID)
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if stored.Name != wantName {
				t.Errorf("Name = %q、期待値 = %q", stored.Name, wantName)
			}
			if stored.Visibility != tt.wantVisibility {
				t.Errorf("Visibility = %v、期待値 = %v", stored.Visibility, tt.wantVisibility)
			}
		})
	}
}

// TestUpdateTopicUsecase_PersistTopicUpdateKeepsChangedVisibilityは、公開範囲を変えられない
// メンバーの保存が、取得後に管理者が変えた公開範囲を上書きしないことを扱う。取得と保存の間に
// 管理者の更新を挟み、競合時の順序を待ち時間に依存せず再現する。
func TestUpdateTopicUsecase_PersistTopicUpdateKeepsChangedVisibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before model.TopicVisibility
		after  model.TopicVisibility
	}{
		{name: "private", before: model.TopicVisibilityPublic, after: model.TopicVisibilityPrivate},
		{name: "public", before: model.TopicVisibilityPrivate, after: model.TopicVisibilityPublic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			f := setupTopicSettingsGeneralFixture(t, "concurrent-"+tt.name, []model.Scope{model.ScopeTopicWrite})
			uc := newUpdateTopicUsecase(f)
			if _, err := uc.topicRepo.Update(ctx, repository.UpdateTopicInput{
				ID: f.topicID, SpaceID: f.spaceID, Name: "日報", Visibility: tt.before,
			}); err != nil {
				t.Fatalf("初期値の設定に失敗: %v", err)
			}

			access, err := fetchTopicUpdateAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.topicRepo, uc.topicMemberRepo, f.identifier, 1, f.userID)
			if err != nil {
				t.Fatalf("認可チェックに失敗: %v", err)
			}
			if access.CanUpdateVisibility {
				t.Fatal("公開範囲を変更できないメンバーを期待した")
			}
			input := UpdateTopicInput{
				SpaceIdentifier: f.identifier, TopicNumber: 1, UserID: f.userID,
				Name: "週報", Description: "毎週の記録",
			}

			// メンバーの認可の後、保存の前に管理者が公開範囲を変更する。
			if _, err := uc.topicRepo.Update(ctx, repository.UpdateTopicInput{
				ID: f.topicID, SpaceID: f.spaceID, Name: "日報", Visibility: tt.after,
			}); err != nil {
				t.Fatalf("公開範囲の変更に失敗: %v", err)
			}
			updated, err := uc.persistTopicUpdate(ctx, input, access, access.Topic.Visibility)
			if err != nil {
				t.Fatalf("名前と説明の保存に失敗: %v", err)
			}
			if updated.Visibility != tt.after {
				t.Errorf("返された公開範囲 = %v、期待値 = %v", updated.Visibility, tt.after)
			}
			stored, err := uc.topicRepo.FindBySpaceAndID(ctx, f.spaceID, f.topicID)
			if err != nil {
				t.Fatalf("保存結果の取得に失敗: %v", err)
			}
			if stored.Name != input.Name || stored.Description != input.Description || stored.Visibility != tt.after {
				t.Errorf("管理者の公開範囲の変更を保って名前と説明を保存できていない: %+v", stored)
			}
		})
	}
}
