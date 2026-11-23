package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// deleteAttachmentFixtureは、添付ファイルの削除のテストが使うスペースと管理者。UseCaseは自身で
// トランザクションを開くため、フィクスチャはDBに直接コミットする
type deleteAttachmentFixture struct {
	db            *sql.DB
	suffix        string
	identifier    model.SpaceIdentifier
	spaceID       model.SpaceID
	userID        model.UserID
	memberID      model.SpaceMemberID
	topicID       model.TopicID
	objectStorage *storage.FakeObjectStorage
	jobClient     *river.Client[*sql.Tx]
	jobSchema     string
	dispatcher    *dispatcher.Dispatcher
}

func setupDeleteAttachmentFixture(t *testing.T, suffix string) *deleteAttachmentFixture {
	t.Helper()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-attachment-" + suffix + "@example.com").
		WithAtname("delete_attachment_" + strings.ReplaceAll(suffix, "-", "_")).
		Build()
	identifier := "del-att-" + suffix
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	memberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).WithRole(model.SpaceRoleAdmin).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).Build()
	jobClient, jobSchema := testutil.SetupRiverClient(t, db)

	return &deleteAttachmentFixture{
		db:            db,
		suffix:        suffix,
		identifier:    model.SpaceIdentifier(identifier),
		spaceID:       spaceID,
		userID:        userID,
		memberID:      memberID,
		topicID:       topicID,
		objectStorage: storage.NewFakeObjectStorage(),
		jobClient:     jobClient,
		jobSchema:     jobSchema,
		dispatcher:    dispatcher.NewDispatcher(jobClient, jobClient),
	}
}

// usecaseは、フィクスチャのオブジェクトストレージを使うUseCaseを組み立てる
func (f *deleteAttachmentFixture) usecase() *DeleteAttachmentUsecase {
	return f.usecaseWith(f.objectStorage)
}

func (f *deleteAttachmentFixture) usecaseWith(objectStorage storage.ObjectStorage) *DeleteAttachmentUsecase {
	queries := query.New(f.db)
	return NewDeleteAttachmentUsecase(
		f.db,
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewAttachmentRepository(queries),
		objectStorage,
		f.dispatcher,
	)
}

func (f *deleteAttachmentFixture) execute(t *testing.T, id model.AttachmentID) error {
	t.Helper()
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	return f.usecase().Execute(ctx, DeleteAttachmentInput{SpaceIdentifier: f.identifier, UserID: f.userID, AttachmentID: id})
}

// addは、添付ファイルの行を作り、実体を置く。添付ファイルのIDとblobのキーを返す
func (f *deleteAttachmentFixture) add(t *testing.T, name string) (model.AttachmentID, string) {
	t.Helper()

	key := "del-att-" + f.suffix + "-" + name
	id := testutil.NewAttachmentBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.memberID).
		WithContentType("image/png").
		WithBlobKey(key).
		Build()
	f.objectStorage.Put(key, []byte("original"), "image/png")
	return id, key
}

// addVariantは、Rails期に作られたのと同じ形のサムネイル (variant) の行を作り、実体を置く。variantのblobの
// キーを返す
func (f *deleteAttachmentFixture) addVariant(t *testing.T, id model.AttachmentID, name string) string {
	t.Helper()

	ctx := context.Background()
	var blobID string
	if err := f.db.QueryRowContext(ctx, `
		SELECT asa.blob_id FROM attachments a
		INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
		WHERE a.id = $1 AND a.space_id = $2`, string(id), string(f.spaceID)).Scan(&blobID); err != nil {
		t.Fatalf("添付ファイルのblobの取得に失敗: %v", err)
	}

	key := "del-att-" + f.suffix + "-" + name + "-variant"
	var variantRecordID, variantBlobID string
	if err := f.db.QueryRowContext(ctx, `
		INSERT INTO active_storage_variant_records (blob_id, variation_digest) VALUES ($1, $2) RETURNING id`,
		blobID, "digest-"+name).Scan(&variantRecordID); err != nil {
		t.Fatalf("variantの行の作成に失敗: %v", err)
	}
	if err := f.db.QueryRowContext(ctx, `
		INSERT INTO active_storage_blobs (key, filename, content_type, metadata, service_name, byte_size, checksum, created_at)
		VALUES ($1, 'thumb.png', 'image/png', '{}', 'test', 1, 'x', NOW()) RETURNING id`, key).Scan(&variantBlobID); err != nil {
		t.Fatalf("variantのblobの作成に失敗: %v", err)
	}
	if _, err := f.db.ExecContext(ctx, `
		INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		VALUES ('image', 'ActiveStorage::VariantRecord', $1, $2, NOW())`, variantRecordID, variantBlobID); err != nil {
		t.Fatalf("variantのattachmentの作成に失敗: %v", err)
	}
	f.objectStorage.Put(key, []byte("thumbnail"), "image/png")
	return key
}

