package space_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	spacehandler "github.com/wikinoapp/wikino/go/internal/handler/space"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newShowRequestはchiのURLパラメータ付きGETリクエストを作成するヘルパーです
func newShowRequest(t *testing.T, path string, params map[string]string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)

	rctx := chi.NewRouteContext()
	for key, val := range params {
		rctx.URLParams.Add(key, val)
	}

	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// setupHandlerはテスト用のスペース詳細ハンドラーを作成するヘルパーです
func setupHandler(t *testing.T, queries *query.Queries) *spacehandler.Handler {
	t.Helper()

	cfg := &config.Config{
		Env:    "test",
		Domain: "localhost",
	}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)

	getSpaceShowUC := usecase.NewGetSpaceShowUsecase(spaceRepo, spaceMemberRepo, pageRepo, topicRepo, topicMemberRepo)

	return spacehandler.NewHandler(cfg, getSpaceShowUC)
}

func TestShow_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/nonexistent", map[string]string{
		"space_identifier": "nonexistent",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_メンバーがピン留めと通常ページを閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-pages-owner@example.com").
		WithAtname("sspagesowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-pages").
		WithName("Pages Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("テストトピックラベル").
		WithVisibility(0).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("通常ページ").
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

	req := newShowRequest(t, "/s/ss-pages", map[string]string{
		"space_identifier": "ss-pages",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "sspagesowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Pages Space") {
		t.Error("レスポンスにスペース名が含まれていない")
	}
	if !strings.Contains(body, "通常ページ") {
		t.Error("レスポンスに通常ページのタイトルが含まれていない")
	}
	if !strings.Contains(body, "ピン留めページ") {
		t.Error("レスポンスにピン留めしたページのタイトルが含まれていない")
	}
	// カードはトピックラベルを表示する (スペース詳細はページが複数トピックに跨る)。
	if !strings.Contains(body, "テストトピックラベル") {
		t.Error("レスポンスのカードにトピックのラベルが含まれていない")
	}
	// space:adminメンバーは編集できるため、カードごとの編集リンクが描画される。
	if !strings.Contains(body, "/s/ss-pages/pages/1/edit") {
		t.Error("レスポンスに通常ページの編集リンクが含まれていない")
	}
	if !strings.Contains(body, "/s/ss-pages/pages/2/edit") {
		t.Error("レスポンスにピン留めしたページの編集リンクが含まれていない")
	}

	// パンくずヘッダーはレイアウトが描画するため、<main> の外に出る (#mainへのスキップ
	// リンクが飛ばせる必要があるため)。この画面の本文幅max-w-3xlも維持する。
	if !strings.Contains(body, `<div class="max-w-3xl mx-auto flex w-full items-center justify-between gap-2 px-4">`) {
		t.Error("共通のパンくずヘッダーがmax-w-3xlのコンテンツ幅を保っていない")
	}
	header, main := strings.Index(body, "<header"), strings.Index(body, `<main id="main" tabindex="-1">`)
	if header == -1 || main == -1 || header > main {
		t.Errorf("共通のパンくずヘッダー (位置%d) が <main> (位置%d) より前にない", header, main)
	}
}

func TestShow_ゲストは公開トピックのページのみ閲覧できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-guest").
		WithName("Guest Space").
		Build()
	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0). // public
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("非公開トピック").
		WithVisibility(1). // private
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("公開ページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("非公開ページ").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-guest", map[string]string{
		"space_identifier": "ss-guest",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "公開ページ") {
		t.Error("レスポンスに公開トピックのページタイトルが含まれていない")
	}
	if strings.Contains(body, "非公開ページ") {
		t.Error("ゲストのレスポンスに非公開トピックのページタイトルが含まれている")
	}
	// トピックラベルはゲストにも表示される。
	if !strings.Contains(body, "公開トピック") {
		t.Error("レスポンスに公開トピックのカードのトピックラベルが含まれていない")
	}
	// ゲストは編集できないため、カードの編集リンクは描画されない。
	if strings.Contains(body, "/s/ss-guest/pages/1/edit") {
		t.Error("ゲストのレスポンスに編集リンクが含まれている")
	}
	// トピックセクションはゲストには公開トピックのみを並べ、非公開トピックは現れてはならない。
	if !strings.Contains(body, "/s/ss-guest/topics/1\"") {
		t.Error("レスポンスのトピックセクションに公開トピックの詳細リンクが含まれていない")
	}
	if strings.Contains(body, "/s/ss-guest/topics/2") {
		t.Error("ゲストのレスポンスに非公開トピックの詳細リンクが含まれている")
	}
	// ゲストはページを作成できないため、トピックごとの新規ページ作成アクションは描画されない。
	if strings.Contains(body, "/s/ss-guest/topics/1/pages/new") {
		t.Error("ゲストのレスポンスに新規ページのリンクが含まれている")
	}
}

