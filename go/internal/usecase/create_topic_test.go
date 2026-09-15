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

// createTopicFixtureはトピック作成のテストが振る舞う元になるスペースとメンバー
type createTopicFixture struct {
	db         *sql.DB
	identifier model.SpaceIdentifier
	spaceID    model.SpaceID
	memberID   model.SpaceMemberID
	userID     model.UserID
}

// setupCreateTopicFixtureは、渡したスコープを持つメンバーが1人いるスペースを作成する。
// テストは同じデータベースに対して並行に走るため、一意性のある列をsuffixで区別する。
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

// newCreateTopicUsecaseは共有プールを使うUseCaseを組み立てる。
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
		t.Fatalf("予期しないエラー: %v", err)
	}

	if output.Topic.Name != "日報" {
		t.Errorf("Name = %q、期待値 = %q", output.Topic.Name, "日報")
	}
	if output.Topic.Description != "毎日の記録" {
		t.Errorf("Description = %q、期待値 = %q", output.Topic.Description, "毎日の記録")
	}
	if output.Topic.Visibility != model.TopicVisibilityPrivate {
		t.Errorf("Visibility = %v、期待値 = %v", output.Topic.Visibility, model.TopicVisibilityPrivate)
	}
	if output.Topic.Number != 1 {
		t.Errorf("Number = %d、期待値 = 1", output.Topic.Number)
	}

	// 作成者がトピックに参加していることを確認する
	topicMemberRepo := repository.NewTopicMemberRepository(query.New(f.db))
	topicMember, err := topicMemberRepo.FindBySpaceMemberAndTopic(ctx, f.spaceID, f.memberID, output.Topic.ID)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if topicMember == nil {
		t.Fatal("作成者がトピックのメンバーになっていない")
	}

	// 2つ目のトピックには次の番号が振られる
	second, err := uc.Execute(ctx, CreateTopicInput{
		SpaceIdentifier: f.identifier,
		UserID:          f.userID,
		Name:            "週報",
		Visibility:      "public",
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if second.Topic.Number != 2 {
		t.Errorf("Number = %d、期待値 = 2", second.Topic.Number)
	}
	if second.Topic.Visibility != model.TopicVisibilityPublic {
		t.Errorf("Visibility = %v、期待値 = %v", second.Topic.Visibility, model.TopicVisibilityPublic)
	}
}

// TestCreateTopicUsecase_ExecuteRefusedは何も作成されない送信を扱う。バリデーターが拒否する
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

			topicRepo := repository.NewTopicRepository(query.New(f.db))
			topics, err := topicRepo.ListActiveBySpace(ctx, f.spaceID)
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if len(topics) != 0 {
				t.Errorf("len(topics) = %d、期待値 = 0", len(topics))
			}
		})
	}
}

// TestCreateTopicUsecase_ExecuteConcurrentlyは同じスペースへ同時に作られる2つのトピックを
// 扱う。番号はMAX(number) + 1で決まるため、スペースのロックが無ければ両者が同じ番号を読み、
// 後からINSERTした側がtopics(space_id, number) に違反する。
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
			t.Fatalf("%qで予期しないエラー: %v", names[i], errs[i])
		}
		numbers[outputs[i].Topic.Number] = true
	}
	if len(numbers) != len(names) {
		t.Errorf("トピック番号 = %v、期待値 = 重複の無い%d個の番号", numbers, len(names))
	}

	topicRepo := repository.NewTopicRepository(query.New(f.db))
	topics, err := topicRepo.ListActiveBySpace(ctx, f.spaceID)
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
	if len(topics) != len(names) {
		t.Errorf("len(topics) = %d、期待値 = %d", len(topics), len(names))
	}
}
