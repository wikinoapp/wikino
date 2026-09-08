package topic_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	topichandler "github.com/wikinoapp/wikino/go/internal/handler/topic"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestCreate_公開設定が不正なら422で未選択のフォームを再描画する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		form       map[string]string
	}{
		{
			name:       "未選択",
			identifier: "topic-create-no-visibility",
			form:       map[string]string{"name": "日報"},
		},
		{
			name:       "未知の値",
			identifier: "topic-create-bad-visibility",
			form: map[string]string{
				"name":       "日報",
				"visibility": "unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.GetTestDB()
			userID, spaceID := topicSpaceDB(t, db, tt.identifier, nil)

			req := newTopicFormRequest(t, http.MethodPost, "/s/"+tt.identifier+"/topics", tt.identifier, userID, tt.form)
			rr := httptest.NewRecorder()
			setupCreateHandler(t, db).Create(rr, req)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status code = %d, want %d", rr.Code, http.StatusUnprocessableEntity)
			}

			body := rr.Body.String()
			for _, want := range []string{
				"公開設定を選択してください",
				`id="visibility-error"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("response does not contain %q", want)
				}
			}
			if tag := topicFormFieldsetTag(t, body); strings.Contains(tag, "aria-describedby") {
				t.Errorf("fieldset tag contains aria-describedby: %s", tag)
			}
			for _, id := range []string{"visibility_public", "visibility_private"} {
				tag := topicFormInputTag(t, body, id)
				for _, want := range []string{`aria-invalid="true"`, `aria-describedby="visibility-error"`} {
					if !strings.Contains(tag, want) {
						t.Errorf("input %q tag does not contain %q: %s", id, want, tag)
					}
				}
				if strings.Contains(tag, "checked") {
					t.Errorf("input %q tag contains checked: %s", id, tag)
				}
			}

			topicRepo := repository.NewTopicRepository(query.New(db))
			topics, err := topicRepo.ListActiveBySpace(context.Background(), spaceID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(topics) != 0 {
				t.Errorf("len(topics) = %d, want 0", len(topics))
			}
		})
	}
}

// setupCreateHandler builds a handler against the shared pool. The create usecase opens its own
// transaction, so the fixtures these tests set up are committed rather than held in one.
//
// [Ja] setupCreateHandler は共有プールを使うハンドラーを組み立てる。作成のユースケースは自身で
// トランザクションを開くため、これらのテストが用意するフィクスチャはトランザクションに閉じ込めず
// コミットする。
func setupCreateHandler(t *testing.T, db *sql.DB) *topichandler.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost"}
	queries := query.New(db)
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)

	return topichandler.NewHandler(
		cfg,
		session.NewFlashManager("", false, true),
		usecase.NewGetTopicDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, pageRepo),
		usecase.NewGetTopicNewUsecase(spaceRepo, spaceMemberRepo),
		usecase.NewCreateTopicUsecase(
			db,
			spaceRepo,
			spaceMemberRepo,
			topicRepo,
			topicMemberRepo,
			validator.NewTopicCreateValidator(topicRepo),
		),
	)
}

// topicSpaceDB seeds a committed space with one member and returns what the tests address them by.
//
// [Ja] topicSpaceDB はメンバーが 1 人いるスペースをコミットして用意し、テストがそれらを指すための
// 値を返す。
func topicSpaceDB(t *testing.T, db *sql.DB, identifier string, scopes []model.Scope) (model.UserID, model.SpaceID) {
	t.Helper()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(identifier).
		Build()
	memberBuilder := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if scopes != nil {
		memberBuilder = memberBuilder.WithScopes(scopes)
	}
	memberBuilder.Build()

	return userID, spaceID
}

func TestCreate_トピックが作成され詳細画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	identifier := "topic-create-ok"
	userID, spaceID := topicSpaceDB(t, db, identifier, nil)

	req := newTopicFormRequest(t, http.MethodPost, "/s/"+identifier+"/topics", identifier, userID, map[string]string{
		"name":        "日報",
		"description": "毎日の記録",
		"visibility":  "public",
	})
	rr := httptest.NewRecorder()
	setupCreateHandler(t, db).Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if location := rr.Header().Get("Location"); location != "/s/"+identifier+"/topics/1" {
		t.Errorf("Location = %q, want %q", location, "/s/"+identifier+"/topics/1")
	}

	topicRepo := repository.NewTopicRepository(query.New(db))
	topics, err := topicRepo.ListActiveBySpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(topics) != 1 {
		t.Fatalf("len(topics) = %d, want 1", len(topics))
	}
	if topics[0].Name != "日報" {
		t.Errorf("Name = %q, want %q", topics[0].Name, "日報")
	}
}

func TestCreate_入力が不正なら422でフォームを再描画する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	identifier := "topic-create-invalid"
	userID, spaceID := topicSpaceDB(t, db, identifier, nil)

	req := newTopicFormRequest(t, http.MethodPost, "/s/"+identifier+"/topics", identifier, userID, map[string]string{
		"name":        "foo/bar",
		"description": "説明",
		"visibility":  "private",
	})
	rr := httptest.NewRecorder()
	setupCreateHandler(t, db).Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusUnprocessableEntity)
	}

	body := rr.Body.String()
	for _, want := range []string{
		"名前に / \\ : * ? &#34; &lt; &gt; | は使用できません",
		`aria-invalid="true"`,
		`aria-describedby="name-error"`,
		`id="name-error"`,
		`value="foo/bar"`,
		`value="説明"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}

	topicRepo := repository.NewTopicRepository(query.New(db))
	topics, err := topicRepo.ListActiveBySpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(topics) != 0 {
		t.Errorf("len(topics) = %d, want 0", len(topics))
	}
}

func TestCreate_トピック作成権限がないメンバーには404が返る(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	identifier := "topic-create-reader"
	userID, _ := topicSpaceDB(t, db, identifier, []model.Scope{model.ScopePageRead})

	req := newTopicFormRequest(t, http.MethodPost, "/s/"+identifier+"/topics", identifier, userID, map[string]string{
		"name":       "日報",
		"visibility": "public",
	})
	rr := httptest.NewRecorder()
	setupCreateHandler(t, db).Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestCreate_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	identifier := "topic-create-anon"
	topicSpaceDB(t, db, identifier, nil)

	req := newTopicFormRequest(t, http.MethodPost, "/s/"+identifier+"/topics", identifier, "", map[string]string{
		"name":       "日報",
		"visibility": "public",
	})
	rr := httptest.NewRecorder()
	setupCreateHandler(t, db).Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q, want %q", location, "/sign_in")
	}
}
