package usecase

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// spaceRowTablesは、space_idを持ち、スペースの物理削除でスペースとともに消えるべき行を持つ表
var spaceRowTables = []string{
	"attachments",
	"draft_page_revisions",
	"draft_pages",
	"exports",
	"oauth_access_tokens",
	"oauth_applications",
	"oauth_grants",
	"page_editors",
	"page_revisions",
	"pages",
	"personal_access_tokens",
	"space_members",
	"suggestion_comments",
	"suggestion_pages",
	"suggestions",
	"topic_members",
	"topics",
}

// deleteSpaceFixtureは、スペースの物理削除のテストが使うスペース。添付ファイルの削除のテストのフィクスチャ
// (スペース・管理者・トピック・River・フェイクのオブジェクトストレージ) に、消える行と実体を足す
type deleteSpaceFixture struct {
	*deleteAttachmentFixture
	attachmentIDs []model.AttachmentID
	objectKeys    []string
}

// setupDeleteSpaceFixtureは、物理削除が消す行と実体を1つずつ以上持つスペースを作る。
// 添付ファイル (サムネイル・ページからの参照・カバー画像を含む)、論理削除したトピックを含むトピックと
// その関連レコード、編集提案のコメント、エクスポート (Go版のアーカイブとRails版のZIP)、
// 個人用アクセストークン、OAuthの行である
func setupDeleteSpaceFixture(t *testing.T, suffix string) *deleteSpaceFixture {
	t.Helper()

	f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "del-space-"+suffix)}
	ctx := context.Background()

	id, key := f.add(t, "image")
	variantKey := f.addVariant(t, id, "image")
	f.attachmentIDs = append(f.attachmentIDs, id)
	f.objectKeys = append(f.objectKeys, key, variantKey)

	s := deleteTopicSpace{spaceID: f.spaceID, spaceMemberID: f.memberID, attachmentID: id}
	_, pageID := seedTopicWithRelatedRecords(t, f.db, s, 10)
	discardedTopicID, _ := seedTopicWithRelatedRecords(t, f.db, s, 11)
	discardTopicForTest(t, f.db, f.spaceID, discardedTopicID)
	testutil.NewPageBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithTopicID(f.topicID).WithNumber(20).WithFeaturedImageAttachmentID(id).Build()

	var suggestionID string
	if err := f.db.QueryRowContext(ctx, `SELECT suggestion_id FROM suggestion_pages WHERE page_id = $1`, string(pageID)).Scan(&suggestionID); err != nil {
		t.Fatalf("編集提案の取得に失敗: %v", err)
	}
	testutil.NewSuggestionCommentBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).
		WithSuggestionID(model.SuggestionID(suggestionID)).
		WithCreatedSpaceMemberID(f.memberID).
		Build()

	exportKey := "exports/del-space-" + suffix + ".zip"
	testutil.NewExportBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.memberID).
		WithStatus(model.ExportStatusSucceeded).
		WithObjectKey(exportKey).
		Build()
	f.objectStorage.Put(exportKey, []byte("zip"), "application/zip")
	legacyExportID := testutil.NewExportBuilderDB(t, f.db).
		WithSpaceID(f.spaceID).
		WithQueuedByID(f.memberID).
		WithStatus(model.ExportStatusSucceeded).
		Build()
	legacyKey := "del-space-" + suffix + "-legacy-export"
	var legacyBlobID string
	if err := f.db.QueryRowContext(ctx, `
		INSERT INTO active_storage_blobs (key, filename, content_type, service_name, byte_size, created_at)
		VALUES ($1, 'old.zip', 'application/zip', 'local', 3, NOW()) RETURNING id`, legacyKey).Scan(&legacyBlobID); err != nil {
		t.Fatalf("Rails版のZIPのblobの作成に失敗: %v", err)
	}
	if _, err := f.db.ExecContext(ctx, `
		INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		VALUES ('file', 'ExportRecord', $1, $2, NOW())`, string(legacyExportID), legacyBlobID); err != nil {
		t.Fatalf("Rails版のZIPの関連の作成に失敗: %v", err)
	}
	f.objectStorage.Put(legacyKey, []byte("zip"), "application/zip")
	f.objectKeys = append(f.objectKeys, exportKey, legacyKey)

	// 個人用アクセストークンとOAuthのビルダーはトランザクションを受け取るため、作ってからコミットする
	tx, err := f.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	testutil.NewPersonalAccessTokenBuilder(t, tx).WithSpaceID(f.spaceID).WithSpaceMemberID(f.memberID).WithTokenDigest("del-space-" + suffix).Build()
	applicationID := testutil.NewOAuthApplicationBuilder(t, tx).WithSpaceID(f.spaceID).WithCreatedSpaceMemberID(f.memberID).WithClientID("del-space-" + suffix).Build()
	grantID := testutil.NewOAuthGrantBuilder(t, tx).WithOAuthApplicationID(applicationID).WithSpaceID(f.spaceID).WithSpaceMemberID(f.memberID).Build()
	testutil.NewOAuthAccessTokenBuilder(t, tx).WithOAuthGrantID(grantID).WithSpaceID(f.spaceID).WithTokenDigest("del-space-" + suffix).Build()
	if err := tx.Commit(); err != nil {
		t.Fatalf("トランザクションのコミットに失敗: %v", err)
	}

	return f
}

