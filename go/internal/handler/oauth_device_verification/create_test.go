package oauth_device_verification_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// findDeviceAuthorizationは、フィクスチャのデバイス認可の要求を今の状態で取得する
func findDeviceAuthorization(t *testing.T, f deviceFixture) *model.OAuthDeviceAuthorization {
	t.Helper()

	d, err := repository.NewOAuthDeviceAuthorizationRepository(query.New(testutil.GetTestDB())).FindByDeviceCodeDigest(context.Background(), f.deviceCodeDigest)
	if err != nil {
		t.Fatalf("FindByDeviceCodeDigest()のエラー = %v", err)
	}
	if d == nil {
		t.Fatal("デバイス認可の要求が見つからない")
	}
	return d
}

func TestCreate_コードを照合して同意画面を表示する(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-consent", model.SpaceRoleViewer, true)

	rr := httptest.NewRecorder()
	setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, "198.51.100.10", url.Values{
		"user_code": {strings.ToLower(f.userCode)},
	}))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d。本文 = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	assertResponseHeaders(t, rr)
	assertBodyContains(t, rr,
		"連携するCLIにアクセスを許可しますか？",
		"このログインを自分で開始していない場合は、拒否してください",
		"ログインを開始した日時",
		"<time datetime=",
		"連携先のスペース",
		"@verifying_user",
		// 同意の送信へ表示の形にそろえたコードを引き継ぐ
		`name="user_code" value="`+auth.FormatOAuthUserCode(f.userCode)+`"`,
		`name="granted_scopes" type="checkbox" value="page:read" checked`,
		`name="granted_scopes" type="checkbox" value="page:write" checked`,
		// スペース閲覧者はpage:writeを持たない
		"今の権限では効きません",
		`name="decision" type="submit" value="approve"`,
		`name="decision" type="submit" value="deny"`,
	)
	if d := findDeviceAuthorization(t, f); d.Status != model.OAuthDeviceAuthorizationStatusPending {
		t.Errorf("要求の状態 = %v、期待値は表示だけでは許可待ちのまま", d.Status)
	}
}

func TestCreate_受け付けないコードは入力画面を422で再表示する(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-invalid", model.SpaceRoleAdmin, true)

	for i, code := range []string{"BCDF-GHJ", "ZZZZ-ZZZZ"} {
		rr := httptest.NewRecorder()
		setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, "198.51.100.11", url.Values{"user_code": {code}}))

		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%d件目のステータス = %d、期待値 = %d", i+1, rr.Code, http.StatusUnprocessableEntity)
		}
		assertResponseHeaders(t, rr)
		assertBodyContains(t, rr,
			"コードが正しくないか、有効期限が切れています",
			`aria-invalid="true"`,
			`aria-describedby="user_code-error user_code-hint"`,
			// 入力を消さずに戻す
			`value="`+code+`"`,
		)
	}
}

func TestCreate_許可すると選んだスコープで要求を許可済みにする(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-approve", model.SpaceRoleEditor, true)

	rr := httptest.NewRecorder()
	setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, "198.51.100.12", url.Values{
		"user_code":      {auth.FormatOAuthUserCode(f.userCode)},
		"decision":       {"approve"},
		"granted_scopes": {"page:read"},
	}))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d。本文 = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	assertResponseHeaders(t, rr)
	assertBodyContains(t, rr, "ログインを許可しました", "連携するCLIを使っている端末に戻ってください")

	d := findDeviceAuthorization(t, f)
	if d.Status != model.OAuthDeviceAuthorizationStatusApproved || d.OAuthGrantID == nil {
		t.Fatalf("要求 = %+v、許可を持つ許可済みの要求を期待", d)
	}
	if len(d.Scopes) != 1 || d.Scopes[0] != model.ScopePageRead {
		t.Errorf("要求のスコープ = %v、期待値 = [page:read]", d.Scopes)
	}

	// 使用済みのコードは、誤ったコードと同じく受け付けない
	rr = httptest.NewRecorder()
	setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, "198.51.100.12", url.Values{
		"user_code": {f.userCode},
	}))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("使用済みのコードのステータス = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	assertBodyContains(t, rr, "コードが正しくないか、有効期限が切れています")
}

func TestCreate_選んだスコープに含まれるスコープを補って許可する(t *testing.T) {
	t.Parallel()

	// フィクスチャの要求はpage:readとpage:writeを求める
	f := setupDeviceFixture(t, "devverify-implied", model.SpaceRoleEditor, true)

	// 同意画面は、選んだpage:writeに含まれるpage:readを無効にして送信しない
	rr := httptest.NewRecorder()
	setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, "198.51.100.13", url.Values{
		"user_code":      {auth.FormatOAuthUserCode(f.userCode)},
		"decision":       {"approve"},
		"granted_scopes": {"page:write"},
	}))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d。本文 = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	want := []model.Scope{model.ScopePageRead, model.ScopePageWrite}
	d := findDeviceAuthorization(t, f)
	if d.OAuthGrantID == nil {
		t.Fatalf("要求 = %+v、許可を持つ許可済みの要求を期待", d)
	}
	// 要求のスコープが、デバイスコードの交換で返すトークン応答のscopeになる
	if !slices.Equal(d.Scopes, want) {
		t.Errorf("要求のスコープ = %v、期待値 = %v", d.Scopes, want)
	}
	grant, err := repository.NewOAuthGrantRepository(query.New(testutil.GetTestDB())).FindByID(context.Background(), *d.OAuthGrantID, f.spaceID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if grant == nil || !slices.Equal(grant.Scopes, want) {
		t.Errorf("許可 = %+v、期待値のスコープ = %v", grant, want)
	}
}

