package topic_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	topichandler "github.com/wikinoapp/wikino/go/internal/handler/topic"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newShowRequest はchiのURLパラメータ付きGETリクエストを作成するヘルパーです
func newShowRequest(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// setupHandler はテスト用のトピックハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *topichandler.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	flashMgr := session.NewFlashManager("", false, true)
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)

	getTopicDetailUC := usecase.NewGetTopicDetailUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, pageRepo)

	return topichandler.NewHandler(
		cfg,
		flashMgr,
		getTopicDetailUC,
	)
}

func TestShow_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/nonexistent/topics/1", map[string]string{
		"space_identifier": "nonexistent",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_不正なトピック番号で404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/test-space/topics/abc", map[string]string{
		"space_identifier": "test-space",
		"topic_number":     "abc",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_存在しないトピックで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-noexist").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-noexist/topics/999", map[string]string{
		"space_identifier": "ts-noexist",
		"topic_number":     "999",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_公開トピックを未ログインで閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-public").
		WithName("Public Space").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-public/topics/1", map[string]string{
		"space_identifier": "ts-public",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "公開トピック") {
		t.Error("response should contain topic name")
	}
}

func TestShow_非公開トピックを未ログインで閲覧すると404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-priv1").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("非公開トピック").
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-priv1/topics/1", map[string]string{
		"space_identifier": "ts-priv1",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_非公開トピックをスペースオーナーが閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-owner@example.com").
		WithAtname("tsowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-priv2").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("オーナー閲覧可能").
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-priv2/topics/1", map[string]string{
		"space_identifier": "ts-priv2",
		"topic_number":     "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "tsowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "オーナー閲覧可能") {
		t.Error("response should contain topic name")
	}
}

func TestShow_非公開トピックをトピックメンバーが閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	memberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-member@example.com").
		WithAtname("tsmember").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-priv3").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberUserID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("メンバー閲覧可能").
		WithVisibility(1). // private
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-priv3/topics/1", map[string]string{
		"space_identifier": "ts-priv3",
		"topic_number":     "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: memberUserID, Atname: "tsmember"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "メンバー閲覧可能") {
		t.Error("response should contain topic name")
	}
}