func TestShow_参加トピックが無いメンバーにトピック作成導線が出る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-notopic-owner@example.com").
		WithAtname("ssnotopicowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-notopic").
		Build()
	// デフォルトのspace:adminスコープでトピック作成が許可される。
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-notopic", map[string]string{
		"space_identifier": "ss-notopic",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "ssnotopicowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "トピックはありません") {
		t.Error("レスポンスにトピックが無いときの空状態のメッセージが含まれていない")
	}
	if !strings.Contains(body, "新規トピック") {
		t.Error("レスポンスに新規トピックのボタンが含まれていない")
	}
	if !strings.Contains(body, "/s/ss-notopic/topics/new") {
		t.Error("レスポンスに新規トピックのフォームへのリンクが含まれていない")
	}
}

func TestShow_メンバーにトピックセクションと作成導線が表示され空状態のボタンは消える(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-nopage-owner@example.com").
		WithAtname("ssnopageowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-nopage").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(5).
		WithName("参加トピック").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-nopage", map[string]string{
		"space_identifier": "ss-nopage",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "ssnopageowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "ページはありません") {
		t.Error("レスポンスにページが無いときの空状態のメッセージが含まれていない")
	}
	// 参加トピックがトピックセクションに表示され、詳細ページへリンクする。
	if !strings.Contains(body, "参加トピック") {
		t.Error("レスポンスのトピックセクションに参加中のトピック名が含まれていない")
	}
	if !strings.Contains(body, "/s/ss-nopage/topics/5\"") {
		t.Error("レスポンスのトピックセクションにトピック詳細のリンクが含まれていない")
	}
	// メンバーは書き込めるため、トピックごとの新規ページ作成アクションがセクションに表示される。
	if !strings.Contains(body, "/s/ss-nopage/topics/5/pages/new") {
		t.Error("レスポンスのトピックセクションにトピックごとの新規ページリンクが含まれていない")
	}
	// スペースレベルの空状態「新規ページ」ボタンは消え、その説明文が表示されてはならない。
	if strings.Contains(body, "最初の1ページ目を作成しましょう") {
		t.Error("レスポンスに削除済みの空状態の新規ページ案内が含まれている")
	}
}

func TestShow_メンバーにスペースオプションメニューが全て表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-options-owner@example.com").
		WithAtname("ssoptionsowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-options").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-options", map[string]string{
		"space_identifier": "ss-options",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "ssoptionsowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	// オプションメニューにはRSSフィードと、メンバー限定のトピック / ゴミ箱 / 設定リンクが表示される。
	for _, want := range []string{
		"/s/ss-options/atom",
		"/s/ss-options/topics/new",
		"/s/ss-options/trash",
		"/s/ss-options/settings",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスにオプションメニューのリンク%qが含まれていない", want)
		}
	}
}

func TestShow_ゲストにはオプションメニューのRSSのみ表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-guest-options").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-guest-options", map[string]string{
		"space_identifier": "ss-guest-options",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	// ゲストにはRSSフィードリンクのみが見え、メンバー限定リンクは表示されない。
	if !strings.Contains(body, "/s/ss-guest-options/atom") {
		t.Error("ゲストのレスポンスにRSSフィードのリンクが含まれていない")
	}
	if strings.Contains(body, "/s/ss-guest-options/trash") {
		t.Error("ゲストのレスポンスにゴミ箱のリンクが含まれている")
	}
	if strings.Contains(body, "/s/ss-guest-options/settings") {
		t.Error("ゲストのレスポンスに設定のリンクが含まれている")
	}
}