// discardは、スペースを論理削除する (スペースの削除の画面が行う論理削除の代わり)
func (f *deleteSpaceFixture) discard(t *testing.T) {
	t.Helper()
	if _, err := f.db.ExecContext(context.Background(), `UPDATE spaces SET discarded_at = NOW() WHERE id = $1`, string(f.spaceID)); err != nil {
		t.Fatalf("スペースの論理削除に失敗: %v", err)
	}
}

// deleteSpaceUsecaseは、objectStorageを使うスペースの物理削除のUseCaseを組み立てる
func (f *deleteSpaceFixture) deleteSpaceUsecase(objectStorage storage.ObjectStorage) *DeleteSpaceUsecase {
	queries := query.New(f.db)
	return NewDeleteSpaceUsecase(f.db, DeleteSpaceRepos{
		SpaceRepo:                   repository.NewSpaceRepository(queries),
		SpaceMemberRepo:             repository.NewSpaceMemberRepository(queries),
		TopicRepo:                   repository.NewTopicRepository(queries),
		TopicMemberRepo:             repository.NewTopicMemberRepository(queries),
		PageRepo:                    repository.NewPageRepository(queries),
		DraftPageRepo:               repository.NewDraftPageRepository(queries),
		PageEditorRepo:              repository.NewPageEditorRepository(queries),
		PageRevisionRepo:            repository.NewPageRevisionRepository(queries),
		PageAttachmentReferenceRepo: repository.NewPageAttachmentReferenceRepository(queries),
		AttachmentRepo:              repository.NewAttachmentRepository(queries),
		ExportRepo:                  repository.NewExportRepository(queries),
	}, objectStorage, f.dispatcher)
}

func (f *deleteSpaceFixture) deleteSpace(t *testing.T) error {
	t.Helper()
	return f.deleteSpaceUsecase(f.objectStorage).Execute(t.Context(), DeleteSpaceInput{SpaceID: f.spaceID})
}

