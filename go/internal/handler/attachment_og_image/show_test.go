package attachment_og_image_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler/attachment_og_image"
	"github.com/wikinoapp/wikino/go/internal/image"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// テスト用のimgproxy設定。HMAC-SHA256署名が成立する16進数のダミー値。
const (
	testImgproxyURL    = "https://imgproxy.test.local"
	testImgproxyKeyHex = "0123456789abcdef0123456789abcdef"
	testImgproxySalt   = "fedcba9876543210fedcba9876543210"
	testR2Bucket       = "test-bucket"
)

func newRequestWithAttachmentID(t *testing.T, attachmentID string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/attachments/"+attachmentID+"/og_image", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("attachment_id", attachmentID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	return req, httptest.NewRecorder()
}

func newOgImageHandler(t *testing.T, attachmentRepo *repository.AttachmentRepository, withBuilder bool) *attachment_og_image.Handler {
	t.Helper()
	uc := usecase.NewGetAttachmentOgImageUsecase(attachmentRepo)
	if !withBuilder {
		return attachment_og_image.NewHandler(nil, uc)
	}
	helper, err := image.NewHelper(testImgproxyURL, testImgproxyKeyHex, testImgproxySalt)
	if err != nil {
		t.Fatalf("image.NewHelper()のエラー = %v", err)
	}
	builder, err := image.NewOgImageBuilder(helper, testR2Bucket)
	if err != nil {
		t.Fatalf("image.NewOgImageBuilder()のエラー = %v", err)
	}
	return attachment_og_image.NewHandler(builder, uc)
}

// referencedAttachmentParamsはnewReferencedAttachmentの入力。og:imageのケースが実際に
// 変える項目だけを名前付きで並べ、「トピックのvisibilityは何か」「ページはゴミ箱に入っているか」
// だけを読めば済むようにする。
type referencedAttachmentParams struct {
	// prefixはユーザー・スペースの識別子の種になるため、ケースごとに一意にする。
	prefix     string
	visibility model.TopicVisibility
	trashed    bool
	// filenameとcontentTypeは空のときAttachmentBuilderの既定値になる。
	filename    string
	contentType string
}

// newReferencedAttachmentはスペース・トピック・ページと、そのページから参照される
// 添付ファイルを作成し、添付ファイルのIDを返す。
func newReferencedAttachment(t *testing.T, tx *sql.Tx, params referencedAttachmentParams) model.AttachmentID {
	t.Helper()

	spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, params.prefix)
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("topic").
		WithVisibility(int32(params.visibility)).
		Build()

	pageBuilder := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Page")
	if params.trashed {
		pageBuilder = pageBuilder.WithTrashed()
	}
	pageID := pageBuilder.Build()

	attachmentBuilder := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID)
	if params.filename != "" {
		attachmentBuilder = attachmentBuilder.WithFilename(params.filename)
	}
	if params.contentType != "" {
		attachmentBuilder = attachmentBuilder.WithContentType(params.contentType)
	}
	attachmentID := attachmentBuilder.Build()

	parRepo := repository.NewPageAttachmentReferenceRepository(testutil.QueriesWithTx(tx))
	if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
		t.Fatalf("CreateBatch()のエラー = %v", err)
	}

	return attachmentID
}

func TestShow(t *testing.T) {
	t.Parallel()

	t.Run("正常系: 公開トピックのページから参照されている添付は302 redirectを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)

		attachmentID := newReferencedAttachment(t, tx, referencedAttachmentParams{
			prefix:      "ogimg-show-1",
			visibility:  model.TopicVisibilityPublic,
			filename:    "og.png",
			contentType: "image/png",
		})

		h := newOgImageHandler(t, attachmentRepo, true)
		req, rr := newRequestWithAttachmentID(t, string(attachmentID))
		h.Show(rr, req)

		if rr.Code != http.StatusFound {
			t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
		}
		location := rr.Header().Get("Location")
		if !strings.HasPrefix(location, testImgproxyURL+"/") {
			t.Errorf("Location = %q、期待する接頭辞 = %q", location, testImgproxyURL+"/")
		}
		if !strings.Contains(location, "resize:fit:1200:630") {
			t.Errorf("Location = %q、リサイズの指定が無い", location)
		}
		if !strings.Contains(location, "format:jpg") {
			t.Errorf("Location = %q、format:jpgが無い", location)
		}
		if !strings.Contains(location, "s3://"+testR2Bucket+"/") {
			t.Errorf("Location = %q、s3のソースURLが無い", location)
		}
		if got := rr.Header().Get("Cache-Control"); got != "public, max-age=60, s-maxage=300" {
			t.Errorf("Cache-Control = %q、期待値 = %q", got, "public, max-age=60, s-maxage=300")
		}
	})

	t.Run("異常系: 非公開トピックのみから参照されている添付は404を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)

		attachmentID := newReferencedAttachment(t, tx, referencedAttachmentParams{
			prefix:     "ogimg-show-2",
			visibility: model.TopicVisibilityPrivate,
		})

		h := newOgImageHandler(t, attachmentRepo, true)
		req, rr := newRequestWithAttachmentID(t, string(attachmentID))
		h.Show(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
		}
		if rr.Header().Get("Location") != "" {
			t.Errorf("Locationヘッダー = %q、期待値 = 空", rr.Header().Get("Location"))
		}
		assertNoStoreCacheControl(t, rr)
	})

	t.Run("異常系: 公開トピックのゴミ箱に入ったページのみから参照されている添付は404を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)

		attachmentID := newReferencedAttachment(t, tx, referencedAttachmentParams{
			prefix:     "ogimg-show-3",
			visibility: model.TopicVisibilityPublic,
			trashed:    true,
		})

		h := newOgImageHandler(t, attachmentRepo, true)
		req, rr := newRequestWithAttachmentID(t, string(attachmentID))
		h.Show(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
		}
		if rr.Header().Get("Location") != "" {
			t.Errorf("Locationヘッダー = %q、期待値 = 空", rr.Header().Get("Location"))
		}
		assertNoStoreCacheControl(t, rr)
	})

	t.Run("異常系: 存在しないattachment_idは404を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)

		h := newOgImageHandler(t, attachmentRepo, true)
		req, rr := newRequestWithAttachmentID(t, "00000000-0000-0000-0000-000000000000")
		h.Show(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
		}
		assertNoStoreCacheControl(t, rr)
	})

	t.Run("異常系: UUID形式でないattachment_idは404を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)

		h := newOgImageHandler(t, attachmentRepo, true)
		req, rr := newRequestWithAttachmentID(t, "not-a-uuid")
		h.Show(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
		}
		assertNoStoreCacheControl(t, rr)
	})

	t.Run("異常系: ogImageBuilderがnil (imgproxy未設定) の場合は500を返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		attachmentRepo := repository.NewAttachmentRepository(q)

		h := newOgImageHandler(t, attachmentRepo, false)
		req, rr := newRequestWithAttachmentID(t, "00000000-0000-0000-0000-000000000000")
		h.Show(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusInternalServerError)
		}
		assertNoStoreCacheControl(t, rr)
	})
}

// assertNoStoreCacheControlは404 / 500レスポンスでCache-Control: private, no-storeが
// セットされていることを検証する。
func assertNoStoreCacheControl(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	got := rr.Header().Get("Cache-Control")
	want := "private, no-store"
	if got != want {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, want)
	}
}