func TestShow_ページネーションの次ページリンクが表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-paginate-owner@example.com").
		WithAtname("sspaginateowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-paginate").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	// 1ページ100件の上限を超える101件の通常ページを作成し、2ページ目を発生させる。
	for i := int32(1); i <= 101; i++ {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(i)).
			WithTitle(fmt.Sprintf("ページ%d", i)).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
	}

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-paginate", map[string]string{
		"space_identifier": "ss-paginate",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "sspaginateowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "/s/ss-paginate?page=2") {
		t.Error("レスポンスに次のページへのリンクが含まれていない")
	}
}

// スペースオプションのドロップダウントリガーはアイコンのみのため、アクセシブルネームは
// 内容ではなく翻訳済みのaria-labelが供給する。
func TestShow_スペースオプションのトリガーにアクセシブルネームがある(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		wantLabel string
	}{
		{
			name:      "日本語",
			locale:    i18n.LangJa,
			wantLabel: "スペースのオプション",
		},
		{
			name:      "英語",
			locale:    i18n.LangEn,
			wantLabel: "Space options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			identifier := "ss-optlabel-" + tt.locale
			atname := "ssoptlabel" + tt.locale

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

			handler := setupHandler(t, queries)

			req := newShowRequest(t, "/s/"+identifier, map[string]string{
				"space_identifier": identifier,
			})
			ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: atname})
			ctx = i18n.SetLocale(ctx, tt.locale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			if !strings.Contains(rr.Body.String(), `aria-label="`+tt.wantLabel+`"`) {
				t.Errorf("スペースオプションのトリガーにaria-label %qが含まれていない", tt.wantLabel)
			}
		})
	}
}

