package profile_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/profile"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newProfileRequestはchiのURLパラメータ・ログイン中のユーザー・画面を描画するロケールを載せた
// リクエストを組み立てる。viewerがnilのときはユーザーを載せず、未ログインの経路へ到達できるようにする。
func newProfileRequest(t *testing.T, atname string, viewer *model.User) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/@"+atname, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("atname", atname)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if viewer != nil {
		ctx = middleware.SetUserToContext(ctx, viewer)
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

func setupHandler(t *testing.T, tx *sql.Tx) *profile.Handler {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)
	return profile.NewHandler(
		&config.Config{Env: "test", Domain: "localhost"},
		usecase.NewGetProfileShowUsecase(
			repository.NewUserRepository(queries),
			repository.NewSpaceRepository(queries),
		),
	)
}

// profileUserは名前と自己紹介を持つユーザーを作り、そのユーザーのモデルを返す
func profileUser(t *testing.T, tx *sql.Tx, atname, name string) *model.User {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(atname + "@example.com").
		WithAtname(atname).
		WithName(name).
		Build()
	if _, err := tx.ExecContext(context.Background(), "UPDATE users SET description = $1 WHERE id = $2", "自己紹介の文章", string(userID)); err != nil {
		t.Fatalf("自己紹介の設定に失敗: %v", err)
	}

	return &model.User{ID: userID, Atname: atname}
}

// joinSpaceはスペースを作り、ユーザーを渡した状態のメンバーにする
func joinSpace(t *testing.T, tx *sql.Tx, userID model.UserID, identifier, name string, discarded, active bool) {
	t.Helper()

	builder := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).WithName(name)
	if discarded {
		builder = builder.WithDiscarded()
	}
	spaceID := builder.Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithActive(active).
		Build()
}

func TestShow_ログインしていなくてもプロフィールを表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := profileUser(t, tx, "profile_guest", "プロフィールの人")
	joinSpace(t, tx, user.ID, "profile-guest-active", "参加中のスペース", false, true)
	joinSpace(t, tx, user.ID, "profile-guest-discarded", "削除したスペース", true, true)
	joinSpace(t, tx, user.ID, "profile-guest-inactive", "抜けたスペース", false, false)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).Show(rr, newProfileRequest(t, "profile_guest", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	// プロフィールは公開のページで、インデックスの対象のままにする
	if got := rr.Header().Get("X-Robots-Tag"); got != "" {
		t.Errorf("X-Robots-Tag = %q、空を期待", got)
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>プロフィールの人 (@profile_guest) | Wikino</title>",
		`<meta name="description" content="自己紹介の文章">`,
		`<link rel="canonical" href="https://localhost/@profile_guest">`,
		"プロフィールの人",
		"@profile_guest",
		"自己紹介の文章",
		"参加中のスペース",
		`href="/s/profile-guest-active"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	for _, unwanted := range []string{
		`href="/s/profile-guest-discarded"`,
		`href="/s/profile-guest-inactive"`,
		`href="/settings/profile"`,
		`href="/settings"`,
	} {
		if strings.Contains(body, unwanted) {
			t.Errorf("レスポンスに %q が含まれている", unwanted)
		}
	}
}

func TestShow_本人にだけ編集と設定へのリンクを出す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		atname    string
		ownViewer bool
		wantLinks bool
	}{
		{name: "本人", atname: "profile_owner", ownViewer: true, wantLinks: true},
		{name: "ほかのユーザー", atname: "profile_others", ownViewer: false, wantLinks: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			owner := profileUser(t, tx, tt.atname, "持ち主")
			viewer := owner
			if !tt.ownViewer {
				viewer = profileUser(t, tx, tt.atname+"_viewer", "閲覧者")
			}

			rr := httptest.NewRecorder()
			setupHandler(t, tx).Show(rr, newProfileRequest(t, tt.atname, viewer))

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			body := rr.Body.String()
			for _, link := range []string{`href="/settings/profile"`, `href="/settings"`} {
				if got := strings.Contains(body, link); got != tt.wantLinks {
					t.Errorf("%q を含むか = %v、期待値 = %v", link, got, tt.wantLinks)
				}
			}
		})
	}
}

func TestShow_名前が空ならアットネームを見出しにする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	profileUser(t, tx, "profile_no_name", "")

	rr := httptest.NewRecorder()
	setupHandler(t, tx).Show(rr, newProfileRequest(t, "profile_no_name", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<title>@profile_no_name | Wikino</title>") {
		t.Error("タイトルがアットネームになっていない")
	}
}

func TestShow_参加中のスペースが無ければ空の状態を表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	profileUser(t, tx, "profile_no_spaces", "スペースの無い人")

	rr := httptest.NewRecorder()
	setupHandler(t, tx).Show(rr, newProfileRequest(t, "profile_no_spaces", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "参加中のスペースは") {
		t.Error("空の状態のメッセージが表示されていない")
	}
}

func TestShow_ユーザーが見つからなければ404を返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		atname    string
		discarded bool
	}{
		{name: "削除済みのユーザー", atname: "profile_discarded", discarded: true},
		{name: "存在しないアットネーム", atname: "profile_missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			if tt.discarded {
				user := profileUser(t, tx, tt.atname, "削除した人")
				if _, err := tx.ExecContext(context.Background(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(user.ID)); err != nil {
					t.Fatalf("ユーザーの削除に失敗: %v", err)
				}
			}

			rr := httptest.NewRecorder()
			setupHandler(t, tx).Show(rr, newProfileRequest(t, tt.atname, nil))

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}