// addReferenceは、ページから添付ファイルへの参照を作る
func (f *deleteAttachmentFixture) addReference(t *testing.T, id model.AttachmentID, pageID model.PageID) {
	t.Helper()
	if _, err := f.db.ExecContext(context.Background(), `
		INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`,
		string(id), string(pageID)); err != nil {
		t.Fatalf("ページからの参照の作成に失敗: %v", err)
	}
}

// countは、添付ファイルに関わる行の数を返す。添付ファイル・Active Storageの行・ページからの参照・variantの行である
func (f *deleteAttachmentFixture) count(t *testing.T, id model.AttachmentID, blobKey string) map[string]int {
	t.Helper()

	counts := map[string]int{}
	queries := map[string]struct {
		sql string
		arg string
	}{
		"attachments":                    {`SELECT count(*) FROM attachments WHERE id = $1`, string(id)},
		"page_attachment_references":     {`SELECT count(*) FROM page_attachment_references WHERE attachment_id = $1`, string(id)},
		"active_storage_blobs":           {`SELECT count(*) FROM active_storage_blobs WHERE key IN ($1::text, $1::text || '-variant')`, blobKey},
		"active_storage_attachments":     {`SELECT count(*) FROM active_storage_attachments asa INNER JOIN active_storage_blobs b ON b.id = asa.blob_id WHERE b.key IN ($1::text, $1::text || '-variant')`, blobKey},
		"active_storage_variant_records": {`SELECT count(*) FROM active_storage_variant_records vr INNER JOIN active_storage_blobs b ON b.id = vr.blob_id WHERE b.key = $1`, blobKey},
	}
	for name, q := range queries {
		var n int
		if err := f.db.QueryRowContext(context.Background(), q.sql, q.arg).Scan(&n); err != nil {
			t.Fatalf("%sの数の取得に失敗: %v", name, err)
		}
		counts[name] = n
	}
	return counts
}

func (f *deleteAttachmentFixture) assertRowsRemain(t *testing.T, id model.AttachmentID, blobKey string, want map[string]int) {
	t.Helper()
	got := f.count(t, id, blobKey)
	for name, n := range want {
		if got[name] != n {
			t.Errorf("%sの行の数 = %d、期待値 = %d", name, got[name], n)
		}
	}
}

func (f *deleteAttachmentFixture) assertNoRows(t *testing.T, id model.AttachmentID, blobKey string) {
	t.Helper()
	for name, n := range f.count(t, id, blobKey) {
		if n != 0 {
			t.Errorf("%sの行が%d件残っている", name, n)
		}
	}
}

func (f *deleteAttachmentFixture) hasObject(key string) bool {
	_, _, ok := f.objectStorage.Object(key)
	return ok
}

func (f *deleteAttachmentFixture) featuredImage(t *testing.T, table string, id string) sql.NullString {
	t.Helper()
	var featured sql.NullString
	if err := f.db.QueryRowContext(context.Background(),
		`SELECT featured_image_attachment_id FROM `+table+` WHERE id = $1 AND space_id = $2`, id, string(f.spaceID)).Scan(&featured); err != nil {
		t.Fatalf("%sのカバー画像の取得に失敗: %v", table, err)
	}
	return featured
}