// 正規URLが指すべきは保存済みの識別子である。spaces.identifierはcitextのため大文字小文字が
// 違うリクエストでも同じ画面に到達し、そのままでは同じ内容に対して2つ目の正規アドレスを宣言して
// しまう。
func TestShow_CanonicalUsesStoredIdentifier(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-canonical").
		WithName("Canonical Space").
		Build()

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/SS-CANONICAL", map[string]string{
		"space_identifier": "SS-CANONICAL",
	})

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>Canonical Space</title>",
		"<meta property=\"og:title\" content=\"Canonical Space\">",
		`<link rel="canonical" href="https://localhost/s/ss-canonical">`,
		`<meta property="og:url" content="https://localhost/s/ss-canonical">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if strings.Contains(body, "SS-CANONICAL") {
		t.Error("リクエストした表記がレスポンスに残っている")
	}
}

// 系列の各ページは載っているページが異なるため、1ページ目ではなく自分自身を正規アドレスとして
// 宣言する。
func TestShow_PaginatedCanonicalPreservesPageParameter(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-canonical-page-owner@example.com").
		WithAtname("sscanonicalpageowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-canonical-page").
		WithName("Paginated Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	// 1ページ100件の上限を超える101件の通常ページを作成し、2ページ目を発生させる。
	for i := int32(1); i <= 101; i++ {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(i)).
			WithTitle(fmt.Sprintf("ページ%d", i)).
			WithLinkedPageIDs([]model.PageID{}).
			Build()
	}

	handler := setupHandler(t, queries)

	req := newShowRequest(t, "/s/ss-canonical-page?page=2", map[string]string{
		"space_identifier": "ss-canonical-page",
	})
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "sscanonicalpageowner"})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	for _, want := range []string{
		"<title>Paginated Space (2 ページ目)</title>",
		"<meta property=\"og:title\" content=\"Paginated Space (2 ページ目)\">",
	} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	if want := `<link rel="canonical" href="https://localhost/s/ss-canonical-page?page=2">`; !strings.Contains(rr.Body.String(), want) {
		t.Errorf("レスポンスに%qが含まれていない", want)
	}
}

func TestShow_PageBeyondTotalReturnsNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-page-out-of-range").
		WithName("Out of Range Space").
		Build()

	handler := setupHandler(t, queries)

	// 最後の値はint32を超える。offsetに収まらない正のページ番号が指すのは、範囲内の範囲外値と
	// 同じく存在しないページのため、1ページ目へフォールバックせず同じ404になる。
	for _, page := range []string{"2", "2147483647", "99999999999"} {
		t.Run("page="+page, func(t *testing.T) {
			req := newShowRequest(t, "/s/ss-page-out-of-range?page="+page, map[string]string{
				"space_identifier": "ss-page-out-of-range",
			})

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
			}

			body := rr.Body.String()
			if strings.Contains(body, "Out of Range Space") {
				t.Error("レスポンスにスペース名が含まれている")
			}
			if strings.Contains(body, "/s/ss-page-out-of-range?page="+page) {
				t.Error("レスポンスに自身を指すcanonical URLが含まれている")
			}
		})
	}
}

// スペースはパンくずの末尾項目のため、ログイン済みの閲覧者には認証後アプリの親である /homeの下で
// 現在ページとして伝える。未ログインの閲覧者にはRails版と同じく /home項目を出さないため、残るのは
// スペース自身だけになる。その経路にはたどれる項目が無いので、リンクの無いナビゲーションランドマークを
// 描画せずパンくずごと落とす。未ログイン時はBreadcrumbList構造化データを出さない。ログイン時は
// 見た目のパンくずと同じホーム › スペースの経路を構造化データとして出す。
func TestShow_BreadcrumbMarksCurrentSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	viewerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-breadcrumb-viewer@example.com").
		WithAtname("ssbreadcrumbviewer").
		Build()
	testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-breadcrumb").
		WithName("Breadcrumb Space").
		Build()

	handler := setupHandler(t, queries)

	tests := []struct {
		name           string
		user           *model.User
		wantBreadcrumb bool
	}{
		{name: "未ログインの閲覧者にはパンくずが出ない", wantBreadcrumb: false},
		{name: "ログイン済みの閲覧者にはホームの下に現在のスペースが出る", user: &model.User{ID: viewerID, Atname: "ssbreadcrumbviewer"}, wantBreadcrumb: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newShowRequest(t, "/s/ss-breadcrumb", map[string]string{
				"space_identifier": "ss-breadcrumb",
			})
			if tt.user != nil {
				req = req.WithContext(middleware.SetUserToContext(req.Context(), tt.user))
			}

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()

			// パンくずのランドマークは専用のラベルで特定する。グローバルナビバーもaria-label付きの
			// <nav> であるため。
			start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
			if !tt.wantBreadcrumb {
				if start != -1 {
					t.Error("現在の項目だけのパンくずが描画されている")
				}
				if strings.Contains(body, `aria-current="page"`) {
					t.Error("パンくずが無いのに現在の項目がある")
				}

				// 構造化データは見た目のパンくずを写したものなので、パンくずを描画しない経路では
				// 出さない。クローラーが見るのはこの状態である。
				if strings.Contains(body, "application/ld+json") {
					t.Error("現在の項目だけのパンくずが構造化データを出している")
				}
				return
			}

			if start == -1 {
				t.Fatal("レスポンスにパンくずのナビゲーションが含まれていない")
			}
			end := strings.Index(body[start:], "</nav>")
			if end == -1 {
				t.Fatal("パンくずのナビゲーションに閉じタグが無い")
			}
			breadcrumb := body[start : start+end]
			if !strings.Contains(breadcrumb, `href="/home"`) {
				t.Error("ログイン済みのパンくずが /homeへのリンクになっていない")
			}
			if !strings.Contains(breadcrumb, `aria-current="page"`) {
				t.Error("現在のパンくずの項目にaria-currentが付いていない")
			}

			// ここでは見た目の経路にたどれる項目があるため、同じ項目列から機械可読な複製を出す。
			// ホームはリンク、スペースは非リンクの現在項目になる。
			for _, want := range []string{
				`"@type":"BreadcrumbList"`,
				`"position":1,"name":"ホーム","item":"https://localhost/home"`,
				`"position":2,"name":"Breadcrumb Space"}`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
		})
	}
}
