package attachment_presign_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/handler/attachment_presign"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// recordingJobInserterは投入されたジョブを数える
type recordingJobInserter struct{ calls int }

func (r *recordingJobInserter) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	r.calls++
	return &rivertype.JobInsertResult{}, nil
}

// presignFixtureは事前署名のテストに使うスペースと、権限の違うユーザー
type presignFixture struct {
	identifier string
	spaceID    model.SpaceID
	editor     *model.User
	viewer     *model.User
}

func setupPresignFixture(t *testing.T, suffix string) presignFixture {
	t.Helper()

	db := testutil.GetTestDB()
	newUser := func(role string) model.UserID {
		return testutil.NewUserBuilderDB(t, db).
			WithEmail("presign-handler-" + role + "-" + suffix + "@example.com").
			WithAtname("presign_handler_" + role + "_" + strings.ReplaceAll(suffix, "-", "_")).
			Build()
	}
	identifier := "presign-handler-" + suffix
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	editorID := newUser("editor")
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(editorID).WithRole(model.SpaceRoleEditor).Build()
	viewerID := newUser("viewer")
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(viewerID).WithRole(model.SpaceRoleViewer).Build()

	return presignFixture{
		identifier: identifier,
		spaceID:    spaceID,
		editor:     &model.User{ID: editorID},
		viewer:     &model.User{ID: viewerID},
	}
}

func newHandler(inserter dispatcher.JobInserter, objectStorage storage.ObjectStorage) *attachment_presign.Handler {
	db := testutil.GetTestDB()
	q := query.New(db)
	return attachment_presign.NewHandler(usecase.NewCreateAttachmentPresignUsecase(
		db,
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewAttachmentRepository(q),
		validator.NewAttachmentPresignCreateValidator(),
		dispatcher.NewDispatcher(inserter, nil),
		objectStorage,
	))
}

// servePresignは、閲覧者viewer (nilならログインしていない) としてスペースidentifierへ事前署名を送る
func servePresign(h *attachment_presign.Handler, identifier, body string, viewer *model.User) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Post("/s/{space_identifier}/attachments/presign", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/s/"+identifier+"/attachments/presign", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := i18n.SetLocale(req.Context(), "ja")
	if viewer != nil {
		ctx = middleware.SetUserToContext(ctx, viewer)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req.WithContext(ctx))
	return rr
}