func TestDeleteAttachmentUsecase_Execute(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "success")
	id, key := f.add(t, "target")
	variantKey := f.addVariant(t, id, "target")
	otherID, otherKey := f.add(t, "other")
	otherVariantKey := f.addVariant(t, otherID, "other")

	pageID := testutil.NewPageBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithTopicID(f.topicID).WithFeaturedImageAttachmentID(id).Build()
	f.addReference(t, id, pageID)
	f.addReference(t, otherID, pageID)
	draftPageID := testutil.NewDraftPageBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).WithTopicID(f.topicID).WithPageID(pageID).WithSpaceMemberID(f.memberID).
		WithFeaturedImageAttachmentID(id).Build()
	suggestionID := testutil.NewSuggestionBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithTopicID(f.topicID).WithCreatedSpaceMemberID(f.memberID).Build()
	suggestionPageID := testutil.NewSuggestionPageBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).WithSuggestionID(suggestionID).WithPageID(pageID).
		WithFeaturedImageAttachmentID(id).Build()

	if err := f.execute(t, id); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	f.assertNoRows(t, id, key)
	for _, k := range []string{key, variantKey} {
		if f.hasObject(k) {
			t.Errorf("実体%qが残っている", k)
		}
	}

	// カバー画像にしていたページ・下書き・編集提案のページからは、カバー画像が外れる
	for table, rowID := range map[string]string{
		"pages":            string(pageID),
		"draft_pages":      string(draftPageID),
		"suggestion_pages": string(suggestionPageID),
	} {
		if featured := f.featuredImage(t, table, rowID); featured.Valid {
			t.Errorf("%sのカバー画像 = %q、NULLを期待", table, featured.String)
		}
	}

	// ほかの添付ファイルの行と実体は変わらない
	f.assertRowsRemain(t, otherID, otherKey, map[string]int{
		"attachments":                    1,
		"page_attachment_references":     1,
		"active_storage_blobs":           2,
		"active_storage_attachments":     2,
		"active_storage_variant_records": 1,
	})
	for _, k := range []string{otherKey, otherVariantKey} {
		if !f.hasObject(k) {
			t.Errorf("ほかの添付ファイルの実体%qが消えた", k)
		}
	}
}

func TestDeleteAttachmentUsecase_Execute_見つからない(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "notfound")
	other := setupDeleteAttachmentFixture(t, "notfound-other")
	otherID, otherKey := other.add(t, "other")

	tests := []struct {
		name string
		id   model.AttachmentID
	}{
		{name: "ほかのスペースの添付ファイル", id: otherID},
		{name: "存在しないID", id: "00000000-0000-0000-0000-000000000000"},
		{name: "不正な形のID", id: "not-a-uuid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f.execute(t, tt.id)
			if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
				t.Errorf("エラー = %v、見つからないことを表すAppErrorを期待", err)
			}
		})
	}

	other.assertRowsRemain(t, otherID, otherKey, map[string]int{"attachments": 1, "active_storage_blobs": 1})
}

func TestDeleteAttachmentUsecase_Execute_権限が無い(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "forbidden")
	id, key := f.add(t, "target")
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	editorID := testutil.NewUserBuilderDB(t, f.db).WithEmail("delete-attachment-editor@example.com").WithAtname("delete_attachment_editor").Build()
	testutil.NewSpaceMemberBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithUserID(editorID).WithRole(model.SpaceRoleEditor).Build()
	outsiderID := testutil.NewUserBuilderDB(t, f.db).WithEmail("delete-attachment-outsider@example.com").WithAtname("delete_attachment_outsider").Build()

	tests := []struct {
		name     string
		userID   model.UserID
		wantCode model.AppErrorCode
	}{
		{name: "編集者は削除できない", userID: editorID, wantCode: model.AppErrCodeForbidden},
		{name: "メンバーでないユーザーには見つからない", userID: outsiderID, wantCode: model.AppErrCodeResourceNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f.usecase().Execute(ctx, DeleteAttachmentInput{SpaceIdentifier: f.identifier, UserID: tt.userID, AttachmentID: id})
			if ae := model.AsAppError(err); ae == nil || ae.Code != tt.wantCode {
				t.Errorf("エラー = %v、コード%vのAppErrorを期待", err, tt.wantCode)
			}
		})
	}

	f.assertRowsRemain(t, id, key, map[string]int{"attachments": 1, "active_storage_blobs": 1})
	if !f.hasObject(key) {
		t.Error("権限の無い削除で実体が消えた")
	}
}

func TestDeleteAttachmentUsecase_Execute_実体の削除に失敗したら行が残る(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "storage-error")
	id, key := f.add(t, "target")
	f.objectStorage.DeleteErr = errors.New("ストレージの障害")

	if err := f.execute(t, id); err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}

	f.assertRowsRemain(t, id, key, map[string]int{"attachments": 1, "active_storage_attachments": 1, "active_storage_blobs": 1})
}

