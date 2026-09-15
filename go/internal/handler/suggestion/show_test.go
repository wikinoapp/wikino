package suggestion_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestShow_存在しないスペースで404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/nonexistent/suggestions/1", map[string]string{
		"space_identifier":  "nonexistent",
		"suggestion_number": "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_不正な提案番号で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/test-space/suggestions/abc", map[string]string{
		"space_identifier":  "test-space",
		"suggestion_number": "abc",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_存在しない提案番号で404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-noexist").
		Build()
	testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/ss-noexist/suggestions/999", map[string]string{
		"space_identifier":  "ss-noexist",
		"suggestion_number": "999",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_公開トピックの編集提案を未ログインで閲覧できる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-pub@example.com").
		WithAtname("sspub").
		WithName("提案者").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-pub-space").
		WithName("Public Space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(0).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("テスト提案詳細").
		WithBody("提案の説明").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	testutil.NewSuggestionCommentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithBody("テストコメント").
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/ss-pub-space/suggestions/1", map[string]string{
		"space_identifier":  "ss-pub-space",
		"suggestion_number": "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "テスト提案詳細") {
		t.Error("レスポンスに編集提案のタイトルが含まれていない")
	}
	if !strings.Contains(body, "@sspub") {
		t.Error("レスポンスに作成者のatnameが含まれていない")
	}
	if !strings.Contains(body, "提案の説明") {
		t.Error("レスポンスに編集提案の本文が含まれていない")
	}
	if !strings.Contains(body, "テストコメント") {
		t.Error("レスポンスにコメントが含まれていない")
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

	// /homeは認証必須のため、この公開画面が経路の起点として提示してはいけない。リンクは未ログインの
	// 訪問者をログイン画面へ送ってしまう。
	if strings.Contains(body, `href="/home"`) {
		t.Error("未ログインの公開画面のパンくずが認証必須の /homeを指している")
	}

	// 見た目の経路は、提案タイトルを非リンクの現在項目として末尾に置く。タイトルは本文とメタ情報
	// にも出るため、パンくず内に絞って検証する。
	breadcrumbStart := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if breadcrumbStart == -1 {
		t.Fatal("レスポンスにパンくずのナビゲーションが含まれていない")
	}
	breadcrumbEnd := strings.Index(body[breadcrumbStart:], "</nav>")
	if breadcrumbEnd == -1 {
		t.Fatal("パンくずのナビゲーションに閉じタグが無い")
	}
	breadcrumb := body[breadcrumbStart : breadcrumbStart+breadcrumbEnd]
	for _, want := range []string{`aria-current="page"`, "テスト提案詳細"} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/ss-pub-space/suggestions/1"`) {
		t.Error("現在の編集提案のパンくずの項目がリンクになっている")
	}

	// インデックス対象の公開画面は、保存済みの識別子から組み立てた自己参照の絶対URLを正規URL
	// として宣言する。空のhrefだとリクエストされたURLに解決されてしまう。
	for _, want := range []string{
		`<link rel="canonical" href="https://localhost/s/ss-pub-space/suggestions/1">`,
		`<meta property="og:url" content="https://localhost/s/ss-pub-space/suggestions/1">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	// インデックス対象の画面のため、見た目のパンくずと同じ項目列から作ったBreadcrumbList JSON-LD
	// で現在の編集提案までの経路を公開する。未ログインの閲覧者は公開スペースから始まるので、機械可読な
	// 複製にも /homeが出てはならない。
	for _, want := range []string{
		`<script type="application/ld+json">`,
		`"@type":"BreadcrumbList"`,
		`"position":1,"name":"Public Space","item":"https://localhost/s/ss-pub-space"`,
		`"position":2,"name":"公開トピック","item":"https://localhost/s/ss-pub-space/topics/1"`,
		`"position":3,"name":"編集提案","item":"https://localhost/s/ss-pub-space/topics/1/suggestions"`,
		`"position":4,"name":"テスト提案詳細"}`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if strings.Contains(body, `https://localhost/home`) {
		t.Error("未ログインの公開画面の構造化データが認証必須の /homeを指している")
	}
}

