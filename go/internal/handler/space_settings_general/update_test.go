package space_settings_general_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// storedSpaceはIDでスペースを読み直す。削除済みのスペースは返らない。
func storedSpace(t *testing.T, queries *query.Queries, spaceID model.SpaceID) *model.Space {
	t.Helper()

	space, err := repository.NewSpaceRepository(queries).FindByID(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if space == nil {
		t.Fatal("スペースが見つからない")
	}
	return space
}

func TestUpdate_保存され新しい識別子の一般設定へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-general-update"
	userID, spaceID := settingsGeneralSpace(t, tx, identifier, "")

	handler := setupSettingsGeneralHandler(t, queries)
	req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, userID, map[string]string{
		"identifier": "space-gen-renamed",
		"name":       "企画チーム",
	})
	rr := httptest.NewRecorder()
	handler.Update(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	want := settingsGeneralPath("space-gen-renamed")
	if location := rr.Header().Get("Location"); location != want {
		t.Errorf("Location = %q、期待値 = %q", location, want)
	}

	stored := storedSpace(t, queries, spaceID)
	if stored.Identifier != "space-gen-renamed" {
		t.Errorf("Identifier = %q、期待値 = %q", stored.Identifier, "space-gen-renamed")
	}
	if stored.Name != "企画チーム" {
		t.Errorf("Name = %q、期待値 = %q", stored.Name, "企画チーム")
	}

	// 識別子を変えたあとは、新しいURLで開け、古いURLは見つからない
	newReq := newSettingsGeneralRequest(t, http.MethodGet, "space-gen-renamed", userID, nil)
	newRR := httptest.NewRecorder()
	handler.Show(newRR, newReq)
	if newRR.Code != http.StatusOK {
		t.Errorf("新しいURLのステータスコード = %d、期待値 = %d", newRR.Code, http.StatusOK)
	}
	oldReq := newSettingsGeneralRequest(t, http.MethodGet, identifier, userID, nil)
	oldRR := httptest.NewRecorder()
	handler.Show(oldRR, oldReq)
	if oldRR.Code != http.StatusNotFound {
		t.Errorf("古いURLのステータスコード = %d、期待値 = %d", oldRR.Code, http.StatusNotFound)
	}
}

func TestUpdate_識別子を変えずに保存できる(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		identifier    string
		newIdentifier string
	}{
		{name: "自分の識別子のまま", identifier: "space-general-same", newIdentifier: "space-general-same"},
		{name: "大文字と小文字だけを変える", identifier: "space-general-case", newIdentifier: "Space-General-Case"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID, spaceID := settingsGeneralSpace(t, tx, tt.identifier, "")

			req := newSettingsGeneralRequest(t, http.MethodPatch, tt.identifier, userID, map[string]string{
				"identifier": tt.newIdentifier,
				"name":       "名前だけ変える",
			})
			rr := httptest.NewRecorder()
			setupSettingsGeneralHandler(t, queries).Update(rr, req)

			if rr.Code != http.StatusSeeOther {
				t.Fatalf("ステータスコード = %d、期待値 = %d (本文: %s)", rr.Code, http.StatusSeeOther, rr.Body.String())
			}
			stored := storedSpace(t, queries, spaceID)
			if string(stored.Identifier) != tt.newIdentifier {
				t.Errorf("Identifier = %q、期待値 = %q", stored.Identifier, tt.newIdentifier)
			}
			if stored.Name != "名前だけ変える" {
				t.Errorf("Name = %q、期待値 = %q", stored.Name, "名前だけ変える")
			}
		})
	}
}