func TestShow_非公開トピックをスペースメンバーが閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-owner4@example.com").
		WithAtname("tsowner4").
		Build()
	memberID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-nonmember@example.com").
		WithAtname("tsnonmember").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-priv4").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberID).
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(1). // private
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-priv4/topics/1", map[string]string{
		"space_identifier": "ts-priv4",
		"topic_number":     "1",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: memberID, Atname: "tsnonmember"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestShow_正常系_ページ一覧が表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-pages").
		WithName("Pages Space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピック").
		WithVisibility(0).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("最初のページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("ピン留めページ").
		WithLinkedPageIDs([]model.PageID{}).
		WithPinnedAt(time.Now()).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-pages/topics/1", map[string]string{
		"space_identifier": "ts-pages",
		"topic_number":     "1",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "テストトピック") {
		t.Error("response should contain topic name")
	}
	if !strings.Contains(body, "最初のページ") {
		t.Error("response should contain regular page title")
	}
	if !strings.Contains(body, "ピン留めページ") {
		t.Error("response should contain pinned page title")
	}

	// The breadcrumb header comes from the layout, so it renders outside <main> (the #main skip
	// link has to bypass it) and keeps this screen's max-w-3xl content width. Its trailing crumb
	// links back to the space.
	//
	// [Ja] パンくずヘッダーはレイアウトが描画するため、<main> の外に出る (#main へのスキップ
	// リンクが飛ばせる必要があるため)。この画面の本文幅 max-w-3xl も維持する。末尾のパンくずは
	// スペースへ戻るリンク。
	if !strings.Contains(body, `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("shared breadcrumb header should keep the max-w-3xl content width")
	}
	if !strings.Contains(body, `href="/s/ts-pages"`) {
		t.Error("breadcrumb should link back to the space")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("shared breadcrumb header (index %d) must precede <main> (index %d)", header, main)
	}
}

// Regression test verifying that the suggestion tab is shown in every viewable
// scenario on the topic detail screen.
//
// [Ja] トピック詳細画面で、閲覧可能なすべてのシナリオで編集提案タブが
// 表示されることを検証する回帰テスト。
func TestShow_編集提案タブが常に表示される(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		spaceIdentifier string
		visibility      int32
		withSpaceMember bool
		withTopicMember bool
	}{
		{
			name:            "公開トピック_未ログイン",
			spaceIdentifier: "ts-tab-pub-guest",
			visibility:      0,
			withSpaceMember: false,
			withTopicMember: false,
		},
		{
			name:            "非公開トピック_スペースメンバー",
			spaceIdentifier: "ts-tab-priv-sm",
			visibility:      1,
			withSpaceMember: true,
			withTopicMember: false,
		},
		{
			name:            "非公開トピック_トピックメンバー",
			spaceIdentifier: "ts-tab-priv-tm",
			visibility:      1,
			withSpaceMember: true,
			withTopicMember: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			spaceID := testutil.NewSpaceBuilder(t, tx).
				WithIdentifier(tt.spaceIdentifier).
				Build()
			topicID := testutil.NewTopicBuilder(t, tx).
				WithSpaceID(spaceID).
				WithNumber(1).
				WithName("タブテスト").
				WithVisibility(tt.visibility).
				Build()

			handler := setupHandler(t, queries)

			req := newShowRequest(t, fmt.Sprintf("/s/%s/topics/1", tt.spaceIdentifier), map[string]string{
				"space_identifier": tt.spaceIdentifier,
				"topic_number":     "1",
			})

			if tt.withSpaceMember {
				atname := strings.ReplaceAll(tt.spaceIdentifier, "-", "")
				userID := testutil.NewUserBuilder(t, tx).
					WithEmail(fmt.Sprintf("%s@example.com", tt.spaceIdentifier)).
					WithAtname(atname).
					Build()
				spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
					WithSpaceID(spaceID).
					WithUserID(userID).
					Build()
				if tt.withTopicMember {
					testutil.NewTopicMemberBuilder(t, tx).
						WithSpaceID(spaceID).
						WithTopicID(topicID).
						WithSpaceMemberID(spaceMemberID).
						Build()
				}
				ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: atname})
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			expected := fmt.Sprintf("/s/%s/topics/1/suggestions", tt.spaceIdentifier)
			if !strings.Contains(body, expected) {
				t.Errorf("response should contain suggestions tab link %q", expected)
			}
		})
	}
}

// The topic options dropdown trigger is icon-only, so its accessible name comes from the
// translated aria-label rather than from its content.
//
// [Ja] トピックオプションのドロップダウントリガーはアイコンのみのため、アクセシブルネームは
// 内容ではなく翻訳済みの aria-label が供給する。
func TestShow_トピックオプションのトリガーにアクセシブルネームがある(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		wantLabel string
	}{
		{
			name:      "日本語",
			locale:    i18n.LangJa,
			wantLabel: "トピックのオプション",
		},
		{
			name:      "英語",
			locale:    i18n.LangEn,
			wantLabel: "Topic options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			identifier := "ts-optlabel-" + tt.locale
			atname := "tsoptlabel" + tt.locale

			ownerID := testutil.NewUserBuilder(t, tx).
				WithEmail(identifier + "@example.com").
				WithAtname(atname).
				Build()
			spaceID := testutil.NewSpaceBuilder(t, tx).
				WithIdentifier(identifier).
				Build()
			testutil.NewSpaceMemberBuilder(t, tx).
				WithSpaceID(spaceID).
				WithUserID(ownerID).
				Build()
			testutil.NewTopicBuilder(t, tx).
				WithSpaceID(spaceID).
				WithNumber(1).
				WithVisibility(0).
				Build()

			handler := setupHandler(t, queries)

			req := newShowRequest(t, "/s/"+identifier+"/topics/1", map[string]string{
				"space_identifier": identifier,
				"topic_number":     "1",
			})
			ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: atname})
			ctx = i18n.SetLocale(ctx, tt.locale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			if !strings.Contains(rr.Body.String(), `aria-label="`+tt.wantLabel+`"`) {
				t.Errorf("トピックオプションのトリガーに aria-label %q が含まれていない", tt.wantLabel)
			}
		})
	}
}

// The stored identifier is what the canonical URL must point at. spaces.identifier is citext, so a
// request whose casing differs reaches the same screen and would otherwise declare a second
// canonical address for the same content. The topic number comes from the stored topic for the same
// reason, so that /topics/007 does not become an address of its own either.
//
// [Ja] 正規 URL が指すべきは保存済みの識別子である。spaces.identifier は citext のため大文字小文字が
// 違うリクエストでも同じ画面に到達し、そのままでは同じ内容に対して 2 つ目の正規アドレスを宣言して
// しまう。トピック番号を保存済みのトピックから取るのも同じ理由で、/topics/007 が独自のアドレスに
// ならないようにするためである。
func TestShow_CanonicalUsesStoredIdentifierAndTopicNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-canonical").
		WithName("Canonical Space").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/TS-CANONICAL/topics/001", map[string]string{
		"space_identifier": "TS-CANONICAL",
		"topic_number":     "001",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>公開トピック | Canonical Space</title>",
		"<meta property=\"og:title\" content=\"公開トピック | Canonical Space\">",
		`<link rel="canonical" href="https://localhost/s/ts-canonical/topics/1">`,
		`<meta property="og:url" content="https://localhost/s/ts-canonical/topics/1">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}
	if strings.Contains(body, "TS-CANONICAL") {
		t.Error("リクエストした表記がレスポンスに残っている")
	}
}

