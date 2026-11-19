package usecase

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// presignChecksumは空の内容のMD5をBase64にした値
const presignChecksum = "1B2M2Y8AsgTpgAmY7PhCfg=="

// createAttachmentPresignFixtureは事前署名のテストに使うスペースとメンバー
type createAttachmentPresignFixture struct {
	db            *sql.DB
	identifier    model.SpaceIdentifier
	spaceID       model.SpaceID
	editorID      model.UserID
	editorMember  model.SpaceMemberID
	viewerID      model.UserID
	outsiderID    model.UserID
	objectStorage *storage.FakeObjectStorage
}

func setupCreateAttachmentPresignFixture(t *testing.T, suffix string) *createAttachmentPresignFixture {
	t.Helper()

	db := testutil.GetTestDB()
	newUser := func(role string) model.UserID {
		return testutil.NewUserBuilderDB(t, db).
			WithEmail("presign-" + role + "-" + suffix + "@example.com").
			WithAtname("presign_" + role + "_" + strings.ReplaceAll(suffix, "-", "_")).
			Build()
	}
	identifier := "presign-" + suffix
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	editorID := newUser("editor")
	editorMember := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(editorID).WithRole(model.SpaceRoleEditor).Build()
	viewerID := newUser("viewer")
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(viewerID).WithRole(model.SpaceRoleViewer).Build()

	return &createAttachmentPresignFixture{
		db:            db,
		identifier:    model.SpaceIdentifier(identifier),
		spaceID:       spaceID,
		editorID:      editorID,
		editorMember:  editorMember,
		viewerID:      viewerID,
		outsiderID:    newUser("outsider"),
		objectStorage: storage.NewFakeObjectStorage(),
	}
}

func (f *createAttachmentPresignFixture) usecase(inserter dispatcher.JobInserter, objectStorage storage.ObjectStorage) *CreateAttachmentPresignUsecase {
	q := query.New(f.db)
	return NewCreateAttachmentPresignUsecase(
		f.db,
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewAttachmentRepository(q),
		validator.NewAttachmentPresignCreateValidator(),
		dispatcher.NewDispatcher(inserter),
		objectStorage,
	)
}

func (f *createAttachmentPresignFixture) input(userID model.UserID) CreateAttachmentPresignInput {
	byteSize := int64(1234)
	return CreateAttachmentPresignInput{
		SpaceIdentifier: f.identifier,
		UserID:          userID,
		Filename:        "../写真.png",
		ContentType:     "image/png",
		ByteSize:        &byteSize,
		Checksum:        presignChecksum,
	}
}

// attachmentCountは、スペースの添付ファイルの行の数を返す
func (f *createAttachmentPresignFixture) attachmentCount(t *testing.T) int {
	t.Helper()

	var n int
	if err := f.db.QueryRowContext(context.Background(), `SELECT count(*) FROM attachments WHERE space_id = $1`, string(f.spaceID)).Scan(&n); err != nil {
		t.Fatalf("添付ファイルの数の取得に失敗: %v", err)
	}
	return n
}

func TestCreateAttachmentPresignUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), "ja")

	t.Run("正常系: 行を処理待ちで作り、ジョブを投入して、条件付きのアップロードの署名を返す", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAttachmentPresignFixture(t, "success")
		inserter := &mockJobInserter{}
		before := time.Now().Add(-time.Second)
		out, err := f.usecase(inserter, f.objectStorage).Execute(ctx, f.input(f.editorID))
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		var (
			key, filename, contentType, checksum string
			byteSize                             int64
			memberID                             string
			status                               int32
			attachedAt                           time.Time
		)
		err = f.db.QueryRowContext(ctx, `
			SELECT b.key, b.filename, b.content_type, b.checksum, b.byte_size, a.attached_space_member_id, a.processing_status, a.attached_at
			FROM attachments a
			INNER JOIN active_storage_attachments asa ON asa.id = a.active_storage_attachment_id
			INNER JOIN active_storage_blobs b ON b.id = asa.blob_id
			WHERE a.id = $1 AND a.space_id = $2`, string(out.AttachmentID), string(f.spaceID),
		).Scan(&key, &filename, &contentType, &checksum, &byteSize, &memberID, &status, &attachedAt)
		if err != nil {
			t.Fatalf("作った添付ファイルの取得に失敗: %v", err)
		}
		if filename != "_写真.png" {
			t.Errorf("filename = %q、期待値 = %q (整えたファイル名)", filename, "_写真.png")
		}
		if strings.Contains(key, "写真") || len(key) != 28 {
			t.Errorf("key = %q、期待値 = ファイル名を含まない28文字", key)
		}
		if contentType != "image/png" || checksum != presignChecksum || byteSize != 1234 {
			t.Errorf("blob = (%q, %q, %d)、期待値 = (%q, %q, %d)", contentType, checksum, byteSize, "image/png", presignChecksum, 1234)
		}
		if memberID != string(f.editorMember) {
			t.Errorf("attached_space_member_id = %q、期待値 = %q", memberID, f.editorMember)
		}
		if status != int32(model.AttachmentProcessingStatusPending) {
			t.Errorf("processing_status = %d、期待値 = %d", status, model.AttachmentProcessingStatusPending)
		}
		// 列はタイムゾーンを持たず、UTCの壁時計の時刻を返す
		if attachedAt.Before(before.UTC().Truncate(time.Microsecond)) {
			t.Errorf("attached_at = %v、期待値 = %v以降", attachedAt, before.UTC())
		}

		args, ok := inserter.args.(dispatcher.ProcessAttachmentArgs)
		if !inserter.called || !ok {
			t.Fatalf("投入したジョブ = %#v、期待値 = ProcessAttachmentArgs", inserter.args)
		}
		if args.AttachmentID != string(out.AttachmentID) || args.SpaceID != string(f.spaceID) {
			t.Errorf("ジョブの引数 = %#v、期待値 = (%q, %q)", args, out.AttachmentID, f.spaceID)
		}

		uploadURL, err := url.Parse(out.DirectUploadURL)
		if err != nil {
			t.Fatalf("アップロードのURLのパースに失敗: %v", err)
		}
		if uploadURL.Path != "/"+key {
			t.Errorf("アップロードのURLのパス = %q、期待値 = %q", uploadURL.Path, "/"+key)
		}
		for name, want := range map[string]string{
			"expires_in":     "300",
			"content_type":   "image/png",
			"content_length": "1234",
			"content_md5":    presignChecksum,
		} {
			if got := uploadURL.Query().Get(name); got != want {
				t.Errorf("アップロードの署名の%s = %q、期待値 = %q", name, got, want)
			}
		}

		wantHeaders := map[string]string{
			"Content-Type":        "image/png",
			"Content-MD5":         presignChecksum,
			"Content-Disposition": `inline; filename="_%3F%3F.png"; filename*=UTF-8''_%E5%86%99%E7%9C%9F.png`,
		}
		if len(out.DirectUploadHeaders) != len(wantHeaders) {
			t.Errorf("DirectUploadHeaders = %v、期待値 = %v", out.DirectUploadHeaders, wantHeaders)
		}
		for name, want := range wantHeaders {
			if got := out.DirectUploadHeaders[name]; got != want {
				t.Errorf("DirectUploadHeaders[%q] = %q、期待値 = %q", name, got, want)
			}
		}
	})

	t.Run("異常系: スペースが無ければ未存在", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAttachmentPresignFixture(t, "no-space")
		input := f.input(f.editorID)
		input.SpaceIdentifier = "presign-missing-space"
		_, err := f.usecase(&mockJobInserter{}, f.objectStorage).Execute(ctx, input)
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、期待値 = 未存在のAppError", err)
		}
	})

	for _, tc := range []struct {
		name   string
		suffix string
		userID func(f *createAttachmentPresignFixture) model.UserID
	}{
		{name: "異常系: attachment:writeを持たない閲覧者は権限不足", suffix: "viewer", userID: func(f *createAttachmentPresignFixture) model.UserID { return f.viewerID }},
		{name: "異常系: スペースのメンバーでなければ権限不足", suffix: "outsider", userID: func(f *createAttachmentPresignFixture) model.UserID { return f.outsiderID }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := setupCreateAttachmentPresignFixture(t, "forbidden-"+tc.suffix)
			inserter := &mockJobInserter{}
			_, err := f.usecase(inserter, f.objectStorage).Execute(ctx, f.input(tc.userID(f)))
			if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeForbidden {
				t.Errorf("エラー = %v、期待値 = 権限不足のAppError", err)
			}
			if n := f.attachmentCount(t); n != 0 || inserter.called {
				t.Errorf("添付ファイルの数 = %d、ジョブの投入 = %v、期待値 = 0、false", n, inserter.called)
			}
		})
	}

	t.Run("異常系: 検証に失敗したら行を作らない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAttachmentPresignFixture(t, "invalid")
		input := f.input(f.editorID)
		input.ContentType = "text/html"
		inserter := &mockJobInserter{}
		_, err := f.usecase(inserter, f.objectStorage).Execute(ctx, input)
		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("content_type") {
			t.Errorf("エラー = %v、期待値 = content_typeのValidationError", err)
		}
		if n := f.attachmentCount(t); n != 0 || inserter.called {
			t.Errorf("添付ファイルの数 = %d、ジョブの投入 = %v、期待値 = 0、false", n, inserter.called)
		}
	})

	t.Run("異常系: ジョブを投入できなければ署名付きURLを返さない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAttachmentPresignFixture(t, "enqueue-failure")
		out, err := f.usecase(&failingJobInserter{}, f.objectStorage).Execute(ctx, f.input(f.editorID))
		if err == nil || out != nil {
			t.Fatalf("結果 = %+v、エラー = %v、期待値 = nilとエラー", out, err)
		}
		if model.AsAppError(err) != nil || model.AsValidationError(err) != nil {
			t.Errorf("エラー = %v、期待値 = 素のエラー", err)
		}
		// 行は処理待ちのまま残る。アップロードされないため、配信もされない
		if n := f.attachmentCount(t); n != 1 {
			t.Errorf("添付ファイルの数 = %d、期待値 = 1", n)
		}
	})

	t.Run("異常系: 署名できなければ行を作らない", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAttachmentPresignFixture(t, "presign-failure")
		f.objectStorage.PresignErr = errors.New("署名できません")
		inserter := &mockJobInserter{}
		_, err := f.usecase(inserter, f.objectStorage).Execute(ctx, f.input(f.editorID))
		if err == nil {
			t.Fatal("エラーを期待したが、nilだった")
		}
		if n := f.attachmentCount(t); n != 0 || inserter.called {
			t.Errorf("添付ファイルの数 = %d、ジョブの投入 = %v、期待値 = 0、false", n, inserter.called)
		}
	})

	t.Run("異常系: オブジェクトストレージが設定されていなければエラー", func(t *testing.T) {
		t.Parallel()

		f := setupCreateAttachmentPresignFixture(t, "no-storage")
		inserter := &mockJobInserter{}
		_, err := f.usecase(inserter, nil).Execute(ctx, f.input(f.editorID))
		if err == nil || model.AsAppError(err) != nil {
			t.Errorf("エラー = %v、期待値 = 素のエラー", err)
		}
		if n := f.attachmentCount(t); n != 0 || inserter.called {
			t.Errorf("添付ファイルの数 = %d、ジョブの投入 = %v、期待値 = 0、false", n, inserter.called)
		}
	})
}