// countRowsは、スペースの行と、スペースに属する表ごとの行の数を返す。space_idを持たない表は、添付ファイルと
// エクスポートのキーからたどる
func (f *deleteSpaceFixture) countRows(t *testing.T) map[string]int {
	t.Helper()

	counts := map[string]int{}
	var n int
	if err := f.db.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM spaces WHERE id = $1`, string(f.spaceID)).Scan(&n); err != nil {
		t.Fatalf("spacesの行の数の取得に失敗: %v", err)
	}
	counts["spaces"] = n
	for _, table := range spaceRowTables {
		counts[table] = countRows(t, f.db, table, f.spaceID, "")
	}
	ids := make([]string, len(f.attachmentIDs))
	for i, id := range f.attachmentIDs {
		ids[i] = string(id)
	}
	for name, q := range map[string]struct {
		sql string
		arg []string
	}{
		"page_attachment_references":     {`SELECT COUNT(*) FROM page_attachment_references WHERE attachment_id::text = ANY($1::text[])`, ids},
		"active_storage_blobs":           {`SELECT COUNT(*) FROM active_storage_blobs WHERE key = ANY($1::text[])`, f.objectKeys},
		"active_storage_attachments":     {`SELECT COUNT(*) FROM active_storage_attachments a INNER JOIN active_storage_blobs b ON b.id = a.blob_id WHERE b.key = ANY($1::text[])`, f.objectKeys},
		"active_storage_variant_records": {`SELECT COUNT(*) FROM active_storage_variant_records vr INNER JOIN active_storage_blobs b ON b.id = vr.blob_id WHERE b.key = ANY($1::text[])`, f.objectKeys},
	} {
		if err := f.db.QueryRowContext(t.Context(), q.sql, pq.Array(q.arg)).Scan(&n); err != nil {
			t.Fatalf("%sの行の数の取得に失敗: %v", name, err)
		}
		counts[name] = n
	}
	return counts
}

// assertNoRowsは、スペースに属する行がすべて消えていることを確かめる
func (f *deleteSpaceFixture) assertNoRows(t *testing.T) {
	t.Helper()
	for table, n := range f.countRows(t) {
		if n != 0 {
			t.Errorf("%sの行 = %d件、期待値 = 0件", table, n)
		}
	}
}

// assertRowsRemainは、スペースに属する行がどの表にも残っていることを確かめる
func (f *deleteSpaceFixture) assertRowsRemain(t *testing.T) {
	t.Helper()
	for table, n := range f.countRows(t) {
		if n == 0 {
			t.Errorf("%sの行が消えた、残ることを期待", table)
		}
	}
}

// remainingObjectsは、スペースの実体のうち、オブジェクトストレージに残っているキーを返す
func (f *deleteSpaceFixture) remainingObjects() []string {
	var keys []string
	for _, key := range f.objectKeys {
		if f.hasObject(key) {
			keys = append(keys, key)
		}
	}
	return keys
}

func TestDeleteSpaceUsecase_Execute(t *testing.T) {
	t.Parallel()

	f := setupDeleteSpaceFixture(t, "main")
	other := setupDeleteSpaceFixture(t, "main-other")
	f.discard(t)

	if err := f.deleteSpace(t); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	f.cleanupExports(t)
	f.assertNoRows(t)
	if keys := f.remainingObjects(); len(keys) != 0 {
		t.Errorf("残った実体 = %v、期待値 = なし", keys)
	}

	// 後始末ジョブは、行の削除で受け取った添付ファイルとサムネイルのキーを持つ
	jobs := f.cleanupJobs(t)
	if len(jobs) != 1 {
		t.Fatalf("後始末ジョブの数 = %d、期待値 = 1", len(jobs))
	}
	input := cleanupInput(t, jobs[0])
	keys := slices.Clone(input.Keys)
	slices.Sort(keys)
	wantKeys := slices.Clone(f.objectKeys[:2])
	slices.Sort(wantKeys)
	if input.SpaceID != string(f.spaceID) || !slices.Equal(keys, wantKeys) {
		t.Errorf("後始末ジョブの引数 = %+v、期待値 = スペース %s とキー %v", input, f.spaceID, wantKeys)
	}

	// ほかのスペースの行と実体は変わらない
	other.assertRowsRemain(t)
	if keys := other.remainingObjects(); len(keys) != len(other.objectKeys) {
		t.Errorf("ほかのスペースの残った実体 = %v、期待値 = %v", keys, other.objectKeys)
	}
}

func TestDeleteSpaceUsecase_Execute_論理削除していないスペースは消さない(t *testing.T) {
	t.Parallel()

	f := setupDeleteSpaceFixture(t, "active")

	if err := f.deleteSpace(t); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	f.assertRowsRemain(t)
	if keys := f.remainingObjects(); len(keys) != len(f.objectKeys) {
		t.Errorf("残った実体 = %v、期待値 = %v", keys, f.objectKeys)
	}
}

func TestDeleteSpaceUsecase_Execute_行の無いスペースでは何もしない(t *testing.T) {
	t.Parallel()

	f := setupDeleteAttachmentFixture(t, "del-space-missing")
	uc := (&deleteSpaceFixture{deleteAttachmentFixture: f}).deleteSpaceUsecase(f.objectStorage)

	if err := uc.Execute(t.Context(), DeleteSpaceInput{SpaceID: "00000000-0000-0000-0000-000000000000"}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
}

func TestDeleteSpaceUsecase_Execute_実体の削除に失敗したら行が残り再試行できる(t *testing.T) {
	t.Parallel()

	f := setupDeleteSpaceFixture(t, "object-error")
	f.discard(t)
	f.objectStorage.DeleteErr = errors.New("ストレージに到達できません")

	if err := f.deleteSpace(t); err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
	f.assertRowsRemain(t)

	f.objectStorage.DeleteErr = nil
	if err := f.deleteSpace(t); err != nil {
		t.Fatalf("再試行のExecute()のエラー = %v", err)
	}
	f.cleanupExports(t)
	f.assertNoRows(t)
	if keys := f.remainingObjects(); len(keys) != 0 {
		t.Errorf("残った実体 = %v、期待値 = なし", keys)
	}
}

func TestDeleteSpaceUsecase_Execute_行の削除が途中で失敗したら1つも消えない(t *testing.T) {
	t.Parallel()

	f := setupDeleteSpaceFixture(t, "rows-error")
	f.discard(t)

	// 添付ファイルのblobをスペースの外のレコードからも参照させ、添付ファイルの行の削除を外部キーで失敗させる。
	// それより前に消したページ・トピックなどの行も、同じトランザクションのため元に戻る
	if _, err := f.db.ExecContext(context.Background(), `
		INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		SELECT 'file', 'OtherRecord', gen_random_uuid(), b.id, NOW() FROM active_storage_blobs b WHERE b.key = $1`, f.objectKeys[0]); err != nil {
		t.Fatalf("blobを参照する行の作成に失敗: %v", err)
	}

	if err := f.deleteSpace(t); err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
	f.assertRowsRemain(t)
	if jobs := f.cleanupJobs(t); len(jobs) != 0 {
		t.Errorf("後始末ジョブの数 = %d、期待値 = 0", len(jobs))
	}
}

func TestDeleteSpaceUsecase_Execute_2回実行しても結果が変わらない(t *testing.T) {
	t.Parallel()

	f := setupDeleteSpaceFixture(t, "twice")
	f.discard(t)

	for i := range 2 {
		if err := f.deleteSpace(t); err != nil {
			t.Fatalf("%d回目のExecute()のエラー = %v", i+1, err)
		}
	}
	f.cleanupExports(t)
	f.assertNoRows(t)
	if jobs := f.cleanupJobs(t); len(jobs) != 1 {
		t.Errorf("後始末ジョブの数 = %d、期待値 = 1", len(jobs))
	}
}

func TestDeleteSpaceUsecase_Execute_添付ファイルが増えても削除のリクエストの数は増えない(t *testing.T) {
	t.Parallel()

	f := setupDeleteSpaceFixture(t, "requests")
	for _, name := range []string{"a", "b", "c", "d"} {
		id, key := f.add(t, name)
		f.attachmentIDs = append(f.attachmentIDs, id)
		f.objectKeys = append(f.objectKeys, key, f.addVariant(t, id, name))
	}
	f.discard(t)

	if err := f.deleteSpace(t); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	// Go版のアーカイブで1回、Rails版のZIPで1回、5件の添付ファイルとサムネイルで1回
	if got := f.objectStorage.DeleteManyCalls(); got != 3 {
		t.Errorf("一括削除のリクエストの数 = %d、期待値 = 3", got)
	}
	if keys := f.remainingObjects(); len(keys) != 0 {
		t.Errorf("残った実体 = %v、期待値 = なし", keys)
	}
}

func TestDeleteSpaceUsecase_Execute_オブジェクトストレージが無い場合(t *testing.T) {
	t.Parallel()

	t.Run("実体を持たないスペースは消せる", func(t *testing.T) {
		t.Parallel()

		f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "del-space-no-storage")}
		s := deleteTopicSpace{spaceID: f.spaceID, spaceMemberID: f.memberID}
		topicID := testutil.NewTopicBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithNumber(10).WithName("物理削除するトピック").Build()
		testutil.NewPageBuilderDB(t, f.db).WithSpaceID(s.spaceID).WithTopicID(topicID).WithNumber(10).Build()
		f.discard(t)

		if err := f.deleteSpaceUsecase(nil).Execute(t.Context(), DeleteSpaceInput{SpaceID: f.spaceID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		f.assertNoRows(t)
	})

	t.Run("実体を持つスペースは失敗して行を残す", func(t *testing.T) {
		t.Parallel()

		f := setupDeleteSpaceFixture(t, "no-storage-objects")
		f.discard(t)

		err := f.deleteSpaceUsecase(nil).Execute(t.Context(), DeleteSpaceInput{SpaceID: f.spaceID})
		if !errors.Is(err, errSpaceObjectStorageNotConfigured) {
			t.Fatalf("Execute()のエラー = %v、期待値 = %v", err, errSpaceObjectStorageNotConfigured)
		}
		f.assertRowsRemain(t)
	})
}

func TestDeleteSpaceUsecase_ストレージ未設定ならロック待ちの間に追加された実体の行を残す(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		kind string
	}{
		{name: "添付ファイル", kind: "attachment"},
		{name: "エクスポート", kind: "export"},
	} {
		kind := tt.kind
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &deleteSpaceFixture{deleteAttachmentFixture: setupDeleteAttachmentFixture(t, "no-storage-late-"+kind)}
			f.discard(t)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			tx, err := f.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback() }()
			var blockerPID int
			var locked string
			if err := tx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockerPID); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRowContext(ctx, "SELECT id FROM spaces WHERE id=$1 FOR KEY SHARE", string(f.spaceID)).Scan(&locked); err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() { result <- f.deleteSpaceUsecase(nil).Execute(ctx, DeleteSpaceInput{SpaceID: f.spaceID}) }()
			waitForQueryBlockedBy(t, ctx, f.db, blockerPID, "LockDiscardedSpace", result)
			key := "no-storage-late-" + kind
			if kind == "attachment" {
				testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.spaceID).WithSpaceMemberID(f.memberID).WithBlobKey(key).Build()
			} else {
				testutil.NewExportBuilder(t, tx).WithSpaceID(f.spaceID).WithQueuedByID(f.memberID).WithObjectKey(key).Build()
			}
			f.objectStorage.Put(key, []byte("file"), "application/octet-stream")
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-result; !errors.Is(err, errSpaceObjectStorageNotConfigured) {
				t.Fatalf("削除のエラー = %v、期待値 = %v", err, errSpaceObjectStorageNotConfigured)
			}
			var rows int
			if err := f.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM spaces WHERE id=$1", string(f.spaceID)).Scan(&rows); err != nil {
				t.Fatal(err)
			}
			if rows != 1 || !f.hasObject(key) || len(f.cleanupJobs(t))+len(f.exportCleanupJobs(t)) != 0 {
				t.Error("実体を残したままスペースの行を消したか、後始末ジョブを登録した")
			}
		})
	}
}

func TestDeleteSpaceUsecase_Execute_スペースメンバーが消えるとアカウントの物理削除が進む(t *testing.T) {
	t.Parallel()

	f := setupDeleteSpaceFixture(t, "account")
	userID := seedAccountWithRelatedRecords(t, f.db, "del-space", true)
	testutil.NewSpaceMemberBuilderDB(t, f.db).WithSpaceID(f.spaceID).WithUserID(userID).Build()
	accountUC := newDeleteAccountUsecase(f.db)

	if err := accountUC.Execute(t.Context(), DeleteAccountInput{UserID: userID}); !errors.Is(err, ErrAccountHasSpaceMembers) {
		t.Fatalf("スペースの削除前のExecute()のエラー = %v、ErrAccountHasSpaceMembersを期待", err)
	}

	f.discard(t)
	if err := f.deleteSpace(t); err != nil {
		t.Fatalf("スペースの物理削除のエラー = %v", err)
	}
	if err := accountUC.Execute(t.Context(), DeleteAccountInput{UserID: userID}); err != nil {
		t.Fatalf("スペースの削除後のExecute()のエラー = %v", err)
	}
	assertUserRows(t, f.db, userID, false)
}

// スペースの物理削除が消す範囲は、スペース・スペースメンバー・添付ファイル・エクスポートを参照する外部キーから
// 決めている。外部キーを持つ表が増えたり、外部キーの削除時の動作が変わったりしたら、DeleteSpaceUsecaseが消す
// 範囲を見直してから、この一覧を更新する。トピックとページを参照する外部キーは、トピックの物理削除のテストが
// 同じ形で照合している
func TestDeleteSpaceUsecase_外部キーの一覧が物理削除の範囲と一致する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()

	// 値は外部キーの削除時の動作 (a: NO ACTION、c: CASCADE、n: SET NULL)。NO ACTIONの外部キーは
	// 親より先に行を削除し、CASCADEとSET NULLはデータベースに任せる。
	want := map[string]string{
		// スペースを参照する表
		"attachments.space_id -> spaces":                 "a", // attachmentDeleter.deleteRowsTx
		"draft_page_revisions.space_id -> spaces":        "c",
		"draft_pages.space_id -> spaces":                 "a", // deletePagesWithRelatedRecordsとDraftPageRepository.DeleteBySpace
		"exports.space_id -> spaces":                     "c",
		"oauth_access_tokens.space_id -> spaces":         "c",
		"oauth_applications.space_id -> spaces":          "c",
		"oauth_authorization_codes.space_id -> spaces":   "c",
		"oauth_device_authorizations.space_id -> spaces": "c",
		"oauth_grants.space_id -> spaces":                "c",
		"oauth_refresh_tokens.space_id -> spaces":        "c",
		"page_editors.space_id -> spaces":                "a", // deletePagesWithRelatedRecords
		"page_revisions.space_id -> spaces":              "a", // deletePagesWithRelatedRecords
		"pages.space_id -> spaces":                       "a", // deletePagesWithRelatedRecords
		"personal_access_tokens.space_id -> spaces":      "c",
		"space_members.space_id -> spaces":               "a", // SpaceMemberRepository.DeleteBySpace
		"suggestion_comments.space_id -> spaces":         "c",
		"suggestion_page_revisions.space_id -> spaces":   "c",
		"suggestion_pages.space_id -> spaces":            "c",
		"suggestions.space_id -> spaces":                 "c",
		"topic_members.space_id -> spaces":               "a", // TopicMemberRepository.DeleteBySpace
		"topics.space_id -> spaces":                      "a", // TopicRepository.DeleteBySpace
		// スペースメンバーを参照する表 (スペースメンバーより先に消すか、CASCADEで消える)
		"attachments.attached_space_member_id -> space_members":             "a",
		"draft_page_revisions.space_member_id -> space_members":             "c",
		"draft_pages.space_member_id -> space_members":                      "a",
		"exports.queued_by_id -> space_members":                             "c",
		"oauth_applications.created_space_member_id -> space_members":       "n",
		"oauth_grants.space_member_id -> space_members":                     "c",
		"page_editors.space_member_id -> space_members":                     "a",
		"page_revisions.space_member_id -> space_members":                   "a",
		"personal_access_tokens.space_member_id -> space_members":           "c",
		"suggestion_comments.created_space_member_id -> space_members":      "c",
		"suggestion_page_revisions.editor_space_member_id -> space_members": "c",
		"suggestions.created_space_member_id -> space_members":              "c",
		"topic_members.space_member_id -> space_members":                    "a",
		// 添付ファイルを参照する表 (attachmentDeleter.deleteRowsTxが添付ファイルより先に消す)
		"draft_pages.featured_image_attachment_id -> attachments":      "n",
		"page_attachment_references.attachment_id -> attachments":      "a",
		"pages.featured_image_attachment_id -> attachments":            "n",
		"suggestion_pages.featured_image_attachment_id -> attachments": "n",
	}

	rows, err := db.QueryContext(t.Context(), `
		SELECT child.relname, a.attname, parent.relname, c.confdeltype
		FROM pg_constraint c
		JOIN pg_class child ON child.oid = c.conrelid
		JOIN pg_class parent ON parent.oid = c.confrelid
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		WHERE c.contype = 'f'
		  AND parent.relname = ANY($1::text[])`,
		"{spaces,space_members,attachments,exports}",
	)
	if err != nil {
		t.Fatalf("外部キーの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	got := map[string]string{}
	for rows.Next() {
		var child, column, parent, action string
		if err := rows.Scan(&child, &column, &parent, &action); err != nil {
			t.Fatalf("外部キーの読み取りに失敗: %v", err)
		}
		got[child+"."+column+" -> "+parent] = action
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("外部キーの読み取りに失敗: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		keys := make([]string, 0, len(got)+len(want))
		for k := range got {
			keys = append(keys, k)
		}
		for k := range want {
			if _, ok := got[k]; !ok {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			if got[k] != want[k] {
				t.Errorf("%s: 実際の削除時の動作 = %q、期待値 = %q", k, got[k], want[k])
			}
		}
	}
}