func TestUpdate_入力が不正なら422で保存されている識別子のフォームを再描画する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		newIdentifier string
		newName       string
		field         string
		message       string
		// takenはほかのスペースに先に使わせておく識別子。discardedTakenは削除済みにする
		taken          string
		discardedTaken bool
	}{
		{name: "識別子の形式が不正", newIdentifier: "space general", newName: "名前", field: "identifier", message: "識別子には半角英数字とハイフンのみ使用できます"},
		{name: "識別子が予約語", newIdentifier: "WWW", newName: "名前", field: "identifier", message: "この識別子は使用できません"},
		{name: "識別子が長すぎる", newIdentifier: strings.Repeat("a", model.SpaceIdentifierMaxLength+1), newName: "名前", field: "identifier", message: "識別子は20文字以内で入力してください"},
		{name: "識別子が未入力", newIdentifier: "", newName: "名前", field: "identifier", message: "識別子を入力してください"},
		{name: "名前が長すぎる", newIdentifier: "space-general-ok", newName: strings.Repeat("あ", 31), field: "name", message: "名前は30文字以内で入力してください"},
		{name: "名前が未入力", newIdentifier: "space-general-ok", newName: "", field: "name", message: "名前を入力してください"},
		{name: "ほかのスペースの識別子", newIdentifier: "Space-General-Taken", newName: "名前", field: "identifier", message: "この識別子は既に使用されています", taken: "space-general-taken"},
		{name: "削除済みのスペースの識別子", newIdentifier: "space-gen-discarded", newName: "名前", field: "identifier", message: "この識別子は既に使用されています", taken: "space-gen-discarded", discardedTaken: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			identifier := "space-general-invalid-" + string(rune('a'+i))
			userID, spaceID := settingsGeneralSpace(t, tx, identifier, "")
			if tt.taken != "" {
				builder := testutil.NewSpaceBuilder(t, tx).WithIdentifier(tt.taken)
				if tt.discardedTaken {
					builder = builder.WithDiscarded()
				}
				builder.Build()
			}

			req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, userID, map[string]string{
				"identifier": tt.newIdentifier,
				"name":       tt.newName,
			})
			rr := httptest.NewRecorder()
			setupSettingsGeneralHandler(t, queries).Update(rr, req)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}

			body := rr.Body.String()
			for _, want := range []string{
				tt.message,
				`<h2>入力内容を確認してください</h2>`,
				`href="#` + tt.field + `">` + tt.message + `</a>`,
				`aria-invalid="true"`,
				`id="` + tt.field + `-error"`,
				// 送信先とキャンセルは、入力された識別子ではなく保存されている識別子を指す
				`action="/s/` + identifier + `/settings/general"`,
				`href="/s/` + identifier + `/settings"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}

			stored := storedSpace(t, queries, spaceID)
			if string(stored.Identifier) != identifier || stored.Name != "開発チーム" {
				t.Errorf("保存されたスペース = (%q, %q)、期待値 = (%q, %q)", stored.Identifier, stored.Name, identifier, "開発チーム")
			}
		})
	}
}

func TestUpdate_入力が不正なら最初の誤りのある項目にフォーカスを置く(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-general-focus"
	userID, _ := settingsGeneralSpace(t, tx, identifier, "")

	req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, userID, map[string]string{
		"identifier": identifier,
		"name":       "",
	})
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Update(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	nameInput := regexp.MustCompile(`<input[^>]*id="name"[^>]*>`).FindString(body)
	if !strings.Contains(nameInput, "autofocus") {
		t.Errorf("名前の入力欄にautofocusが無い: %s", nameInput)
	}
	if got := len(regexp.MustCompile(`<input[^>]*autofocus[^>]*>`).FindAllString(body, -1)); got != 1 {
		t.Errorf("autofocusを持つ入力欄の数 = %d、期待値 = 1", got)
	}
}

func TestUpdate_スペースを変更する権限が無いメンバーには404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
		role       model.SpaceRole
		outsider   bool
	}{
		{name: "編集者", identifier: "space-general-update-editor", role: model.SpaceRoleEditor},
		{name: "スペースのメンバーではないユーザー", identifier: "space-general-update-outsider", outsider: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID, spaceID := settingsGeneralSpace(t, tx, tt.identifier, tt.role)
			if tt.outsider {
				userID = testutil.NewUserBuilder(t, tx).
					WithEmail(tt.identifier + "-other@example.com").
					WithAtname(strings.ReplaceAll(tt.identifier, "-", "_") + "_other").
					Build()
			}

			req := newSettingsGeneralRequest(t, http.MethodPatch, tt.identifier, userID, map[string]string{
				"identifier": tt.identifier + "-x",
				"name":       "乗っ取り",
			})
			rr := httptest.NewRecorder()
			setupSettingsGeneralHandler(t, queries).Update(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
			stored := storedSpace(t, queries, spaceID)
			if string(stored.Identifier) != tt.identifier || stored.Name != "開発チーム" {
				t.Errorf("保存されたスペース = (%q, %q)、変わっていないことを期待", stored.Identifier, stored.Name)
			}
		})
	}
}

func TestUpdate_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-general-update-anonymous"
	settingsGeneralSpace(t, tx, identifier, "")

	req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, "", map[string]string{
		"identifier": identifier,
		"name":       "名前",
	})
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Update(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}

func TestUpdate_CSRFトークンが無いと403が返り保存されない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "space-general-csrf"
	userID, spaceID := settingsGeneralSpace(t, tx, identifier, "")

	req := newSettingsGeneralRequest(t, http.MethodPatch, identifier, userID, map[string]string{
		"identifier": "space-general-csrf-x",
		"name":       "名前",
	})
	rr := httptest.NewRecorder()
	csrf := middleware.NewCSRF(&config.Config{SessionHTTPOnly: true})
	csrf.Middleware(http.HandlerFunc(setupSettingsGeneralHandler(t, queries).Update)).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusForbidden)
	}
	stored := storedSpace(t, queries, spaceID)
	if string(stored.Identifier) != identifier {
		t.Errorf("Identifier = %q、期待値 = %q", stored.Identifier, identifier)
	}
}
