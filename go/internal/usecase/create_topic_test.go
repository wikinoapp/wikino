package usecase

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// createTopicFixture is the space and the member a topic-creation test acts as.
//
// [Ja] createTopicFixture はトピック作成のテストが振る舞う元になるスペースとメンバー
type createTopicFixture struct {
	db         *sql.DB
	identifier model.SpaceIdentifier
	spaceID    model.SpaceID
	memberID   model.SpaceMemberID
	userID     model.UserID
}

// setupCreateTopicFixture creates a space with one member holding the given scopes. The suffix
// keeps the unique columns apart between tests, which run in parallel against the same database.
//
// [Ja] setupCreateTopicFixture は、渡したスコープを持つメンバーが 1 人いるスペースを作成する。
// テストは同じデータベースに対して並行に走るため、一意性のある列を suffix で区別する。
func setupCreateTopicFixture(t *testing.T, suffix string, scopes []model.Scope) createTopicFixture {
	t.Helper()

	db := testutil.GetTestDB()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("create-topic-" + suffix + "@example.com").
		WithAtname("create_topic_" + suffix).
		Build()

	identifier := "create-topic-" + suffix
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(identifier).
		Build()

	memberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithScopes(scopes).
		Build()

	return createTopicFixture{
		db:         db,
		identifier: model.SpaceIdentifier(identifier),
		spaceID:    spaceID,
		memberID:   memberID,
		userID:     userID,
	}
}

// newCreateTopicUsecase builds the UseCase against the shared pool.
//
// [Ja] newCreateTopicUsecase は共有プールを使う UseCase を組み立てる。
func newCreateTopicUsecase(f createTopicFixture) *CreateTopicUsecase {
	queries := query.New(f.db)
	topicRepo := repository.NewTopicRepository(queries)
	return NewCreateTopicUsecase(
		f.db,
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		topicRepo,
		repository.NewTopicMemberRepository(queries),
		validator.NewTopicCreateValidator(topicRepo),
	)
}

func TestCreateTopicUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupCreateTopicFixture(t, "success", []model.Scope{model.ScopeSpaceAdmin})
	uc := newCreateTopicUsecase(f)

	output, err := uc.Execute(ctx, CreateTopicInput{
		SpaceIdentifier: f.identifier,
		UserID:          f.userID,
		Name:            "日報",
		Description:     "毎日の記録",
		Visibility:      "private",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Topic.Name != "日報" {
		t.Errorf("Name = %q, want %q", output.Topic.Name, "日報")
	}
	if output.Topic.Description != "毎日の記録" {
		t.Errorf("Description = %q, want %q", output.Topic.Description, "毎日の記録")
	}
	if output.Topic.Visibility != model.TopicVisibilityPrivate {
		t.Errorf("Visibility = %v, want %v", output.Topic.Visibility, model.TopicVisibilityPrivate)
	}
	if output.Topic.Number != 1 {
		t.Errorf("Number = %d, want 1", output.Topic.Number)
	}

	// 作成者がトピックに参加していることを確認する
	topicMemberRepo := repository.NewTopicMemberRepository(query.New(f.db))
	topicMember, err := topicMemberRepo.FindBySpaceMemberAndTopic(ctx, f.spaceID, f.memberID, output.Topic.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topicMember == nil {
		t.Fatal("expected the creator to be a member of the topic but was not")
	}

	// 2つ目のトピックには次の番号が振られる
	second, err := uc.Execute(ctx, CreateTopicInput{
		SpaceIdentifier: f.identifier,
		UserID:          f.userID,
		Name:            "週報",
		Visibility:      "public",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if second.Topic.Number != 2 {
		t.Errorf("Number = %d, want 2", second.Topic.Number)
	}
	if second.Topic.Visibility != model.TopicVisibilityPublic {
		t.Errorf("Visibility = %v, want %v", second.Topic.Visibility, model.TopicVisibilityPublic)
	}
}

// TestCreateTopicUsecase_ExecuteRefused covers the submissions that create nothing: an input the
// validator refuses, a member without the scope to create a topic, and a space that does not exist.
//
// [Ja] TestCreateTopicUsecase_ExecuteRefused は何も作成されない送信を扱う。バリデーターが拒否する
// 入力・トピックを作成するスコープを持たないメンバー・存在しないスペースである。
func TestCreateTopicUsecase_ExecuteRefused(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name            string
		suffix          string
		scopes          []model.Scope
		spaceIdentifier func(f createTopicFixture) model.SpaceIdentifier
		topicName       string
		visibility      string
		wantValidation  bool
		wantAppErrCode  model.AppErrorCode
	}{
		{
			name:            "名前が不正な場合は作成されない",
			suffix:          "invalid",
			scopes:          []model.Scope{model.ScopeSpaceAdmin},
			spaceIdentifier: func(f createTopicFixture) model.SpaceIdentifier { return f.identifier },
			topicName:       "foo/bar",
			visibility:      "public",
			wantValidation:  true,
		},
		{
			name:            "トピック作成権限がない場合は作成されない",
			suffix:          "reader",
			scopes:          []model.Scope{model.ScopePageRead},
			spaceIdentifier: func(f createTopicFixture) model.SpaceIdentifier { return f.identifier },
			topicName:       "日報",
			visibility:      "public",
			wantAppErrCode:  model.AppErrCodeForbidden,
		},
		{
			name:            "存在しないスペースでは作成されない",
			suffix:          "nospace",
			scopes:          []model.Scope{model.ScopeSpaceAdmin},
			spaceIdentifier: func(_ createTopicFixture) model.SpaceIdentifier { return "nonexistent-space" },
			topicName:       "日報",
			visibility:      "public",
			wantAppErrCode:  model.AppErrCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateTopicFixture(t, tt.suffix, tt.scopes)
			uc := newCreateTopicUsecase(f)

			_, err := uc.Execute(ctx, CreateTopicInput{
				SpaceIdentifier: tt.spaceIdentifier(f),
				UserID:          f.userID,
				Name:            tt.topicName,
				Visibility:      tt.visibility,
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

			topicRepo := repository.NewTopicRepository(query.New(f.db))
			topics, err := topicRepo.ListActiveBySpace(ctx, f.spaceID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(topics) != 0 {
				t.Errorf("len(topics) = %d, want 0", len(topics))
			}
		})
	}
}

// TestCreateTopicUsecase_ExecuteConcurrently covers two topics created in the same space at the
// same time. The numbers come from MAX(number) + 1, so without the space lock both would read the
// same number and the later insert would violate topics(space_id, number).
//
// [Ja] TestCreateTopicUsecase_ExecuteConcurrently は同じスペースへ同時に作られる 2 つのトピックを
// 扱う。番号は MAX(number) + 1 で決まるため、スペースのロックが無ければ両者が同じ番号を読み、
// 後から INSERT した側が topics(space_id, number) に違反する。
func TestCreateTopicUsecase_ExecuteConcurrently(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupCreateTopicFixture(t, "concurrent", []model.Scope{model.ScopeSpaceAdmin})
	uc := newCreateTopicUsecase(f)

	names := []string{"日報", "週報"}
	outputs := make([]*CreateTopicOutput, len(names))
	errs := make([]error, len(names))

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i, name := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			outputs[i], errs[i] = uc.Execute(ctx, CreateTopicInput{
				SpaceIdentifier: f.identifier,
				UserID:          f.userID,
				Name:            name,
				Visibility:      "public",
			})
		}()
	}
	close(start)
	wg.Wait()

	numbers := make(map[int32]bool, len(names))
	for i := range names {
		if errs[i] != nil {
			t.Fatalf("unexpected error for %q: %v", names[i], errs[i])
		}
		numbers[outputs[i].Topic.Number] = true
	}
	if len(numbers) != len(names) {
		t.Errorf("topic numbers = %v, want %d distinct numbers", numbers, len(names))
	}

	topicRepo := repository.NewTopicRepository(query.New(f.db))
	topics, err := topicRepo.ListActiveBySpace(ctx, f.spaceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(topics) != len(names) {
		t.Errorf("len(topics) = %d, want %d", len(topics), len(names))
	}
}
