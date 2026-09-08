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

// topicSettingsGeneralFixture is the space, the member and the topic a general-settings UseCase
// test acts on. Its queries are bound to the transaction that owns the fixture.
//
// [Ja] topicSettingsGeneralFixture は一般設定の UseCase テストが対象にするスペース・メンバー・
// トピック。queries はフィクスチャを所有するトランザクションに束縛されている。
type topicSettingsGeneralFixture struct {
	queries    *query.Queries
	identifier model.SpaceIdentifier
	spaceID    model.SpaceID
	topicID    model.TopicID
	userID     model.UserID
}

// setupTopicSettingsGeneralFixture creates a transaction-isolated space holding one topic, with
// one member of the given scopes. The suffix keeps unique columns apart while parallel
// transactions are open.
//
// [Ja] setupTopicSettingsGeneralFixture は、トランザクションで分離された、トピックを 1 つ持つ
// スペースと、渡したスコープを持つメンバー 1 人を作成する。並行トランザクションが開いている間も
// 一意な列が衝突しないよう、suffix で区別する。
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

// newUpdateTopicUsecase builds the UseCase against the fixture transaction.
//
// [Ja] newUpdateTopicUsecase はフィクスチャのトランザクションを使う UseCase を組み立てる。
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
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Topic.ID != f.topicID {
		t.Errorf("ID = %v, want %v", output.Topic.ID, f.topicID)
	}
	if output.Topic.Number != 1 {
		t.Errorf("Number = %d, want 1", output.Topic.Number)
	}

	topicRepo := repository.NewTopicRepository(f.queries)
	stored, err := topicRepo.FindBySpaceAndID(ctx, f.spaceID, f.topicID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Name != "週報" {
		t.Errorf("Name = %q, want %q", stored.Name, "週報")
	}
	if stored.Description != "毎週の記録" {
		t.Errorf("Description = %q, want %q", stored.Description, "毎週の記録")
	}
	if stored.Visibility != model.TopicVisibilityPrivate {
		t.Errorf("Visibility = %v, want %v", stored.Visibility, model.TopicVisibilityPrivate)
	}
}

// TestUpdateTopicUsecase_ExecuteKeepsOwnName covers the submission that leaves the name as it is.
// The uniqueness check counts the topics of the space, so it must not find the topic being updated.
//
// [Ja] TestUpdateTopicUsecase_ExecuteKeepsOwnName は名前をそのままにした送信を扱う。一意性チェックは
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
		t.Fatalf("unexpected error: %v", err)
	}

	topicRepo := repository.NewTopicRepository(f.queries)
	stored, err := topicRepo.FindBySpaceAndID(ctx, f.spaceID, f.topicID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Description != "説明だけ変える" {
		t.Errorf("Description = %q, want %q", stored.Description, "説明だけ変える")
	}
}

// TestUpdateTopicUsecase_ExecuteRefused covers the submissions that change nothing: an input the
// validator refuses, a member without the scope to change a topic, and a topic number that belongs
// to no topic.
//
// [Ja] TestUpdateTopicUsecase_ExecuteRefused は何も変わらない送信を扱う。バリデーターが拒否する
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
				t.Fatal("expected error but got nil")
			}

			if tt.wantValidation {
				if ve := model.AsValidationError(err); ve == nil {
					t.Fatalf("expected ValidationError but got %v", err)
				}
			} else {
				ae := model.AsAppError(err)
				if ae == nil {
					t.Fatalf("expected AppError but got %v", err)
				}
				if ae.Code != tt.wantAppErrCode {
					t.Errorf("Code = %v, want %v", ae.Code, tt.wantAppErrCode)
				}
			}

			topicRepo := repository.NewTopicRepository(f.queries)
			stored, err := topicRepo.FindBySpaceAndID(ctx, f.spaceID, f.topicID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored.Name != "日報 "+tt.suffix {
				t.Errorf("Name = %q, want %q", stored.Name, "日報 "+tt.suffix)
			}
		})
	}
}