func TestCreate_拒否すると要求を拒否済みにする(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-deny", model.SpaceRoleEditor, true)

	rr := httptest.NewRecorder()
	setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, "198.51.100.13", url.Values{
		"user_code": {f.userCode},
		"decision":  {"deny"},
	}))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	assertResponseHeaders(t, rr)
	assertBodyContains(t, rr, "ログインを拒否しました")
	if d := findDeviceAuthorization(t, f); d.Status != model.OAuthDeviceAuthorizationStatusDenied || d.OAuthGrantID != nil {
		t.Errorf("要求 = %+v、許可を持たない拒否済みの要求を期待", d)
	}
}

func TestCreate_スコープを1つも選ばなければ同意画面を422で再表示する(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-no-scope", model.SpaceRoleEditor, true)

	rr := httptest.NewRecorder()
	setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, "198.51.100.14", url.Values{
		"user_code": {f.userCode},
		"decision":  {"approve"},
	}))

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	assertResponseHeaders(t, rr)
	assertBodyContains(t, rr, "許可する操作を1つ以上選んでください", `aria-invalid="true"`)
	// 利用者が選び直せるよう、チェックを外した状態のまま戻す
	if strings.Contains(rr.Body.String(), `value="page:read" checked`) {
		t.Error("選ばなかったスコープがチェックされている")
	}
	if d := findDeviceAuthorization(t, f); d.Status != model.OAuthDeviceAuthorizationStatusPending {
		t.Errorf("要求の状態 = %v、期待値は許可待ちのまま", d.Status)
	}
}

func TestCreate_メンバーでなければ理由を示し許可を受け付けない(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-outsider", model.SpaceRoleEditor, true)
	outsider := setupDeviceFixture(t, "devverify-outsider-user", model.SpaceRoleEditor, true)
	h := setupHandler(t)

	rr := httptest.NewRecorder()
	h.Create(rr, newRequest(t, http.MethodPost, outsider.userID, "198.51.100.15", url.Values{"user_code": {f.userCode}}))
	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	assertBodyContains(t, rr, "連携するCLIと連携できません", "このスペースのメンバーではないため", "ログインを中止する")
	if strings.Contains(rr.Body.String(), `value="approve"`) {
		t.Error("許可できない利用者に許可のボタンが出ている")
	}

	// 改ざんした送信で許可しようとしても受け付けず、理由を示す画面を出し直す
	rr = httptest.NewRecorder()
	h.Create(rr, newRequest(t, http.MethodPost, outsider.userID, "198.51.100.15", url.Values{
		"user_code":      {f.userCode},
		"decision":       {"approve"},
		"granted_scopes": {"page:read"},
	}))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusForbidden)
	}
	assertBodyContains(t, rr, "このスペースのメンバーではないため")
	if d := findDeviceAuthorization(t, f); d.Status != model.OAuthDeviceAuthorizationStatusPending {
		t.Errorf("要求の状態 = %v、期待値は許可待ちのまま", d.Status)
	}
}

func TestCreate_フィーチャーフラグが無効なら404を返す(t *testing.T) {
	t.Parallel()

	f := setupDeviceFixture(t, "devverify-create-flag", model.SpaceRoleAdmin, false)

	rr := httptest.NewRecorder()
	setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, "198.51.100.16", url.Values{"user_code": {f.userCode}}))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
	if d := findDeviceAuthorization(t, f); d.Status != model.OAuthDeviceAuthorizationStatusPending {
		t.Errorf("要求の状態 = %v、期待値は許可待ちのまま", d.Status)
	}
}

func TestCreate_レート制限を超えると429とRetryAfterを返しコードを照合しない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		remoteIP string
		limit    int
		counter  func(f deviceFixture, ip string) string
	}{
		{
			name:     "ユーザー単位",
			key:      "devverify-limit-user",
			remoteIP: "198.51.100.20",
			limit:    10,
			counter: func(f deviceFixture, _ string) string {
				return ratelimit.OAuthDeviceVerificationUserKey(f.userID.String())
			},
		},
		{
			name:     "IPアドレス単位",
			key:      "devverify-limit-ip",
			remoteIP: "198.51.100.21",
			limit:    30,
			counter:  func(_ deviceFixture, ip string) string { return ratelimit.OAuthDeviceVerificationIPKey(ip) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupDeviceFixture(t, tt.key, model.SpaceRoleAdmin, true)
			limiter := newLimiter()
			for range tt.limit {
				if _, err := limiter.Check(context.Background(), ratelimit.CheckInput{Key: tt.counter(f, tt.remoteIP), Limit: tt.limit, Window: 10 * time.Minute}); err != nil {
					t.Fatalf("Check()のエラー = %v", err)
				}
			}

			rr := httptest.NewRecorder()
			setupHandler(t).Create(rr, newRequest(t, http.MethodPost, f.userID, tt.remoteIP, url.Values{
				"user_code":      {f.userCode},
				"decision":       {"approve"},
				"granted_scopes": {"page:read"},
			}))

			if rr.Code != http.StatusTooManyRequests {
				t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusTooManyRequests)
			}
			assertResponseHeaders(t, rr)
			if seconds, err := strconv.Atoi(rr.Header().Get("Retry-After")); err != nil || seconds < 1 || seconds > 600 {
				t.Errorf("Retry-After = %q、期待値は1〜600秒", rr.Header().Get("Retry-After"))
			}
			assertBodyContains(t, rr, "リクエストが多すぎます")
			if d := findDeviceAuthorization(t, f); d.Status != model.OAuthDeviceAuthorizationStatusPending {
				t.Errorf("要求の状態 = %v、期待値は許可待ちのまま", d.Status)
			}
		})
	}
}
