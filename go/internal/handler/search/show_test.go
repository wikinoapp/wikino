package search_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/search"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// newSearchRequestは、キーワード・ログイン中のユーザー・画面を描画するロケールを載せたリクエストを
// 組み立てる。userIDが空のときはユーザーを載せず、未ログインの経路へ到達できるようにする。
func newSearchRequest(t *testing.T, q string, userID model.UserID) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/search?q="+url.QueryEscape(q), nil)
	ctx := context.Background()
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "search-user"})
	}
	return req.WithContext(i18n.SetLocale(ctx, i18n.LangJa))
}

func setupHandler(queries *query.Queries) *search.Handler {
	return search.NewHandler(
		&config.Config{Env: "test", Domain: "localhost"},
		usecase.NewGetSearchShowUsecase(
			repository.NewSpaceRepository(queries),
			repository.NewSpaceMemberRepository(queries),
			repository.NewPageRepository(queries),
			repository.NewTopicRepository(queries),
			repository.NewTopicMemberRepository(queries),
			validator.NewSearchShowValidator(),
		),
	)
}

func TestShow(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	h := setupHandler(queries)

	userID := testutil.NewUserBuilder(t, tx).WithEmail("search-handler@example.com").WithAtname("search_handler").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("search-handler").WithName("検索のスペース").Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("検索のトピック").Build()
	testutil.NewPageBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(7).WithTitle("検索ハンドラーのページ").Build()

	tests := []struct {
		name       string
		query      string
		wantBody   []string
		unwantBody []string
	}{
		{
			name:  "結果のページをスペースとトピックの名前とともに表示する",
			query: "検索ハンドラー",
			wantBody: []string{
				`value="検索ハンドラー"`,
				`href="/s/search-handler/pages/7"`,
				"検索ハンドラーのページ",
				"検索のスペース",
				"検索のトピック",
			},
			unwantBody: []string{"検索結果が見つかりませんでした"},
		},
		{
			name:     "結果が無ければ、見つからなかったと伝える",
			query:    "一致しない検索ハンドラー",
			wantBody: []string{"検索結果が見つかりませんでした"},
		},
		{
			name:       "キーワードが無ければ、フォームだけを表示する",
			query:      "",
			wantBody:   []string{`name="q"`},
			unwantBody: []string{"検索結果が見つかりませんでした"},
		},
		{
			name:  "space:だけなら、カーソルを指定の後ろに置く印を付けて検索しない",
			query: "space:search-handler",
			wantBody: []string{
				"data-search-query-has-space-filter",
				`value="space:search-handler"`,
			},
			unwantBody: []string{"検索結果が見つかりませんでした"},
		},
		{
			name:  "キーワードの誤りを欄の下に出す",
			query: "a",
			wantBody: []string{
				`aria-invalid="true"`,
				`aria-describedby="q-error"`,
				"検索キーワードは2文字以上で入力してください",
			},
			unwantBody: []string{"検索結果が見つかりませんでした"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.Show(rr, newSearchRequest(t, tt.query, userID))

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
				t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
			}
			body := rr.Body.String()
			for _, want := range tt.wantBody {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに %q が含まれていない", want)
				}
			}
			for _, unwant := range tt.unwantBody {
				if strings.Contains(body, unwant) {
					t.Errorf("レスポンスに %q が含まれている", unwant)
				}
			}
		})
	}
}

func TestShow_未ログインならログインへリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)

	rr := httptest.NewRecorder()
	setupHandler(testutil.QueriesWithTx(tx)).Show(rr, newSearchRequest(t, "wiki", ""))

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
	}
}
