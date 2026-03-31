package page_move_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/page_move"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/sidebar"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// setupHandlerWithUsecase はユースケース付きのテスト用ハンドラーを生成するヘルパーです
func setupHandlerWithUsecase(t *testing.T, queries *query.Queries, movePageUC *usecase.MovePageUsecase) *page_move.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	pageRepo := repository.NewPageRepository(queries)

	getPageMoveDataUC := usecase.NewGetPageMoveDataUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

	return page_move.NewHandler(
		cfg,
		flashMgr,
		getPageMoveDataUC,
		movePageUC,
		sidebar.NewHelper(topicRepo, draftPageRepo),
	)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	spaceIdentifier := "pm-create-ok"

	// テストデータを作成（DB直接書き込み: usecaseが独自トランザクションを管理するため）
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("pm-create-ok@example.com").
		WithAtname("pm-create-ok").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(spaceIdentifier).
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	srcTopicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Source Topic").
		Build()
	testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Dest Topic").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(srcTopicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(srcTopicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	pageMoveValidator := validator.NewPageMoveCreateValidator(pageRepo, topicRepo, topicMemberRepo)
	movePageUC := usecase.NewMovePageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo, pageMoveValidator)
	handler := setupHandlerWithUsecase(t, q, movePageUC)

	form := url.Values{}
	form.Set("dest_topic", "2")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/s/%s/pages/1/move", spaceIdentifier), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req = addChiParams(t, req, map[string]string{
		"space_identifier": spaceIdentifier,
		"page_number":      "1",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "pm-create-ok"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// 303リダイレクトされること
	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusSeeOther)
	}

	// リダイレクト先がページパスであること
	expectedLocation := fmt.Sprintf("/s/%s/pages/1", spaceIdentifier)
	if location := rr.Header().Get("Location"); location != expectedLocation {
		t.Errorf("wrong redirect location: got %v want %v", location, expectedLocation)
	}
}

func TestCreate_ValidationError(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)

	spaceIdentifier := "pm-create-ve"

	// テストデータを作成
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("pm-create-ve@example.com").
		WithAtname("pm-create-ve").
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(spaceIdentifier).
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Topic 1").
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewPageBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	spaceRepo := repository.NewSpaceRepository(q)
	spaceMemberRepo := repository.NewSpaceMemberRepository(q)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	pageMoveValidator := validator.NewPageMoveCreateValidator(pageRepo, topicRepo, topicMemberRepo)
	movePageUC := usecase.NewMovePageUsecase(db, spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo, pageMoveValidator)
	handler := setupHandlerWithUsecase(t, q, movePageUC)

	// 移動先トピックを選択しない（空文字）
	form := url.Values{}
	form.Set("dest_topic", "")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/s/%s/pages/1/move", spaceIdentifier), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req = addChiParams(t, req, map[string]string{
		"space_identifier": spaceIdentifier,
		"page_number":      "1",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "testuser"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// 422 Unprocessable Entityが返ること
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// フォームが再表示されること（ページタイトルが含まれる）
	body := rr.Body.String()
	if !strings.Contains(body, "Test Page") {
		t.Error("page title not found in re-rendered form")
	}
}

func TestCreate_NotLoggedIn(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	form := url.Values{}
	form.Set("dest_topic", "2")

	req := httptest.NewRequest(http.MethodPost, "/s/my-space/pages/1/move", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req = addChiParams(t, req, map[string]string{
		"space_identifier": "my-space",
		"page_number":      "1",
	})

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("wrong redirect location: got %v want /sign_in", location)
	}
}
