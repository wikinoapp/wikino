package attachment_signed_url_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/attachment_signed_url"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func newHandler(tx *sql.Tx, objectStorage storage.ObjectStorage) *attachment_signed_url.Handler {
	q := testutil.QueriesWithTx(tx)
	return attachment_signed_url.NewHandler(usecase.NewGetAttachmentSignedURLsUsecase(
		repository.NewAttachmentRepository(q),
		repository.NewSpaceMemberRepository(q),
		objectStorage,
	))
}

// newCreateRequestは本文bodyと閲覧者を載せたリクエストを組み立てる。viewerがnilのときはユーザーを載せない
func newCreateRequest(body string, viewer *model.User) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/attachments/signed_urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if viewer != nil {
		req = req.WithContext(middleware.SetUserToContext(req.Context(), viewer))
	}
	return req
}

// createFixtureはスペースのメンバーと、公開トピックと非公開トピックのページがそれぞれ参照する添付ファイル
type createFixture struct {
	member            *model.User
	publicAttachment  model.AttachmentID
	privateAttachment model.AttachmentID
}

func setupCreateFixture(t *testing.T, tx *sql.Tx, prefix string) createFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).WithEmail(prefix + "@example.com").WithAtname(strings.ReplaceAll(prefix, "-", "_")).Build()
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

	return createFixture{
		member:            &model.User{ID: userID},
		publicAttachment:  attachmentFor(1, model.TopicVisibilityPublic, "public"),
		privateAttachment: attachmentFor(2, model.TopicVisibilityPrivate, "private"),
	}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupCreateFixture(t, tx, "attachment-signed-urls")
	h := newHandler(tx, storage.NewFakeObjectStorage())
	ids := `{"attachment_ids":["` + string(f.publicAttachment) + `","` + string(f.privateAttachment) + `","00000000-0000-0000-0000-000000000000","not-a-uuid"]}`

	tests := []struct {
		name     string
		body     string
		viewer   *model.User
		wantKeys map[string]string
	}{
		{
			name:   "メンバーには見られる添付ファイルの署名付きURLだけを返す",
			body:   ids,
			viewer: f.member,
			wantKeys: map[string]string{
				string(f.publicAttachment):  "attachment-signed-urls-public",
				string(f.privateAttachment): "attachment-signed-urls-private",
			},
		},
		{
			name:     "ログインしていなければ公開トピックの添付ファイルだけを返す",
			body:     ids,
			wantKeys: map[string]string{string(f.publicAttachment): "attachment-signed-urls-public"},
		},
		{
			name:     "IDが空なら空のオブジェクトを返す",
			body:     `{"attachment_ids":[]}`,
			viewer:   f.member,
			wantKeys: map[string]string{},
		},
		{
			name:     "JSONの末尾が空白だけなら受け付ける",
			body:     "{\"attachment_ids\":[]} \n\t",
			viewer:   f.member,
			wantKeys: map[string]string{},
		},
		{
			name:     "本文が空なら空のオブジェクトを返す",
			body:     "",
			viewer:   f.member,
			wantKeys: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.Create(rr, newCreateRequest(tt.body, tt.viewer))

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			if got := rr.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
			}
			if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
				t.Errorf("Content-Type = %q、期待値 = application/json", got)
			}

			var resp struct {
				SignedURLs map[string]string `json:"signed_urls"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("応答のパースに失敗: %v (本文 = %s)", err, rr.Body.String())
			}
			if resp.SignedURLs == nil {
				t.Fatalf("signed_urls = null、期待値 = オブジェクト (本文 = %s)", rr.Body.String())
			}
			if len(resp.SignedURLs) != len(tt.wantKeys) {
				t.Fatalf("signed_urlsの件数 = %d、期待値 = %d (%v)", len(resp.SignedURLs), len(tt.wantKeys), resp.SignedURLs)
			}
			for id, wantKey := range tt.wantKeys {
				parsed, err := url.Parse(resp.SignedURLs[id])
				if err != nil {
					t.Fatalf("署名付きURLのパースに失敗: %v", err)
				}
				if parsed.Path != "/"+wantKey {
					t.Errorf("%sの署名付きURLのパス = %q、期待値 = %q", id, parsed.Path, "/"+wantKey)
				}
			}
		})
	}
}

func TestCreate_InvalidBody(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := newHandler(tx, storage.NewFakeObjectStorage())

	for _, body := range []string{
		`{`,
		`{"attachment_ids":"not-a-list"}`,
		`{"attachment_ids":[1,2]}`,
		`{"attachment_ids":[]} garbage`,
		`{"attachment_ids":[]} {"attachment_ids":[]}`,
		`{"attachment_ids":[]} null`,
	} {
		t.Run(body, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.Create(rr, newCreateRequest(body, nil))

			if rr.Code != http.StatusBadRequest {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestCreate_ObjectStorageNotConfigured(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupCreateFixture(t, tx, "attachment-signed-urls-nostorage")
	h := newHandler(tx, nil)

	rr := httptest.NewRecorder()
	h.Create(rr, newCreateRequest(`{"attachment_ids":["`+string(f.publicAttachment)+`"]}`, f.member))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusInternalServerError)
	}
}
