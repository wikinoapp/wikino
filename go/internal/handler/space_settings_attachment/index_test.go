package space_settings_attachment_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	settingsattachment "github.com/wikinoapp/wikino/go/internal/handler/space_settings_attachment"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newAttachmentsRequestは添付ファイルの管理の画面向けに、chiのURLパラメータ・CSRFトークン・ログイン中の
// ユーザー・画面を描画するロケールを載せたリクエストを組み立てる。userIDが空のときはユーザーを載せない。
// attachmentIDが空でなければ、削除の対象としてURLパラメータに載せる。
func newAttachmentsRequest(t *testing.T, method, target, identifier, attachmentID string, userID model.UserID) *http.Request {
	t.Helper()

	req := httptest.NewRequest(method, target, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)
	if attachmentID != "" {
		rctx.URLParams.Add("attachment_id", attachmentID)
	}

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "space-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// setupAttachmentsHandlerは、渡したクエリ・プール・オブジェクトストレージ・ディスパッチャーを使うハンドラーを
// 組み立てる。一覧は自身でトランザクションを開かないため、一覧のテストはフィクスチャをトランザクションに
// 閉じ込めてよい。削除は自身でトランザクションを開くため、削除のテストはDBに直接コミットする。
func setupAttachmentsHandler(t *testing.T, db *sql.DB, queries *query.Queries, objectStorage storage.ObjectStorage, d *dispatcher.Dispatcher) *settingsattachment.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost"}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	attachmentRepo := repository.NewAttachmentRepository(queries)

	return settingsattachment.NewHandler(
		cfg,
		session.NewFlashManager("", false, true),
		usecase.NewGetSpaceSettingsAttachmentsUsecase(spaceRepo, spaceMemberRepo, attachmentRepo, objectStorage),
		usecase.NewDeleteAttachmentUsecase(db, spaceRepo, spaceMemberRepo, attachmentRepo, objectStorage, d),
	)
}

// attachmentsSpaceはスペースと、渡したロールを持つメンバー1人を用意し、ユーザー・スペース・メンバーのIDを返す。
func attachmentsSpace(t *testing.T, tx *sql.Tx, identifier string, role model.SpaceRole) (model.UserID, model.SpaceID, model.SpaceMemberID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).WithName("開発チーム").Build()
	memberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).WithRole(role).Build()
	return userID, spaceID, memberID
}

func addAttachment(t *testing.T, tx *sql.Tx, spaceID model.SpaceID, memberID model.SpaceMemberID, filename, contentType string, attachedAt time.Time) model.AttachmentID {
	t.Helper()
	return testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(memberID).
		WithFilename(filename).
		WithContentType(contentType).
		WithByteSize(1536).
		WithAttachedAt(attachedAt).
		Build()
}

// serveIndexは一覧のハンドラーを、トランザクションのクエリとフェイクのオブジェクトストレージで呼ぶ。
func serveIndex(t *testing.T, tx *sql.Tx, objectStorage storage.ObjectStorage, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	setupAttachmentsHandler(t, testutil.GetTestDB(), testutil.QueriesWithTx(tx), objectStorage, nil).Index(rr, req)
	return rr
}

var attachmentsBase = time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

