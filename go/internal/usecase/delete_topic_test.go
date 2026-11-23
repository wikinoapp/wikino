package usecase

import (
	"context"
	"database/sql"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newDeleteTopicUsecaseはテスト用のデータベースを使うUseCaseを組み立てる。
func newDeleteTopicUsecase(db *sql.DB) *DeleteTopicUsecase {
	queries := query.New(db)
	return NewDeleteTopicUsecase(
		db,
		repository.NewTopicRepository(queries),
		repository.NewTopicMemberRepository(queries),
		repository.NewPageRepository(queries),
		repository.NewDraftPageRepository(queries),
		repository.NewPageEditorRepository(queries),
		repository.NewPageRevisionRepository(queries),
		repository.NewPageAttachmentReferenceRepository(queries),
	)
}

// deleteTopicSpaceはスペースと、そのメンバー・添付ファイルを保持する。
type deleteTopicSpace struct {
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
	attachmentID  model.AttachmentID
}

// setupDeleteTopicSpaceは、管理者のメンバーと添付ファイルを1つずつ持つスペースを作成する。
func setupDeleteTopicSpace(t *testing.T, db *sql.DB, suffix string) deleteTopicSpace {
	t.Helper()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-topic-" + suffix + "@example.com").
		WithAtname(strings.ReplaceAll("delete_topic_"+suffix, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier("delete-topic-" + suffix).
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	attachmentID := testutil.NewAttachmentBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	return deleteTopicSpace{spaceID: spaceID, spaceMemberID: spaceMemberID, attachmentID: attachmentID}
}

// seedTopicWithRelatedRecordsは、物理削除が消す関連レコードを1つずつ持つトピックを作成する。
// ページの下書き (とそのリビジョン)・編集者・リビジョン・添付ファイル参照、トピックメンバー、
// 編集提案 (とそのページ) を作る。
func seedTopicWithRelatedRecords(t *testing.T, db *sql.DB, s deleteTopicSpace, number int32) (model.TopicID, model.PageID) {
	t.Helper()

	ctx := context.Background()
	queries := query.New(db)

	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithNumber(number).
		WithName("トピック" + string(rune('A'+number))).
		Build()
	testutil.NewTopicMemberBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(s.spaceMemberID).
		Build()

	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithTopicID(topicID).
		WithNumber(model.PageNumber(number)).
		Build()
	draftPageID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(s.spaceMemberID).
		WithTopicID(topicID).
		Build()
	testutil.NewDraftPageRevisionBuilderDB(t, db).
		WithDraftPageID(draftPageID).
		WithSpaceID(s.spaceID).
		WithSpaceMemberID(s.spaceMemberID).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithSpaceMemberID(s.spaceMemberID).
		WithPageID(pageID).
		Build()
	if _, err := repository.NewPageEditorRepository(queries).FindOrCreate(ctx, repository.FindOrCreateInput{
		SpaceID:            s.spaceID,
		PageID:             pageID,
		SpaceMemberID:      s.spaceMemberID,
		LastPageModifiedAt: time.Now(),
	}); err != nil {
		t.Fatalf("ページの編集者の作成に失敗: %v", err)
	}
	if _, err := repository.NewPageAttachmentReferenceRepository(queries).CreateBatch(ctx, pageID, s.spaceID, []model.AttachmentID{s.attachmentID}); err != nil {
		t.Fatalf("添付ファイル参照の作成に失敗: %v", err)
	}

	suggestionID := testutil.NewSuggestionBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(s.spaceMemberID).
		Build()
	testutil.NewSuggestionPageBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	return topicID, pageID
}

// countTopicRelatedRowsはトピックとそのページに属する行を表ごとに数える。
func countTopicRelatedRows(t *testing.T, db *sql.DB, spaceID model.SpaceID, topicID model.TopicID, pageID model.PageID) map[string]int {
	t.Helper()

	counts := map[string]int{
		"topics":               countRows(t, db, "topics", spaceID, "id = $2", topicID),
		"topic_members":        countRows(t, db, "topic_members", spaceID, "topic_id = $2", topicID),
		"pages":                countRows(t, db, "pages", spaceID, "id = $2", pageID),
		"draft_pages":          countRows(t, db, "draft_pages", spaceID, "page_id = $2", pageID),
		"draft_page_revisions": countRows(t, db, "draft_page_revisions", spaceID, "draft_page_id IN (SELECT id FROM draft_pages WHERE page_id = $2)", pageID),
		"page_editors":         countRows(t, db, "page_editors", spaceID, "page_id = $2", pageID),
		"page_revisions":       countRows(t, db, "page_revisions", spaceID, "page_id = $2", pageID),
		"suggestions":          countRows(t, db, "suggestions", spaceID, "topic_id = $2", topicID),
		"suggestion_pages":     countRows(t, db, "suggestion_pages", spaceID, "page_id = $2", pageID),
	}

	var refs int
	if err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM page_attachment_references WHERE page_id = $1", pageID).Scan(&refs); err != nil {
		t.Fatalf("page_attachment_referencesの行数の取得に失敗: %v", err)
	}
	counts["page_attachment_references"] = refs

	return counts
}

// discardTopicForTestはトピックを論理削除する。
func discardTopicForTest(t *testing.T, db *sql.DB, spaceID model.SpaceID, topicID model.TopicID) {
	t.Helper()

	discarded, err := repository.NewTopicRepository(query.New(db)).Discard(t.Context(), topicID, spaceID, time.Now())
	if err != nil || !discarded {
		t.Fatalf("トピックの論理削除に失敗: discarded = %v, err = %v", discarded, err)
	}
}

func TestDeleteTopicUsecase_Execute(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	s := setupDeleteTopicSpace(t, db, "execute")
	other := setupDeleteTopicSpace(t, db, "execute-other")

	targetTopicID, targetPageID := seedTopicWithRelatedRecords(t, db, s, 1)
	keptTopicID, keptPageID := seedTopicWithRelatedRecords(t, db, s, 2)
	otherTopicID, otherPageID := seedTopicWithRelatedRecords(t, db, other, 1)

	// ページが別のトピックへ移ったあとも、下書きが削除するトピックを指したまま残っている場合。
	// 下書きはページとメンバーの組に1つのため、別のメンバーの下書きにする。
	strayUserID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-topic-execute-stray@example.com").
		WithAtname("delete_topic_execute_stray").
		Build()
	strayMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithUserID(strayUserID).
		Build()
	strayDraftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithPageID(keptPageID).
		WithSpaceMemberID(strayMemberID).
		WithTopicID(targetTopicID).
		WithTitle("残すページの下書き").
		WithBody("まだ公開していない編集内容").
		WithLinkedPageIDs([]model.PageID{keptPageID}).
		WithFeaturedImageAttachmentID(s.attachmentID).
		Build()
	testutil.NewDraftPageRevisionBuilderDB(t, db).
		WithDraftPageID(strayDraftID).
		WithSpaceID(s.spaceID).
		WithSpaceMemberID(strayMemberID).
		Build()
	queries := query.New(db)
	draftRepo := repository.NewDraftPageRepository(queries)
	revisionRepo := repository.NewDraftPageRevisionRepository(queries)
	beforeDraft, err := draftRepo.FindByID(t.Context(), strayDraftID, s.spaceID)
	if err != nil || beforeDraft == nil {
		t.Fatalf("下書きの取得に失敗: draft = %+v, err = %v", beforeDraft, err)
	}
	beforeRevisions, err := revisionRepo.ListByDraftPageID(t.Context(), strayDraftID, s.spaceID, 10)
	if err != nil {
		t.Fatalf("下書きの履歴の取得に失敗: %v", err)
	}

	discardTopicForTest(t, db, s.spaceID, targetTopicID)

	uc := newDeleteTopicUsecase(db)
	if err := uc.Execute(t.Context(), DeleteTopicInput{SpaceID: s.spaceID, TopicID: targetTopicID}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	for table, n := range countTopicRelatedRows(t, db, s.spaceID, targetTopicID, targetPageID) {
		if n != 0 {
			t.Errorf("削除したトピックの%sが%d行残っている", table, n)
		}
	}
	afterDraft, err := draftRepo.FindByID(t.Context(), strayDraftID, s.spaceID)
	if err != nil {
		t.Fatalf("参照修復後の下書きの取得に失敗: %v", err)
	}
	wantDraft := *beforeDraft
	wantDraft.TopicID = keptTopicID
	if !reflect.DeepEqual(afterDraft, &wantDraft) {
		t.Errorf("参照修復後の下書き = %+v、期待値 = %+v", afterDraft, &wantDraft)
	}
	afterRevisions, err := revisionRepo.ListByDraftPageID(t.Context(), strayDraftID, s.spaceID, 10)
	if err != nil {
		t.Fatalf("参照修復後の下書きの履歴の取得に失敗: %v", err)
	}
	if !reflect.DeepEqual(afterRevisions, beforeRevisions) {
		t.Errorf("参照修復で下書きの履歴が変わった: 修復後 = %+v、修復前 = %+v", afterRevisions, beforeRevisions)
	}
	if n := countRows(t, db, "attachments", s.spaceID, "id = $2", s.attachmentID); n != 1 {
		t.Errorf("添付ファイルの行数 = %d、期待値 = 1", n)
	}

	// 同じスペースのほかのトピックと、ほかのスペースのレコードは消えない
	for _, kept := range []struct {
		name    string
		spaceID model.SpaceID
		topicID model.TopicID
		pageID  model.PageID
	}{
		{name: "同じスペースのほかのトピック", spaceID: s.spaceID, topicID: keptTopicID, pageID: keptPageID},
		{name: "ほかのスペースのトピック", spaceID: other.spaceID, topicID: otherTopicID, pageID: otherPageID},
	} {
		for table, n := range countTopicRelatedRows(t, db, kept.spaceID, kept.topicID, kept.pageID) {
			want := 1
			if kept.pageID == keptPageID && (table == "draft_pages" || table == "draft_page_revisions") {
				want = 2
			}
			if n != want {
				t.Errorf("%sの%sの行数 = %d、期待値 = %d", kept.name, table, n, want)
			}
		}
	}
}

func TestDeleteTopicUsecase_Execute_移動先のページと下書きを保持する(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	db := testutil.GetTestDB()
	s := setupDeleteTopicSpace(t, db, "concurrent-move")
	sourceTopicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithNumber(1).
		WithName("移動元").
		Build()
	destTopicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithNumber(2).
		WithName("移動先").
		Build()
	pageID := testutil.NewPageBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithTopicID(sourceTopicID).
		WithNumber(1).
		Build()
	draftID := testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithTopicID(sourceTopicID).
		WithPageID(pageID).
		WithSpaceMemberID(s.spaceMemberID).
		WithBody("移動後も残す下書き").
		Build()
	testutil.NewDraftPageRevisionBuilderDB(t, db).
		WithDraftPageID(draftID).
		WithSpaceID(s.spaceID).
		WithSpaceMemberID(s.spaceMemberID).
		Build()
	testutil.NewPageRevisionBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithSpaceMemberID(s.spaceMemberID).
		WithPageID(pageID).
		Build()

	// 認可のあとに進んだページ移動と同じ更新を未コミットにし、物理削除が移動の完了を待つ状況を作る。
	moveTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("ページ移動のトランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = moveTx.Rollback() }()
	var movePID int
	if err := moveTx.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&movePID); err != nil {
		t.Fatalf("ページ移動の接続の取得に失敗: %v", err)
	}
	moveQueries := query.New(moveTx)
	if _, err := repository.NewPageRepository(moveQueries).MoveTopic(ctx, repository.MoveTopicInput{
		ID: pageID, SpaceID: s.spaceID, TopicID: destTopicID,
	}); err != nil {
		t.Fatalf("ページの移動に失敗: %v", err)
	}
	if err := repository.NewDraftPageRepository(moveQueries).UpdateTopicByPageID(ctx, pageID, s.spaceID, destTopicID); err != nil {
		t.Fatalf("下書きの移動に失敗: %v", err)
	}
	discardTopicForTest(t, db, s.spaceID, sourceTopicID)

	result := make(chan error, 1)
	go func() {
		result <- newDeleteTopicUsecase(db).Execute(ctx, DeleteTopicInput{SpaceID: s.spaceID, TopicID: sourceTopicID})
	}()
	// ページと下書きのどちらで待っていても、移動がコミットしたあとの保持を検証する。
	waitForQueryBlockedBy(t, ctx, db, movePID, "pages", result)
	if err := moveTx.Commit(); err != nil {
		t.Fatalf("ページ移動のコミットに失敗: %v", err)
	}
	if err := <-result; err != nil {
		t.Fatalf("トピックの物理削除に失敗: %v", err)
	}

	queries := query.New(db)
	page, err := repository.NewPageRepository(queries).FindBySpaceAndNumber(ctx, s.spaceID, 1)
	if err != nil || page == nil || page.TopicID != destTopicID {
		t.Fatalf("移動先のページが保持されていない: page = %+v, err = %v", page, err)
	}
	draft, err := repository.NewDraftPageRepository(queries).FindByID(ctx, draftID, s.spaceID)
	if err != nil || draft == nil || draft.TopicID != destTopicID || draft.Body != "移動後も残す下書き" {
		t.Fatalf("移動先の下書きが保持されていない: draft = %+v, err = %v", draft, err)
	}
	for _, table := range []string{"page_revisions", "draft_page_revisions"} {
		condition, id := "page_id = $2", string(pageID)
		if table == "draft_page_revisions" {
			condition, id = "draft_page_id = $2", string(draftID)
		}
		if n := countRows(t, db, table, s.spaceID, condition, id); n != 1 {
			t.Errorf("移動先の%sの行数 = %d、期待値 = 1", table, n)
		}
	}
	if n := countRows(t, db, "topics", s.spaceID, "id = $2", sourceTopicID); n != 0 {
		t.Errorf("削除した移動元のトピックが%d行残っている", n)
	}
}

// 物理削除は論理削除したトピックだけを対象にする。論理削除していないトピックを指すジョブが
// 動いても何も消さない。
func TestDeleteTopicUsecase_Execute_論理削除していないトピックは消さない(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	s := setupDeleteTopicSpace(t, db, "active")
	topicID, pageID := seedTopicWithRelatedRecords(t, db, s, 1)

	uc := newDeleteTopicUsecase(db)
	if err := uc.Execute(t.Context(), DeleteTopicInput{SpaceID: s.spaceID, TopicID: topicID}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	for table, n := range countTopicRelatedRows(t, db, s.spaceID, topicID, pageID) {
		if n != 1 {
			t.Errorf("%sの行数 = %d、期待値 = 1", table, n)
		}
	}
}

// 再試行で同じジョブが2度動いても失敗しない。
func TestDeleteTopicUsecase_Execute_物理削除したトピックでは何もしない(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	s := setupDeleteTopicSpace(t, db, "twice")
	topicID, _ := seedTopicWithRelatedRecords(t, db, s, 1)
	discardTopicForTest(t, db, s.spaceID, topicID)

	uc := newDeleteTopicUsecase(db)
	for i := range 2 {
		if err := uc.Execute(t.Context(), DeleteTopicInput{SpaceID: s.spaceID, TopicID: topicID}); err != nil {
			t.Fatalf("%d回目のExecute()のエラー = %v", i+1, err)
		}
	}
}

// トピックの物理削除が消す範囲は、トピックとページを参照する外部キーから決めている。外部キーを
// 持つ表が増えたり、外部キーの削除時の動作が変わったりしたら、DeleteTopicUsecaseと
// deletePagesWithRelatedRecordsが消す範囲を見直してから、この一覧を更新する。
func TestDeleteTopicUsecase_外部キーの一覧が物理削除の範囲と一致する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()

	// 値は外部キーの削除時の動作 (a: NO ACTION、c: CASCADE、n: SET NULL)。NO ACTIONの外部キーは
	// 親より先に関連レコードを削除するか参照を修復し、CASCADEとSET NULLはデータベースに任せる。
	want := map[string]string{
		// トピックを参照する表
		"pages.topic_id -> topics":         "a", // deletePagesWithRelatedRecords
		"draft_pages.topic_id -> topics":   "a", // DraftPageRepository.SyncTopicsFromPages
		"topic_members.topic_id -> topics": "a", // TopicMemberRepository.DeleteByTopic
		"suggestions.topic_id -> topics":   "c",
		// ページを参照する表 (deletePagesWithRelatedRecordsがページより先に消す)
		"draft_pages.page_id -> pages":                "a",
		"page_editors.page_id -> pages":               "a",
		"page_revisions.page_id -> pages":             "a",
		"page_attachment_references.page_id -> pages": "a",
		"suggestion_pages.page_id -> pages":           "c",
		// 削除する表を参照する表
		"draft_page_revisions.draft_page_id -> draft_pages":                "c",
		"suggestion_pages.page_revision_id -> page_revisions":              "n",
		"suggestion_comments.suggestion_id -> suggestions":                 "c",
		"suggestion_pages.suggestion_id -> suggestions":                    "c",
		"draft_pages.suggestion_page_id -> suggestion_pages":               "n",
		"suggestion_page_revisions.suggestion_page_id -> suggestion_pages": "c",
	}

	rows, err := db.QueryContext(t.Context(), `
		SELECT child.relname, a.attname, parent.relname, c.confdeltype
		FROM pg_constraint c
		JOIN pg_class child ON child.oid = c.conrelid
		JOIN pg_class parent ON parent.oid = c.confrelid
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		WHERE c.contype = 'f'
		  AND parent.relname = ANY($1::text[])`,
		"{topics,pages,topic_members,draft_pages,page_editors,page_revisions,page_attachment_references,suggestions,suggestion_pages}",
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
			t.Errorf("外部キー %s の削除時の動作 = %q、期待値 = %q", k, got[k], want[k])
		}
	}
}
