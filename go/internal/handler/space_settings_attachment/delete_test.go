package space_settings_attachment_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// deleteFixtureは、削除のテストが使うスペース・管理者・オブジェクトストレージ・ディスパッチャー。
// 削除は自身でトランザクションを開くため、フィクスチャはDBに直接コミットする。
type deleteFixture struct {
	identifier    string
	spaceID       model.SpaceID
	adminID       model.UserID
	memberID      model.SpaceMemberID
	objectStorage *storage.FakeObjectStorage
	dispatcher    *dispatcher.Dispatcher
}

func setupDeleteFixture(t *testing.T, identifier string) *deleteFixture {
	t.Helper()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	memberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).WithRole(model.SpaceRoleAdmin).Build()
	jobClient, _ := testutil.SetupRiverClient(t, db)

	return &deleteFixture{
		identifier:    identifier,
		spaceID:       spaceID,
		adminID:       userID,
		memberID:      memberID,
		objectStorage: storage.NewFakeObjectStorage(),
		dispatcher:    dispatcher.NewDispatcher(jobClient, jobClient),
	}
}

// addは添付ファイルの行を作って実体を置き、IDとblobのキーを返す。
func (f *deleteFixture) add(t *testing.T, spaceID model.SpaceID, memberID model.SpaceMemberID, name string) (model.AttachmentID, string) {
	t.Helper()

	key := f.identifier + "-" + name
	id := testutil.NewAttachmentBuilderDB(t, testutil.GetTestDB()).
		WithSpaceID(spaceID).
		WithSpaceMemberID(memberID).
		WithFilename(name + ".png").
		WithContentType("image/png").
		WithBlobKey(key).
		Build()
	f.objectStorage.Put(key, []byte("original"), "image/png")
	return id, key
}

func (f *deleteFixture) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	db := testutil.GetTestDB()
	return setupAttachmentsHandler(t, db, query.New(db), f.objectStorage, f.dispatcher).Delete
}

func (f *deleteFixture) request(t *testing.T, attachmentID string, userID model.UserID) *http.Request {
	t.Helper()
	return newAttachmentsRequest(t, http.MethodDelete, "/s/"+f.identifier+"/settings/attachments/"+attachmentID, f.identifier, attachmentID, userID)
}

func attachmentExists(t *testing.T, id model.AttachmentID) bool {
	t.Helper()
	var exists bool
	if err := testutil.GetTestDB().QueryRowContext(t.Context(), "SELECT EXISTS(SELECT 1 FROM attachments WHERE id = $1)", string(id)).Scan(&exists); err != nil {
		t.Fatalf("添付ファイルの確認に失敗: %v", err)
	}
	return exists
}

func TestDelete_削除して一覧へリダイレクトする(t *testing.T) {
	t.Parallel()

	f := setupDeleteFixture(t, "ssa-del-ok")
	id, key := f.add(t, f.spaceID, f.memberID, "target")
	otherID, otherKey := f.add(t, f.spaceID, f.memberID, "kept")

	rr := httptest.NewRecorder()
	f.handler(t)(rr, f.request(t, string(id), f.adminID))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got, want := rr.Header().Get("Location"), "/s/"+f.identifier+"/settings/attachments"; got != want {
		t.Errorf("Location = %q、期待値 = %q", got, want)
	}
	if !strings.Contains(rr.Header().Get("Set-Cookie"), "flash") {
		t.Error("フラッシュが設定されていない")
	}
	if attachmentExists(t, id) {
		t.Error("削除した添付ファイルの行が残っている")
	}
	if objectExists(t, f.objectStorage, key) {
		t.Error("削除した添付ファイルの実体が残っている")
	}
	if !attachmentExists(t, otherID) {
		t.Error("ほかの添付ファイルの行が消えた")
	}
	if !objectExists(t, f.objectStorage, otherKey) {
		t.Error("ほかの添付ファイルの実体が消えた")
	}
}

func TestDelete_削除できない添付ファイルは404が返り残る(t *testing.T) {
	t.Parallel()

	f := setupDeleteFixture(t, "ssa-del-deny")
	db := testutil.GetTestDB()
	id, _ := f.add(t, f.spaceID, f.memberID, "target")

	otherUserID := testutil.NewUserBuilderDB(t, db).WithEmail("ssa-del-deny-other@example.com").WithAtname("ssa_del_deny_other").Build()
	otherSpaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("ssa-del-deny-other").Build()
	otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(otherSpaceID).WithUserID(otherUserID).WithRole(model.SpaceRoleAdmin).Build()
	otherSpaceAttachmentID, _ := f.add(t, otherSpaceID, otherMemberID, "other-space")

	editorID := testutil.NewUserBuilderDB(t, db).WithEmail("ssa-del-deny-editor@example.com").WithAtname("ssa_del_deny_editor").Build()
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(f.spaceID).WithUserID(editorID).WithRole(model.SpaceRoleEditor).Build()

	tests := []struct {
		name         string
		attachmentID string
		userID       model.UserID
	}{
		{name: "ほかのスペースの添付ファイル", attachmentID: string(otherSpaceAttachmentID), userID: f.adminID},
		{name: "存在しないID", attachmentID: "00000000-0000-0000-0000-000000000000", userID: f.adminID},
		{name: "不正な形のID", attachmentID: "not-a-uuid", userID: f.adminID},
		{name: "attachment:deleteを持たないメンバー", attachmentID: string(id), userID: editorID},
		{name: "メンバーでない閲覧者", attachmentID: string(id), userID: otherUserID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			f.handler(t)(rr, f.request(t, tt.attachmentID, tt.userID))
			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}

	if !attachmentExists(t, id) {
		t.Error("削除できないはずの添付ファイルが消えた")
	}
	if !attachmentExists(t, otherSpaceAttachmentID) {
		t.Error("ほかのスペースの添付ファイルが消えた")
	}
}

func TestDelete_CSRFトークンが無いと403が返り削除されない(t *testing.T) {
	t.Parallel()

	f := setupDeleteFixture(t, "ssa-del-csrf")
	id, _ := f.add(t, f.spaceID, f.memberID, "target")

	rr := httptest.NewRecorder()
	csrf := middleware.NewCSRF(&config.Config{SessionHTTPOnly: true})
	csrf.Middleware(f.handler(t)).ServeHTTP(rr, f.request(t, string(id), f.adminID))

	if rr.Code != http.StatusForbidden {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusForbidden)
	}
	if !attachmentExists(t, id) {
		t.Error("CSRFトークンの無い削除で添付ファイルが消えた")
	}
}

func TestDelete_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	f := setupDeleteFixture(t, "ssa-del-anon")
	id, _ := f.add(t, f.spaceID, f.memberID, "target")

	rr := httptest.NewRecorder()
	f.handler(t)(rr, f.request(t, string(id), ""))

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
	}
	if !attachmentExists(t, id) {
		t.Error("未ログインの削除で添付ファイルが消えた")
	}
}

func objectExists(t *testing.T, objectStorage *storage.FakeObjectStorage, key string) bool {
	t.Helper()
	exists, err := objectStorage.Exists(t.Context(), key)
	if err != nil {
		t.Fatalf("実体の確認に失敗: %v", err)
	}
	return exists
}