func TestShow_PaginatedTitleIncludesPageNumber(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-paginated-title").
		WithName("Paginated Space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Paginated Topic").
		WithVisibility(0).
		Build()

	for i := int32(1); i <= 101; i++ {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(i)).
			WithTitle(fmt.Sprintf("Page %d", i)).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
	}

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ts-paginated-title/topics/1?page=2", map[string]string{
		"space_identifier": "ts-paginated-title",
		"topic_number":     "1",
	})
	req = req.WithContext(i18n.SetLocale(req.Context(), i18n.LangEn))

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Each page of the series carries different pages, so it declares itself rather than the first
	// page as its canonical address.
	//
	// [Ja] 系列の各ページは載っているページが異なるため、1 ページ目ではなく自分自身を正規アドレスとして
	// 宣言する。
	for _, want := range []string{
		"<title>Paginated Topic | Paginated Space (Page 2)</title>",
		"<meta property=\"og:title\" content=\"Paginated Topic | Paginated Space (Page 2)\">",
		`<link rel="canonical" href="https://localhost/s/ts-paginated-title/topics/1?page=2">`,
		`<meta property="og:url" content="https://localhost/s/ts-paginated-title/topics/1?page=2">`,
	} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Errorf("response does not contain %q", want)
		}
	}
}

func TestShow_PageBeyondTotalReturnsNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-page-out-of-range").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Out of Range Topic").
		WithVisibility(0).
		Build()

	handler := setupHandler(t, queries)

	for _, page := range []string{"2", "2147483647"} {
		t.Run("page="+page, func(t *testing.T) {
			req := newShowRequest(t, "/s/ts-page-out-of-range/topics/1?page="+page, map[string]string{
				"space_identifier": "ts-page-out-of-range",
				"topic_number":     "1",
			})

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNotFound)
			}

			body := rr.Body.String()
			if strings.Contains(body, "Out of Range Topic") {
				t.Error("response should not contain the topic name")
			}
			if strings.Contains(body, "/s/ts-page-out-of-range/topics/1?page="+page) {
				t.Error("response should not contain a self-referencing canonical URL")
			}
		})
	}
}

// The topic is the last item of its breadcrumb, so it ends the trail as a plain crumb marked as
// the current page. The same visible items are published as BreadcrumbList structured data:
// signed-out viewers start at the public space, while signed-in viewers also get /home.
//
// [Ja] トピックはパンくずの末尾項目のため、現在ページとして印を付けたリンク無しの項目で経路を
// 締める。同じ表示項目を BreadcrumbList 構造化データにも出し、未ログインの閲覧者は公開スペースから、
// ログイン済みの閲覧者は /home から始める。
func TestShow_BreadcrumbMarksCurrentTopicWithStructuredData(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	viewerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ts-breadcrumb-viewer@example.com").
		WithAtname("tsbreadcrumbviewer").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ts-breadcrumb").
		WithName("Breadcrumb Space").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()

	handler := setupHandler(t, queries)

	tests := []struct {
		name            string
		user            *model.User
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "signed-out viewer starts at public space",
			wantContains: []string{
				`aria-current="page"`,
				"<script type=\"application/ld+json\">",
				"\"@type\":\"BreadcrumbList\"",
				"\"position\":1,\"name\":\"Breadcrumb Space\",\"item\":\"https://localhost/s/ts-breadcrumb\"",
				"\"position\":2,\"name\":\"公開トピック\"",
			},
			wantNotContains: []string{`href="/home"`, `https://localhost/home`},
		},
		{
			name: "signed-in viewer starts at authenticated home",
			user: &model.User{ID: viewerID, Atname: "tsbreadcrumbviewer"},
			wantContains: []string{
				`aria-current="page"`,
				"<script type=\"application/ld+json\">",
				"\"@type\":\"BreadcrumbList\"",
				"\"position\":1,\"name\":\"ホーム\",\"item\":\"https://localhost/home\"",
				"\"position\":2,\"name\":\"Breadcrumb Space\",\"item\":\"https://localhost/s/ts-breadcrumb\"",
				"\"position\":3,\"name\":\"公開トピック\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newShowRequest(t, "/s/ts-breadcrumb/topics/1", map[string]string{
				"space_identifier": "ts-breadcrumb",
				"topic_number":     "1",
			})
			if tt.user != nil {
				req = req.WithContext(middleware.SetUserToContext(req.Context(), tt.user))
			}

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("response does not contain %q", want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(body, notWant) {
					t.Errorf("response unexpectedly contains %q", notWant)
				}
			}
			if strings.Contains(body, "\"name\":\"公開トピック\",\"item\":") {
				t.Error("current breadcrumb structured-data item must not link to itself")
			}
		})
	}
}