// TestCreateAttachmentPresignUsecase_createAttachmentは、3つの行の作成が途中で失敗したら、どの行も残らないことを確かめる
func TestCreateAttachmentPresignUsecase_createAttachment(t *testing.T) {
	t.Parallel()

	f := setupCreateAttachmentPresignFixture(t, "rollback")
	const blobKey = "presignrollbackkey0000000000"
	// 存在しないスペースメンバーを指すため、最後のattachmentsの行の作成が外部キーで失敗する
	_, err := f.usecase(&mockJobInserter{}, f.objectStorage).createAttachment(context.Background(), repository.CreateAttachmentInput{
		SpaceID:       f.spaceID,
		UserID:        f.editorID,
		SpaceMemberID: model.SpaceMemberID("00000000-0000-0000-0000-000000000000"),
		BlobKey:       blobKey,
		Filename:      "a.png",
		ContentType:   "image/png",
		ByteSize:      1,
		Checksum:      presignChecksum,
		AttachedAt:    time.Now(),
	})
	if err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}

	var blobs, storageAttachments int
	if err := f.db.QueryRow(`SELECT count(*) FROM active_storage_blobs WHERE key = $1`, blobKey).Scan(&blobs); err != nil {
		t.Fatalf("blobの数の取得に失敗: %v", err)
	}
	if err := f.db.QueryRow(`SELECT count(*) FROM active_storage_attachments WHERE record_id = $1`, string(f.spaceID)).Scan(&storageAttachments); err != nil {
		t.Fatalf("blobの関連付けの数の取得に失敗: %v", err)
	}
	if blobs != 0 || storageAttachments != 0 || f.attachmentCount(t) != 0 {
		t.Errorf("残った行 = (blob %d, 関連付け %d, 添付ファイル %d)、期待値 = すべて0", blobs, storageAttachments, f.attachmentCount(t))
	}
}