func TestShow_非公開トピックを未ログインで閲覧すると404が返る(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-priv@example.com").
		WithAtname("sspriv").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-priv1").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(1). // private
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("非公開提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/ss-priv1/suggestions/1", map[string]string{
		"space_identifier":  "ss-priv1",
		"suggestion_number": "1",
	}, nil)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusNotFound)
	}
}

func TestShow_非公開トピックをスペースオーナーが閲覧できる(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("ss-owner@example.com").
		WithAtname("ssowner").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("ss-priv2").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(ownerID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(1). // private
		Build()
	testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithTitle("オーナー閲覧提案").
		WithStatus(model.SuggestionStatusOpen).
		Build()

	handler := setupHandler(t, db, queries)

	req := newSuggestionRequest(t, http.MethodGet, "/s/ss-priv2/suggestions/1", map[string]string{
		"space_identifier":  "ss-priv2",
		"suggestion_number": "1",
	}, nil)
	ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: ownerID, Atname: "ssowner"})
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "オーナー閲覧提案") {
		t.Error("レスポンスに編集提案のタイトルが含まれていない")
	}
	if !strings.Contains(body, `aria-label="コメント"`) {
		t.Error("コメントのtextareaにアクセシブルな名前が無い")
	}
	// ログイン済みの閲覧者は /homeへ到達できるため、経路はそこから始まり続ける。
	if !strings.Contains(body, `href="/home"`) {
		t.Error("ログイン済みの閲覧者のパンくずに /homeが含まれていない")
	}
}

// コメントの操作メニューのトリガーはアイコンのみのため、アクセシブルネームは本ページが
// components.Postへ渡す翻訳済みのラベルが供給する。
func TestShow_コメントの操作メニューにアクセシブルネームがある(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		locale    string
		wantLabel string
	}{
		{
			name:      "日本語",
			locale:    i18n.LangJa,
			wantLabel: "コメントの操作",
		},
		{
			name:      "英語",
			locale:    i18n.LangEn,
			wantLabel: "Comment actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)

			identifier := "ss-cmt-" + tt.locale
			atname := "sscmt" + tt.locale

			userID := testutil.NewUserBuilder(t, tx).
				WithEmail(identifier + "@example.com").
				WithAtname(atname).
				Build()
			spaceID := testutil.NewSpaceBuilder(t, tx).
				WithIdentifier(identifier).
				Build()
			spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
				WithSpaceID(spaceID).
				WithUserID(userID).
				Build()
			topicID := testutil.NewTopicBuilder(t, tx).
				WithSpaceID(spaceID).
				WithNumber(1).
				WithVisibility(0).
				Build()
			suggestionID := testutil.NewSuggestionBuilder(t, tx).
				WithSpaceID(spaceID).
				WithTopicID(topicID).
				WithCreatedSpaceMemberID(spaceMemberID).
				WithTitle("コメント付き提案").
				WithStatus(model.SuggestionStatusOpen).
				Build()
			testutil.NewSuggestionCommentBuilder(t, tx).
				WithSpaceID(spaceID).
				WithSuggestionID(suggestionID).
				WithCreatedSpaceMemberID(spaceMemberID).
				WithBody("テストコメント").
				Build()

			handler := setupHandler(t, db, queries)

			req := newSuggestionRequest(t, http.MethodGet, "/s/"+identifier+"/suggestions/1", map[string]string{
				"space_identifier":  identifier,
				"suggestion_number": "1",
			}, nil)
			ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID, Atname: atname})
			ctx = i18n.SetLocale(ctx, tt.locale)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			handler.Show(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
			}

			if !strings.Contains(rr.Body.String(), `aria-label="`+tt.wantLabel+`"`) {
				t.Errorf("コメントの操作メニューのトリガーにaria-label %qが含まれていない", tt.wantLabel)
			}
		})
	}
}