const validPresignBody = `{"filename":"photo.png","content_type":"image/png","byte_size":1234,"checksum":"1B2M2Y8AsgTpgAmY7PhCfg=="}`

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("正常系: attachment:writeを持つメンバーは事前署名できる", func(t *testing.T) {
		t.Parallel()

		f := setupPresignFixture(t, "success")
		inserter := &recordingJobInserter{}
		rr := servePresign(newHandler(inserter, storage.NewFakeObjectStorage()), f.identifier, validPresignBody, f.editor)

		if rr.Code != http.StatusOK {
			t.Fatalf("ステータスコード = %d、期待値 = %d (本文 = %s)", rr.Code, http.StatusOK, rr.Body.String())
		}
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Errorf("Content-Type = %q、期待値 = %q", got, "application/json; charset=utf-8")
		}

		var body map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("応答のJSONのパースに失敗: %v", err)
		}
		for _, key := range []string{"directUploadUrl", "directUploadHeaders", "attachmentId"} {
			if _, ok := body[key]; !ok {
				t.Errorf("応答に%sが無い (本文 = %s)", key, rr.Body.String())
			}
		}
		if _, ok := body["blobSignedId"]; ok {
			t.Error("応答にblobSignedIdが含まれている")
		}
		if url, _ := body["directUploadUrl"].(string); !strings.HasPrefix(url, "https://storage.example.com/") {
			t.Errorf("directUploadUrl = %q、期待値 = オブジェクトストレージのURL", url)
		}
		headers, _ := body["directUploadHeaders"].(map[string]any)
		if headers["Content-Type"] != "image/png" || headers["Content-MD5"] != "1B2M2Y8AsgTpgAmY7PhCfg==" {
			t.Errorf("directUploadHeaders = %v、期待値 = 種類とチェックサムを含む", headers)
		}
		if inserter.calls != 1 {
			t.Errorf("ジョブの投入の回数 = %d、期待値 = 1", inserter.calls)
		}
	})

	t.Run("異常系: ログインしていなければ401", func(t *testing.T) {
		t.Parallel()

		f := setupPresignFixture(t, "unauthenticated")
		rr := servePresign(newHandler(&recordingJobInserter{}, storage.NewFakeObjectStorage()), f.identifier, validPresignBody, nil)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("異常系: スペースが無ければ404", func(t *testing.T) {
		t.Parallel()

		f := setupPresignFixture(t, "no-space")
		rr := servePresign(newHandler(&recordingJobInserter{}, storage.NewFakeObjectStorage()), "presign-handler-missing", validPresignBody, f.editor)
		if rr.Code != http.StatusNotFound {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("異常系: attachment:writeを持たなければ403", func(t *testing.T) {
		t.Parallel()

		f := setupPresignFixture(t, "forbidden")
		inserter := &recordingJobInserter{}
		rr := servePresign(newHandler(inserter, storage.NewFakeObjectStorage()), f.identifier, validPresignBody, f.viewer)
		if rr.Code != http.StatusForbidden {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusForbidden)
		}
		if inserter.calls != 0 {
			t.Errorf("ジョブの投入の回数 = %d、期待値 = 0", inserter.calls)
		}
	})

	t.Run("異常系: 検証に失敗したら422でメッセージの配列を返す", func(t *testing.T) {
		t.Parallel()

		f := setupPresignFixture(t, "invalid")
		body := `{"filename":"","content_type":"application/x-msdownload","byte_size":0,"checksum":""}`
		rr := servePresign(newHandler(&recordingJobInserter{}, storage.NewFakeObjectStorage()), f.identifier, body, f.editor)
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
		}
		var got struct {
			Errors []string `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("応答のJSONのパースに失敗: %v", err)
		}
		want := []string{
			"ファイル名を入力してください",
			"ファイル形式はサポートされていない形式です",
			"ファイルサイズは0より大きい値にしてください",
			"チェックサムを入力してください",
		}
		if strings.Join(got.Errors, "\n") != strings.Join(want, "\n") {
			t.Errorf("errors = %q、期待値 = %q", got.Errors, want)
		}
	})

	t.Run("異常系: チェックサムに改行があれば422で行とジョブを作らない", func(t *testing.T) {
		t.Parallel()

		f := setupPresignFixture(t, "checksum-newline")
		inserter := &recordingJobInserter{}
		body := `{"filename":"photo.png","content_type":"image/png","byte_size":1234,"checksum":"1B2M\r\n2Y8AsgTpgAmY7PhCfg=="}`
		rr := servePresign(newHandler(inserter, storage.NewFakeObjectStorage()), f.identifier, body, f.editor)
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("ステータスコード = %d、期待値 = %d (本文 = %s)", rr.Code, http.StatusUnprocessableEntity, rr.Body.String())
		}
		var got struct {
			Errors []string `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("応答のJSONのパースに失敗: %v", err)
		}
		if len(got.Errors) != 1 || got.Errors[0] != "チェックサムの形式が正しくありません" {
			t.Errorf("errors = %q、期待値 = [チェックサムの形式が正しくありません]", got.Errors)
		}
		if inserter.calls != 0 {
			t.Errorf("ジョブの投入の回数 = %d、期待値 = 0", inserter.calls)
		}

		db := testutil.GetTestDB()
		for _, tc := range []struct {
			name  string
			query string
		}{
			{name: "blob", query: `SELECT count(*) FROM active_storage_blobs WHERE metadata::jsonb ->> 'space_id' = $1`},
			{name: "blobの関連付け", query: `SELECT count(*) FROM active_storage_attachments WHERE record_type = 'SpaceRecord' AND record_id = $1`},
			{name: "添付ファイル", query: `SELECT count(*) FROM attachments WHERE space_id = $1`},
		} {
			var count int
			if err := db.QueryRowContext(context.Background(), tc.query, string(f.spaceID)).Scan(&count); err != nil {
				t.Fatalf("%sの行の数の取得に失敗: %v", tc.name, err)
			}
			if count != 0 {
				t.Errorf("%sの行の数 = %d、期待値 = 0", tc.name, count)
			}
		}
	})

	t.Run("異常系: JSONとして読めなければ400", func(t *testing.T) {
		t.Parallel()

		f := setupPresignFixture(t, "bad-json")
		rr := servePresign(newHandler(&recordingJobInserter{}, storage.NewFakeObjectStorage()), f.identifier, `{"filename":`, f.editor)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("異常系: オブジェクトストレージが設定されていなければ500", func(t *testing.T) {
		t.Parallel()

		f := setupPresignFixture(t, "no-storage")
		rr := servePresign(newHandler(&recordingJobInserter{}, nil), f.identifier, validPresignBody, f.editor)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusInternalServerError)
		}
	})
}
