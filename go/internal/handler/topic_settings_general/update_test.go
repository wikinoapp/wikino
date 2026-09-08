package topic_settings_general_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUpdate_保存され一般設定へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-general-update"
	userID, spaceID, topicID := settingsGeneralSpace(t, tx, identifier, nil)

	req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, "1", userID, map[string]string{
		"name":        "週報",
		"description": "毎週の記録",
		"visibility":  "public",
	})
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	want := settingsGeneralPath(identifier, "1")
	if location := rr.Header().Get("Location"); location != want {
		t.Errorf("Location = %q, want %q", location, want)
	}

	topicRepo := repository.NewTopicRepository(queries)
	stored, err := topicRepo.FindBySpaceAndID(context.Background(), spaceID, topicID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Name != "週報" {
		t.Errorf("Name = %q, want %q", stored.Name, "週報")
	}
	if stored.Description != "毎週の記録" {
		t.Errorf("Description = %q, want %q", stored.Description, "毎週の記録")
	}
	if stored.Visibility != model.TopicVisibilityPublic {
		t.Errorf("Visibility = %v, want %v", stored.Visibility, model.TopicVisibilityPublic)
	}
}

func TestUpdate_入力が不正なら422でフォームを再描画する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-general-invalid"
	userID, spaceID, topicID := settingsGeneralSpace(t, tx, identifier, nil)

	req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, "1", userID, map[string]string{
		"name":        "foo/bar",
		"description": "説明",
		"visibility":  "public",
	})
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Update(rr, req)

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

	topicRepo := repository.NewTopicRepository(queries)
	stored, err := topicRepo.FindBySpaceAndID(context.Background(), spaceID, topicID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Name != "日報" {
		t.Errorf("Name = %q, want %q", stored.Name, "日報")
	}
}

func TestUpdate_トピック更新権限がないメンバーには404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-general-update-reader"
	userID, spaceID, topicID := settingsGeneralSpace(t, tx, identifier, []model.Scope{model.ScopePageRead})

	req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, "1", userID, map[string]string{
		"name":       "週報",
		"visibility": "public",
	})
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Update(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}

	topicRepo := repository.NewTopicRepository(queries)
	stored, err := topicRepo.FindBySpaceAndID(context.Background(), spaceID, topicID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Name != "日報" {
		t.Errorf("Name = %q, want %q", stored.Name, "日報")
	}
}

func TestUpdate_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-general-update-anon"
	settingsGeneralSpace(t, tx, identifier, nil)

	req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, "1", "", map[string]string{
		"name":       "週報",
		"visibility": "public",
	})
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q, want %q", location, "/sign_in")
	}
}