func TestDeleteAttachmentUsecase_Execute_オブジェクトストレージが無ければ行が残る(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "no-storage")
	id, key := f.add(t, "target")
	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	err := f.usecaseWith(nil).Execute(ctx, DeleteAttachmentInput{SpaceIdentifier: f.identifier, UserID: f.userID, AttachmentID: id})
	if !errors.Is(err, errAttachmentObjectStorageNotConfigured) {
		t.Fatalf("エラー = %v、オブジェクトストレージが設定されていないエラーを期待", err)
	}

	f.assertRowsRemain(t, id, key, map[string]int{"attachments": 1, "active_storage_blobs": 1})
}

func TestDeleteAttachmentUsecase_Execute_行の削除が途中で失敗したら1つも消えない(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "rows-error")
	id, key := f.add(t, "target")
	f.addVariant(t, id, "target")
	pageID := testutil.NewPageBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithTopicID(f.topicID).Build()
	f.addReference(t, id, pageID)

	// 添付ファイルのblobを別のレコードからも参照させ、2つ目の文 (blobの削除) を外部キーで失敗させる。
	// 1つ目の文で消したvariantの行も、同じトランザクションのため元に戻る
	if _, err := f.db.ExecContext(context.Background(), `
		INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		SELECT 'file', 'OtherRecord', gen_random_uuid(), b.id, NOW() FROM active_storage_blobs b WHERE b.key = $1`, key); err != nil {
		t.Fatalf("blobを参照する行の作成に失敗: %v", err)
	}

	if err := f.execute(t, id); err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}

	f.assertRowsRemain(t, id, key, map[string]int{
		"attachments":                    1,
		"page_attachment_references":     1,
		"active_storage_blobs":           2,
		"active_storage_attachments":     3,
		"active_storage_variant_records": 1,
	})
}

func TestDeleteAttachmentUsecase_Execute_実体が既に無くても削除できる(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "no-object")
	id, key := f.add(t, "target")
	variantKey := f.addVariant(t, id, "target")
	if err := f.objectStorage.Delete(context.Background(), key); err != nil {
		t.Fatalf("実体の削除に失敗: %v", err)
	}
	if err := f.objectStorage.Delete(context.Background(), variantKey); err != nil {
		t.Fatalf("variantの実体の削除に失敗: %v", err)
	}

	if err := f.execute(t, id); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	f.assertNoRows(t, id, key)
}

func TestDeleteAttachmentUsecase_Execute_2回実行しても結果が変わらない(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "twice")
	id, key := f.add(t, "target")
	f.addVariant(t, id, "target")
	otherID, otherKey := f.add(t, "other")

	if err := f.execute(t, id); err != nil {
		t.Fatalf("1回目のExecute()のエラー = %v", err)
	}
	// 2回目は消す添付ファイルが無いため「見つからない」になり、ほかの添付ファイルは変わらない
	if ae := model.AsAppError(f.execute(t, id)); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("2回目のエラー = %v、見つからないことを表すAppErrorを期待", ae)
	}

	// 共通の削除処理を同じIDで呼び直しても、エラーにならず何も消さない
	deleter := &attachmentDeleter{db: f.db, attachmentRepo: repository.NewAttachmentRepository(query.New(f.db)), objectStorage: f.objectStorage, dispatcher: f.dispatcher}
	if err := deleter.delete(context.Background(), f.spaceID, []model.AttachmentID{id}); err != nil {
		t.Fatalf("delete()のエラー = %v", err)
	}

	f.assertNoRows(t, id, key)
	f.assertRowsRemain(t, otherID, otherKey, map[string]int{"attachments": 1, "active_storage_blobs": 1})
	if !f.hasObject(otherKey) {
		t.Error("ほかの添付ファイルの実体が消えた")
	}
}

// writeBackObjectStorageは、指定した時点で、アップロード後の処理が書き戻したのと同じ形で実体を置き直す。
// 削除とアップロード後の処理が重なったときの順序を再現する
type writeBackObjectStorage struct {
	*storage.FakeObjectStorage
	onDelete func(key string)
	onUpload func()
}

func (s *writeBackObjectStorage) Delete(ctx context.Context, key string) error {
	if err := s.FakeObjectStorage.Delete(ctx, key); err != nil {
		return err
	}
	if s.onDelete != nil {
		s.onDelete(key)
	}
	return nil
}

func (s *writeBackObjectStorage) Upload(ctx context.Context, input storage.UploadInput) error {
	if s.onUpload != nil {
		s.onUpload()
	}
	return s.FakeObjectStorage.Upload(ctx, input)
}