func TestIndex_添付ファイルが新しい順に表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	identifier := "ssa-index"
	userID, spaceID, memberID := attachmentsSpace(t, tx, identifier, model.SpaceRoleAdmin)
	png := addAttachment(t, tx, spaceID, memberID, "older-photo.png", "image/png", attachmentsBase)
	pdf := addAttachment(t, tx, spaceID, memberID, "newer-doc.pdf", "application/pdf", attachmentsBase.Add(time.Hour))

	req := newAttachmentsRequest(t, http.MethodGet, "/s/"+identifier+"/settings/attachments", identifier, "", userID)
	rr := serveIndex(t, tx, storage.NewFakeObjectStorage(), req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}

	body := rr.Body.String()
	for _, want := range []string{
		`<title>添付ファイル | 開発チーム</title>`,
		`href="/attachments/` + string(png) + `"`,
		`href="/attachments/` + string(pdf) + `"`,
		`target="_blank"`,
		`rel="noopener"`,
		"image/png • 1.5KB",
		"2026/11/20 00:00にアップロード",
		`action="/s/` + identifier + `/settings/attachments/` + string(png) + `"`,
		`name="_method" value="DELETE"`,
		`name="csrf_token" value="test-csrf-token"`,
		`aria-label="older-photo.pngを削除"`,
		`data-confirm="「older-photo.png」を削除しますか？削除すると元に戻せません。"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if strings.Index(body, "newer-doc.pdf") > strings.Index(body, "older-photo.png") {
		t.Error("新しい添付ファイルが先に並んでいない")
	}
}

func TestIndex_画像にだけ遅延読み込みのサムネイルが出る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	identifier := "ssa-thumb"
	userID, spaceID, memberID := attachmentsSpace(t, tx, identifier, model.SpaceRoleAdmin)
	addAttachment(t, tx, spaceID, memberID, "photo.png", "image/png", attachmentsBase)
	addAttachment(t, tx, spaceID, memberID, "icon.svg", "image/svg+xml", attachmentsBase)
	addAttachment(t, tx, spaceID, memberID, "doc.pdf", "application/pdf", attachmentsBase)

	req := newAttachmentsRequest(t, http.MethodGet, "/s/"+identifier+"/settings/attachments", identifier, "", userID)
	rr := serveIndex(t, tx, storage.NewFakeObjectStorage(), req)

	// サムネイルの署名付きURLを含む応答を、ブラウザにも途中のキャッシュにも保存させない
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
	}

	imgs := regexp.MustCompile(`<img [^>]*>`).FindAllString(rr.Body.String(), -1)
	if len(imgs) != 2 {
		t.Fatalf("<img>の数 = %d、期待値 = 2 (画像の添付ファイルだけ)", len(imgs))
	}
	for _, img := range imgs {
		for _, want := range []string{`alt=""`, `loading="lazy"`} {
			if !strings.Contains(img, want) {
				t.Errorf("%sに%qが含まれていない", img, want)
			}
		}
	}
}

func TestIndex_オブジェクトストレージが無くても一覧が表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	identifier := "ssa-nostorage"
	userID, spaceID, memberID := attachmentsSpace(t, tx, identifier, model.SpaceRoleAdmin)
	addAttachment(t, tx, spaceID, memberID, "photo.png", "image/png", attachmentsBase)

	req := newAttachmentsRequest(t, http.MethodGet, "/s/"+identifier+"/settings/attachments", identifier, "", userID)
	rr := serveIndex(t, tx, nil, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "photo.png") {
		t.Error("添付ファイルが表示されていない")
	}
	if strings.Contains(body, "<img ") {
		t.Error("オブジェクトストレージが無いのにサムネイルが出ている")
	}
}

func TestIndex_添付ファイルが無いときは空の表示になる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	identifier := "ssa-empty"
	userID, _, _ := attachmentsSpace(t, tx, identifier, model.SpaceRoleAdmin)

	req := newAttachmentsRequest(t, http.MethodGet, "/s/"+identifier+"/settings/attachments", identifier, "", userID)
	rr := serveIndex(t, tx, storage.NewFakeObjectStorage(), req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "添付ファイルが見つかりません") {
		t.Error("空の表示が含まれていない")
	}
}

func TestIndex_ファイル名で絞り込み検索語を検索欄とページ送りに引き継ぐ(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	identifier := "ssa-search"
	userID, spaceID, memberID := attachmentsSpace(t, tx, identifier, model.SpaceRoleAdmin)
	for i := range 51 {
		addAttachment(t, tx, spaceID, memberID, fmt.Sprintf("Meeting-%02d.png", i), "image/png", attachmentsBase.Add(time.Duration(i)*time.Minute))
	}
	addAttachment(t, tx, spaceID, memberID, "other.png", "image/png", attachmentsBase)

	req := newAttachmentsRequest(t, http.MethodGet, "/s/"+identifier+"/settings/attachments?q=meeting", identifier, "", userID)
	rr := serveIndex(t, tx, storage.NewFakeObjectStorage(), req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, "other.png") {
		t.Error("検索語に一致しない添付ファイルが表示されている")
	}
	if got := strings.Count(body, `href="/attachments/`); got != 50 {
		t.Errorf("1ページ目の件数 = %d、期待値 = 50", got)
	}
	for _, want := range []string{
		`value="meeting"`,
		`href="/s/` + identifier + `/settings/attachments?q=meeting&amp;page=2"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	req = newAttachmentsRequest(t, http.MethodGet, "/s/"+identifier+"/settings/attachments?q=meeting&page=2", identifier, "", userID)
	rr = serveIndex(t, tx, storage.NewFakeObjectStorage(), req)
	body = rr.Body.String()
	if got := strings.Count(body, `href="/attachments/`); got != 1 {
		t.Errorf("2ページ目の件数 = %d、期待値 = 1", got)
	}
	if !strings.Contains(body, "Meeting-00.png") {
		t.Error("2ページ目に最も古い添付ファイルが表示されていない")
	}
	if !strings.Contains(body, `<title>添付ファイル (2 ページ目) | 開発チーム</title>`) {
		t.Error("2ページ目のタイトルになっていない")
	}
}

