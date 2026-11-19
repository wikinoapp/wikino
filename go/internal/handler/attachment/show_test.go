package attachment_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/attachment"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// newShowRequestはchiのURLパラメータと閲覧者を載せたリクエストを組み立てる。viewerがnilのときは
// ユーザーを載せず、ログインしていない閲覧者の経路へ到達できるようにする
func newShowRequest(method, attachmentID string, viewer *model.User) *http.Request {
	req := httptest.NewRequest(method, "/attachments/"+attachmentID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("attachment_id", attachmentID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if viewer != nil {
		ctx = middleware.SetUserToContext(ctx, viewer)
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	return req.WithContext(ctx)
}

func newHandler(tx *sql.Tx, objectStorage storage.ObjectStorage) *attachment.Handler {
	q := testutil.QueriesWithTx(tx)
	return attachment.NewHandler(usecase.NewGetAttachmentUsecase(
		repository.NewAttachmentRepository(q),
		repository.NewSpaceMemberRepository(q),
		objectStorage,
	))
}

// showFixtureはスペースのメンバーと、公開トピックと非公開トピックのページがそれぞれ参照する添付ファイル
type showFixture struct {
	member            *model.User
	publicAttachment  model.AttachmentID
	privateAttachment model.AttachmentID
}

func setupShowFixture(t *testing.T, tx *sql.Tx, prefix string) showFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).WithEmail(prefix + "@example.com").WithAtname(prefix).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(prefix).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	parRepo := repository.NewPageAttachmentReferenceRepository(testutil.QueriesWithTx(tx))

	attachmentFor := func(number int32, visibility model.TopicVisibility, name string) model.AttachmentID {
		topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(number).WithName(name).
			WithVisibility(int32(visibility)).Build()
		pageID := testutil.NewPageBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topicID).
			WithNumber(model.PageNumber(number)).WithTitle(name).Build()
		id := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
			WithFilename(name + ".png").WithContentType("image/png").WithBlobKey(prefix + "-" + name).Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{id}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}
		return id
	}

	return showFixture{
		member:            &model.User{ID: userID},
		publicAttachment:  attachmentFor(1, model.TopicVisibilityPublic, "public"),
		privateAttachment: attachmentFor(2, model.TopicVisibilityPrivate, "private"),
	}
}

func TestShow(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupShowFixture(t, tx, "attachment-show")
	h := newHandler(tx, storage.NewFakeObjectStorage())

	tests := []struct {
		name         string
		method       string
		attachmentID string
		viewer       *model.User
		wantStatus   int
		wantKey      string
	}{
		{name: "メンバーは非公開トピックの添付ファイルの署名付きURLへリダイレクトされる", method: http.MethodGet, attachmentID: string(f.privateAttachment), viewer: f.member, wantStatus: http.StatusFound, wantKey: "attachment-show-private"},
		{name: "HEADもリダイレクトする", method: http.MethodHead, attachmentID: string(f.privateAttachment), viewer: f.member, wantStatus: http.StatusFound, wantKey: "attachment-show-private"},
		{name: "ログインしていなくても公開トピックの添付ファイルへリダイレクトされる", method: http.MethodGet, attachmentID: string(f.publicAttachment), wantStatus: http.StatusFound, wantKey: "attachment-show-public"},
		{name: "ログインしていなければ非公開トピックの添付ファイルは404", method: http.MethodGet, attachmentID: string(f.privateAttachment), wantStatus: http.StatusNotFound},
		{name: "存在しないIDは404", method: http.MethodGet, attachmentID: "00000000-0000-0000-0000-000000000000", viewer: f.member, wantStatus: http.StatusNotFound},
		{name: "不正な形のIDは404", method: http.MethodGet, attachmentID: "not-a-uuid", viewer: f.member, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.Show(rr, newShowRequest(tt.method, tt.attachmentID, tt.viewer))

			if rr.Code != tt.wantStatus {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusFound {
				if location := rr.Header().Get("Location"); location != "" {
					t.Errorf("Location = %q、期待値 = 空", location)
				}
				return
			}

			if got := rr.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
			}
			location, err := url.Parse(rr.Header().Get("Location"))
			if err != nil {
				t.Fatalf("Locationのパースに失敗: %v", err)
			}
			if location.Path != "/"+tt.wantKey {
				t.Errorf("Locationのパス = %q、期待値 = %q", location.Path, "/"+tt.wantKey)
			}
			if got := location.Query().Get("expires_in"); got != "300" {
				t.Errorf("expires_in = %q、期待値 = %q", got, "300")
			}
			if got := location.Query().Get("response-content-type"); got != "image/png" {
				t.Errorf("response-content-type = %q、期待値 = %q", got, "image/png")
			}
		})
	}
}

func TestShow_ObjectStorageNotConfigured(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupShowFixture(t, tx, "attachment-show-nostorage")
	h := newHandler(tx, nil)

	rr := httptest.NewRecorder()
	h.Show(rr, newShowRequest(http.MethodGet, string(f.publicAttachment), f.member))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusInternalServerError)
	}
	if location := rr.Header().Get("Location"); location != "" {
		t.Errorf("Location = %q、期待値 = 空", location)
	}
}