// TestDeleteAttachmentUsecase_Execute_行の削除の前に書き戻された実体を残さないは、削除が実体を消してから
// 行を消すまでの間に、アップロード後の処理が実体を書き戻した順序を再現する
func TestDeleteAttachmentUsecase_Execute_行の削除の前に書き戻された実体を残さない(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "write-back-before")
	id, key := f.add(t, "target")

	wrote := false
	objectStorage := &writeBackObjectStorage{FakeObjectStorage: f.objectStorage}
	objectStorage.onDelete = func(deleted string) {
		if deleted == key && !wrote {
			wrote = true
			f.objectStorage.Put(key, []byte("processed"), "image/png")
		}
	}

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	if err := f.usecaseWith(objectStorage).Execute(ctx, DeleteAttachmentInput{SpaceIdentifier: f.identifier, UserID: f.userID, AttachmentID: id}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if !wrote {
		t.Fatal("書き戻しを再現できていない")
	}
	f.cleanup(t)
	if f.hasObject(key) {
		t.Error("行の削除の前に書き戻された実体が残っている")
	}
	f.assertNoRows(t, id, key)
}

// TestProcessAttachmentUsecase_DeletesObjectWrittenBackAfterDeletionは、削除が実体と行を消したあとに、
// アップロード後の処理が実体を書き戻した順序を再現する
func TestProcessAttachmentUsecase_DeletesObjectWrittenBackAfterDeletion(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "write-back-after")
	key := "del-att-" + f.suffix + "-svg"
	id := testutil.NewAttachmentBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.memberID).
		WithContentType("image/svg+xml").
		WithBlobKey(key).
		WithAttachedAt(time.Now()).
		WithProcessingStatus(model.AttachmentProcessingStatusPending).
		Build()
	f.objectStorage.Put(key, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), "image/svg+xml")

	queries := query.New(f.db)
	attachmentRepo := repository.NewAttachmentRepository(queries)
	objectStorage := &writeBackObjectStorage{FakeObjectStorage: f.objectStorage}
	deleted := false
	objectStorage.onUpload = func() {
		// 書き戻す直前に削除が終わる。書き戻した実体は、どの行からも参照されない
		deleter := &attachmentDeleter{db: f.db, attachmentRepo: attachmentRepo, objectStorage: f.objectStorage, dispatcher: f.dispatcher}
		if err := deleter.delete(context.Background(), f.spaceID, []model.AttachmentID{id}); err != nil {
			t.Errorf("delete()のエラー = %v", err)
		}
		deleted = true
	}

	uc := NewProcessAttachmentUsecase(f.db, attachmentRepo, objectStorage)
	if _, err := uc.Execute(context.Background(), ProcessAttachmentInput{AttachmentID: id, SpaceID: f.spaceID}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if !deleted {
		t.Fatal("書き戻しの前の削除を再現できていない")
	}
	if f.hasObject(key) {
		t.Error("削除のあとに書き戻された実体が残っている")
	}
	f.assertNoRows(t, id, key)
}

// TestProcessAttachmentUsecase_KeepsObjectWhenAnotherAttemptCompletedは、ほかの試行が先に完了にしていた
// (行は残っている) ときに、実体を消さないことを確かめる
func TestProcessAttachmentUsecase_KeepsObjectWhenAnotherAttemptCompleted(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "another-attempt")
	key := "del-att-" + f.suffix + "-svg"
	id := testutil.NewAttachmentBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).
		WithSpaceMemberID(f.memberID).
		WithContentType("image/svg+xml").
		WithBlobKey(key).
		WithAttachedAt(time.Now()).
		WithProcessingStatus(model.AttachmentProcessingStatusPending).
		Build()
	f.objectStorage.Put(key, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), "image/svg+xml")

	attachmentRepo := repository.NewAttachmentRepository(query.New(f.db))
	objectStorage := &writeBackObjectStorage{FakeObjectStorage: f.objectStorage}
	objectStorage.onUpload = func() {
		if _, err := f.db.ExecContext(context.Background(),
			`UPDATE attachments SET processing_status = $1 WHERE id = $2 AND space_id = $3`,
			int(model.AttachmentProcessingStatusCompleted), string(id), string(f.spaceID)); err != nil {
			t.Errorf("状態の更新に失敗: %v", err)
		}
	}

	uc := NewProcessAttachmentUsecase(f.db, attachmentRepo, objectStorage)
	if _, err := uc.Execute(context.Background(), ProcessAttachmentInput{AttachmentID: id, SpaceID: f.spaceID}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if !f.hasObject(key) {
		t.Error("行が残っている添付ファイルの実体が消えた")
	}
}