func TestIndex_範囲の外のページと不正なページは404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	identifier := "ssa-page"
	userID, spaceID, memberID := attachmentsSpace(t, tx, identifier, model.SpaceRoleAdmin)
	addAttachment(t, tx, spaceID, memberID, "photo.png", "image/png", attachmentsBase)

	for _, page := range []string{"2", "99999999999"} {
		req := newAttachmentsRequest(t, http.MethodGet, "/s/"+identifier+"/settings/attachments?page="+page, identifier, "", userID)
		if rr := serveIndex(t, tx, storage.NewFakeObjectStorage(), req); rr.Code != http.StatusNotFound {
			t.Errorf("page=%sのステータスコード = %d、期待値 = %d", page, rr.Code, http.StatusNotFound)
		}
	}
}

func TestIndex_到達できない場合は404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	identifier := "ssa-index-deny"
	_, spaceID, _ := attachmentsSpace(t, tx, identifier, model.SpaceRoleAdmin)
	editorID := testutil.NewUserBuilder(t, tx).WithEmail("ssa-index-editor@example.com").WithAtname("ssa_index_editor").Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(editorID).WithRole(model.SpaceRoleEditor).Build()
	outsiderID := testutil.NewUserBuilder(t, tx).WithEmail("ssa-index-outsider@example.com").WithAtname("ssa_index_outsider").Build()

	for name, userID := range map[string]model.UserID{"権限の無いメンバー": editorID, "メンバーでない閲覧者": outsiderID} {
		req := newAttachmentsRequest(t, http.MethodGet, "/s/"+identifier+"/settings/attachments", identifier, "", userID)
		if rr := serveIndex(t, tx, storage.NewFakeObjectStorage(), req); rr.Code != http.StatusNotFound {
			t.Errorf("%sのステータスコード = %d、期待値 = %d", name, rr.Code, http.StatusNotFound)
		}
	}
}

func TestIndex_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	req := newAttachmentsRequest(t, http.MethodGet, "/s/ssa-anon/settings/attachments", "ssa-anon", "", "")
	rr := serveIndex(t, tx, storage.NewFakeObjectStorage(), req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
	}
}

func TestIndex_HEADでも200が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	identifier := "ssa-head"
	userID, _, _ := attachmentsSpace(t, tx, identifier, model.SpaceRoleAdmin)

	req := newAttachmentsRequest(t, http.MethodHead, "/s/"+identifier+"/settings/attachments", identifier, "", userID)
	if rr := serveIndex(t, tx, storage.NewFakeObjectStorage(), req); rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
}
