package space_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreate_スペースを作成してスペースの画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("space-create-success@example.com").
		WithAtname("space_create_success").
		Build()

	req := newSpaceFormRequest(t, http.MethodPost, "/spaces", userID, map[string]string{
		"identifier": "space-create-success",
		"name":       "作成したスペース",
	})
	rr := httptest.NewRecorder()
	setupCreateHandler(t, db).Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "/s/space-create-success" {
		t.Errorf("Location = %q、期待値 = %q", got, "/s/space-create-success")
	}
	hasFlash := slices.ContainsFunc(rr.Result().Cookies(), func(c *http.Cookie) bool {
		return c.Name == session.FlashCookieName && c.Value != ""
	})
	if !hasFlash {
		t.Error("フラッシュメッセージのCookieが設定されていない")
	}

	space := findSpace(t, db, "space-create-success")
	if space == nil {
		t.Fatal("スペースが作成されていない")
	}
	if space.Name != "作成したスペース" {
		t.Errorf("space.Name = %q、期待値 = %q", space.Name, "作成したスペース")
	}

	member, err := repository.NewSpaceMemberRepository(query.New(db)).FindActiveBySpaceAndUser(context.Background(), space.ID, userID)
	if err != nil {
		t.Fatalf("スペースメンバーの取得に失敗: %v", err)
	}
	if member == nil {
		t.Fatal("作成者がスペースのメンバーになっていない")
	}
	if member.Role != model.SpaceRoleAdmin {
		t.Errorf("member.Role = %q、期待値 = %q", member.Role, model.SpaceRoleAdmin)
	}
}

func TestCreate_入力が不正なら422で入力を戻したフォームを再描画する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("space-create-invalid@example.com").
		WithAtname("space_create_invalid").
		Build()
	testutil.NewSpaceBuilderDB(t, db).WithIdentifier("space-create-taken").Build()

	tests := []struct {
		name       string
		identifier string
		spaceName  string
		wantMsgs   []string
	}{
		{
			name:       "識別子の形式が不正",
			identifier: "bad_identifier",
			spaceName:  "不正な識別子",
			wantMsgs:   []string{"識別子には半角英数字とハイフンのみ使用できます"},
		},
		{
			name:       "識別子が使われている",
			identifier: "space-create-taken",
			spaceName:  "重複した識別子",
			wantMsgs:   []string{"この識別子は既に使用されています"},
		},
		{
			name:       "識別子と名前が空",
			identifier: "",
			spaceName:  "",
			wantMsgs:   []string{"識別子を入力してください", "名前を入力してください"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newSpaceFormRequest(t, http.MethodPost, "/spaces", userID, map[string]string{
				"identifier": tt.identifier,
				"name":       tt.spaceName,
			})
			rr := httptest.NewRecorder()
			setupCreateHandler(t, db).Create(rr, req)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}
			if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
				t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
			}

			body := rr.Body.String()
			for _, want := range tt.wantMsgs {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
			for _, want := range []string{
				`value="` + tt.identifier + `"`,
				`value="` + tt.spaceName + `"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに送信した値%qが含まれていない", want)
				}
			}
		})
	}

	// 拒否した送信ではスペースを作らない
	spaces, err := repository.NewSpaceRepository(query.New(db)).ListActiveByUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if len(spaces) != 0 {
		t.Errorf("len(spaces) = %d、期待値 = 0", len(spaces))
	}
}

func TestCreate_未ログインならサインイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	req := newSpaceFormRequest(t, http.MethodPost, "/spaces", "", map[string]string{
		"identifier": "space-create-anonymous",
		"name":       "未ログイン",
	})
	rr := httptest.NewRecorder()
	setupCreateHandler(t, db).Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
	}
	if space := findSpace(t, db, "space-create-anonymous"); space != nil {
		t.Error("未ログインの送信でスペースが作成された")
	}
}
