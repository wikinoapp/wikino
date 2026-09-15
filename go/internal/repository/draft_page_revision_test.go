package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestDraftPageRevisionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRevisionRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("draftpagerev-create@example.com").
		WithAtname("draftpagerevcreate").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("draftpagerev-create").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		WithBodyHTML("<p>Draft body</p>").
		Build()

	t.Run("下書きページリビジョンを作成できる", func(t *testing.T) {
		revision, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Revision Title",
			Body:          "revision body",
			BodyHTML:      "<p>revision body</p>",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if revision == nil {
			t.Fatal("Create()がnilを返した、期待値 = 下書きのリビジョン")
		}
		if revision.ID == "" {
			t.Error("revision.IDが空")
		}
		if revision.DraftPageID != draftPageID {
			t.Errorf("revision.DraftPageID = %v、期待値 = %v", revision.DraftPageID, draftPageID)
		}
		if revision.SpaceID != spaceID {
			t.Errorf("revision.SpaceID = %v、期待値 = %v", revision.SpaceID, spaceID)
		}
		if revision.SpaceMemberID != spaceMemberID {
			t.Errorf("revision.SpaceMemberID = %v、期待値 = %v", revision.SpaceMemberID, spaceMemberID)
		}
		if revision.Title != "Revision Title" {
			t.Errorf("revision.Title = %v、期待値 = 'Revision Title'", revision.Title)
		}
		if revision.Body != "revision body" {
			t.Errorf("revision.Body = %v、期待値 = 'revision body'", revision.Body)
		}
		if revision.BodyHTML != "<p>revision body</p>" {
			t.Errorf("revision.BodyHTML = %v、期待値 = '<p>revision body</p>'", revision.BodyHTML)
		}
		if revision.CreatedAt.IsZero() {
			t.Error("revision.CreatedAtがゼロ値")
		}
	})

	t.Run("下書きページIDに紐づくリビジョンをすべて削除できる", func(t *testing.T) {
		_, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Delete Test 1",
			Body:          "delete body 1",
			BodyHTML:      "<p>delete body 1</p>",
		})
		if err != nil {
			t.Fatalf("Create() (1件目のリビジョン) のエラー = %v", err)
		}

		_, err = repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Delete Test 2",
			Body:          "delete body 2",
			BodyHTML:      "<p>delete body 2</p>",
		})
		if err != nil {
			t.Fatalf("Create() (2件目のリビジョン) のエラー = %v", err)
		}

		err = repo.DeleteByDraftPageID(context.Background(), draftPageID, spaceID)
		if err != nil {
			t.Fatalf("DeleteByDraftPageID()のエラー = %v", err)
		}
	})

	t.Run("下書きページIDに紐づくリビジョン件数を取得できる", func(t *testing.T) {
		// 検証用に独立したPageとDraftPageを作成 (unique制約と他サブテストの干渉を避けるため)。
		countPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(2).
			WithTitle("Count Test Page").
			Build()

		countDraftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(countPageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("Count Draft Title").
			WithBody("count draft body").
			WithBodyHTML("<p>count draft body</p>").
			Build()

		count, err := repo.CountByDraftPageID(context.Background(), countDraftPageID, spaceID)
		if err != nil {
			t.Fatalf("CountByDraftPageID()のエラー = %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d、期待値 = 0", count)
		}

		_, err = repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   countDraftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Count Test 1",
			Body:          "count body 1",
			BodyHTML:      "<p>count body 1</p>",
		})
		if err != nil {
			t.Fatalf("Create() (1件目のリビジョン) のエラー = %v", err)
		}
		_, err = repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   countDraftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Count Test 2",
			Body:          "count body 2",
			BodyHTML:      "<p>count body 2</p>",
		})
		if err != nil {
			t.Fatalf("Create() (2件目のリビジョン) のエラー = %v", err)
		}

		count, err = repo.CountByDraftPageID(context.Background(), countDraftPageID, spaceID)
		if err != nil {
			t.Fatalf("CountByDraftPageID()のエラー = %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d、期待値 = 2", count)
		}
	})

	// Rails側の削除経路が頼るON DELETE CASCADEの契約を検証する。
	// draft_pagesの行を直接DELETEしたとき、draft_page_revisionsへの明示的な
	// DELETEなしでリビジョンも一緒に消えること。
	t.Run("下書きページの行を直接DELETEするとリビジョンも消える", func(t *testing.T) {
		// 検証用に独立したPageとDraftPageを作成 (他サブテストの干渉を避けるため)
		cascadePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(3).
			WithTitle("Cascade Test Page").
			Build()

		cascadeDraftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(cascadePageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle("Cascade Draft Title").
			WithBody("cascade draft body").
			WithBodyHTML("<p>cascade draft body</p>").
			Build()

		_, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   cascadeDraftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Cascade Test",
			Body:          "cascade body",
			BodyHTML:      "<p>cascade body</p>",
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}

		// 親のdraft_pagesの行を直接削除 (アプリケーションコードを経由しない)
		_, err = tx.ExecContext(
			context.Background(),
			"DELETE FROM draft_pages WHERE id = $1 AND space_id = $2",
			string(cascadeDraftPageID), string(spaceID),
		)
		if err != nil {
			t.Fatalf("DELETE draft_pagesのエラー = %v", err)
		}

		count, err := repo.CountByDraftPageID(context.Background(), cascadeDraftPageID, spaceID)
		if err != nil {
			t.Fatalf("CountByDraftPageID()のエラー = %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d、期待値 = 0 (リビジョンがカスケード削除されていない)", count)
		}
	})

	t.Run("同じ下書きに対して複数のリビジョンを作成できる", func(t *testing.T) {
		revision1, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "First Revision",
			Body:          "first body",
			BodyHTML:      "<p>first body</p>",
		})
		if err != nil {
			t.Fatalf("Create() (1件目のリビジョン) のエラー = %v", err)
		}

		revision2, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
			DraftPageID:   draftPageID,
			SpaceID:       spaceID,
			SpaceMemberID: spaceMemberID,
			Title:         "Second Revision",
			Body:          "second body",
			BodyHTML:      "<p>second body</p>",
		})
		if err != nil {
			t.Fatalf("Create() (2件目のリビジョン) のエラー = %v", err)
		}

		if revision1.ID == revision2.ID {
			t.Errorf("revision1.IDとrevision2.IDが同じ: %v", revision1.ID)
		}
		if revision1.Title != "First Revision" {
			t.Errorf("revision1.Title = %v、期待値 = 'First Revision'", revision1.Title)
		}
		if revision2.Title != "Second Revision" {
			t.Errorf("revision2.Title = %v、期待値 = 'Second Revision'", revision2.Title)
		}
	})
}

func TestDraftPageRevisionRepository_ListByDraftPageID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRevisionRepository(q)

	spaceID, spaceMemberID, draftPageID := setupDraftRevisionFixture(t, tx, "list")

	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// 3件を古い順に作成し (created_at: rev1 < rev2 < rev3)、一覧が新しい順へ並べ替えることを検証する。
	rev1 := insertDraftRevision(t, tx, draftPageID, spaceID, spaceMemberID, "rev1", base.Add(1*time.Second))
	rev2 := insertDraftRevision(t, tx, draftPageID, spaceID, spaceMemberID, "rev2", base.Add(2*time.Second))
	rev3 := insertDraftRevision(t, tx, draftPageID, spaceID, spaceMemberID, "rev3", base.Add(3*time.Second))

	t.Run("新しい順に取得できる", func(t *testing.T) {
		revisions, err := repo.ListByDraftPageID(context.Background(), draftPageID, spaceID, 20)
		if err != nil {
			t.Fatalf("ListByDraftPageID()のエラー = %v", err)
		}
		wantOrder := []model.DraftPageRevisionID{rev3.ID, rev2.ID, rev1.ID}
		if len(revisions) != len(wantOrder) {
			t.Fatalf("len(revisions) = %d、期待値 = %d", len(revisions), len(wantOrder))
		}
		for i, want := range wantOrder {
			if revisions[i].ID != want {
				t.Errorf("revisions[%d].ID = %v、期待値 = %v", i, revisions[i].ID, want)
			}
		}
	})

	t.Run("limitで取得件数を制限できる", func(t *testing.T) {
		revisions, err := repo.ListByDraftPageID(context.Background(), draftPageID, spaceID, 2)
		if err != nil {
			t.Fatalf("ListByDraftPageID()のエラー = %v", err)
		}
		if len(revisions) != 2 {
			t.Fatalf("len(revisions) = %d、期待値 = 2", len(revisions))
		}
		// 新しい2件 (rev3, rev2) のみが返る。
		if revisions[0].ID != rev3.ID || revisions[1].ID != rev2.ID {
			t.Errorf("ID = [%v, %v]、期待値 = [%v, %v]", revisions[0].ID, revisions[1].ID, rev3.ID, rev2.ID)
		}
	})

	t.Run("リビジョンが無い下書きでは空スライスを返す", func(t *testing.T) {
		emptySpaceID, _, emptyDraftPageID := setupDraftRevisionFixture(t, tx, "listempty")
		revisions, err := repo.ListByDraftPageID(context.Background(), emptyDraftPageID, emptySpaceID, 20)
		if err != nil {
			t.Fatalf("ListByDraftPageID()のエラー = %v", err)
		}
		if revisions == nil {
			t.Fatal("revisionsがnil (期待値 = nilではない空のスライス)")
		}
		if len(revisions) != 0 {
			t.Errorf("len(revisions) = %d、期待値 = 0", len(revisions))
		}
	})

	t.Run("別スペースのスペースIDでは取得できない", func(t *testing.T) {
		otherSpaceID, _, _ := setupDraftRevisionFixture(t, tx, "listother")
		revisions, err := repo.ListByDraftPageID(context.Background(), draftPageID, otherSpaceID, 20)
		if err != nil {
			t.Fatalf("ListByDraftPageID()のエラー = %v", err)
		}
		if len(revisions) != 0 {
			t.Errorf("len(revisions) = %d、期待値 = 0 (別スペースからリビジョンが見えている)", len(revisions))
		}
	})
}

func TestDraftPageRevisionRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRevisionRepository(q)

	spaceID, spaceMemberID, draftPageID := setupDraftRevisionFixture(t, tx, "find")

	created, err := repo.Create(context.Background(), CreateDraftPageRevisionInput{
		DraftPageID:   draftPageID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         "Find Title",
		Body:          "find body",
		BodyHTML:      "<p>find body</p>",
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}

	t.Run("IDで取得できる", func(t *testing.T) {
		revision, err := repo.FindByID(context.Background(), created.ID, spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if revision == nil {
			t.Fatal("FindByID()がnilを返した、期待値 = リビジョン")
		}
		if revision.ID != created.ID {
			t.Errorf("revision.ID = %v、期待値 = %v", revision.ID, created.ID)
		}
		if revision.Title != "Find Title" {
			t.Errorf("revision.Title = %v、期待値 = 'Find Title'", revision.Title)
		}
	})

	t.Run("存在しないIDでは(nil, nil)を返す", func(t *testing.T) {
		revision, err := repo.FindByID(context.Background(), model.DraftPageRevisionID("00000000-0000-0000-0000-000000000000"), spaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if revision != nil {
			t.Errorf("FindByID() = %v、期待値 = nil", revision)
		}
	})

	t.Run("別スペースのスペースIDでは(nil, nil)を返す", func(t *testing.T) {
		otherSpaceID, _, _ := setupDraftRevisionFixture(t, tx, "findother")
		revision, err := repo.FindByID(context.Background(), created.ID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if revision != nil {
			t.Errorf("FindByID() = %v、期待値 = nil (別スペースからリビジョンが見えている)", revision)
		}
	})
}

func TestDraftPageRevisionRepository_FindPrevious(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewDraftPageRevisionRepository(q)

	spaceID, spaceMemberID, draftPageID := setupDraftRevisionFixture(t, tx, "prev")

	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	rev1 := insertDraftRevision(t, tx, draftPageID, spaceID, spaceMemberID, "rev1", base.Add(1*time.Second))
	rev2 := insertDraftRevision(t, tx, draftPageID, spaceID, spaceMemberID, "rev2", base.Add(2*time.Second))
	rev3 := insertDraftRevision(t, tx, draftPageID, spaceID, spaceMemberID, "rev3", base.Add(3*time.Second))

	t.Run("直前のリビジョンを取得できる", func(t *testing.T) {
		prev, err := repo.FindPrevious(context.Background(), rev3)
		if err != nil {
			t.Fatalf("FindPrevious()のエラー = %v", err)
		}
		if prev == nil {
			t.Fatal("FindPrevious()がnilを返した、期待値 = rev2")
		}
		if prev.ID != rev2.ID {
			t.Errorf("prev.ID = %v、期待値 = %v (rev2)", prev.ID, rev2.ID)
		}
	})

	t.Run("最古のリビジョンでは(nil, nil)を返す", func(t *testing.T) {
		prev, err := repo.FindPrevious(context.Background(), rev1)
		if err != nil {
			t.Fatalf("FindPrevious()のエラー = %v", err)
		}
		if prev != nil {
			t.Errorf("FindPrevious() = %v、期待値 = nil (最古のリビジョンには前のリビジョンが無い)", prev)
		}
	})

	// created_atが同一でも (created_at, id) の全順序で直前を一意に決められる。
	// idが小さい行が「より古い」。
	t.Run("created_atが同一の場合はidで直前を決める", func(t *testing.T) {
		tieSpaceID, tieMemberID, tieDraftPageID := setupDraftRevisionFixture(t, tx, "prevtie")
		sameTime := base.Add(10 * time.Second)
		a := insertDraftRevision(t, tx, tieDraftPageID, tieSpaceID, tieMemberID, "tieA", sameTime)
		b := insertDraftRevision(t, tx, tieDraftPageID, tieSpaceID, tieMemberID, "tieB", sameTime)

		// idの大小で新旧を決める (大きいidが新しい)。直前はidが小さい行になる。
		newer, older := a, b
		if a.ID < b.ID {
			newer, older = b, a
		}

		prev, err := repo.FindPrevious(context.Background(), newer)
		if err != nil {
			t.Fatalf("FindPrevious()のエラー = %v", err)
		}
		if prev == nil || prev.ID != older.ID {
			t.Fatalf("FindPrevious(newer).ID = %v、期待値 = %v (older)", prev, older.ID)
		}

		prevOfOlder, err := repo.FindPrevious(context.Background(), older)
		if err != nil {
			t.Fatalf("FindPrevious()のエラー = %v", err)
		}
		if prevOfOlder != nil {
			t.Errorf("FindPrevious(older) = %v、期待値 = nil", prevOfOlder)
		}
	})
}

// ユーザー / スペース / メンバー / トピック / ページ / 下書きページの一連を作成し、リビジョン
// クエリの検証に必要なIDを返す。suffixで識別子をサブテスト間で一意にする (atnameは20文字
// 上限のため短く保つ)。
func setupDraftRevisionFixture(t *testing.T, tx *sql.Tx, suffix string) (model.SpaceID, model.SpaceMemberID, model.DraftPageID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("dpr-" + suffix + "@example.com").
		WithAtname("dpr" + suffix).
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("dpr-" + suffix).
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		WithBodyHTML("<p>Draft body</p>").
		Build()

	return spaceID, spaceMemberID, draftPageID
}

// created_atを明示指定して下書きページリビジョンを挿入する。リポジトリのCreateは
// created_atにtime.Now() を打つため順序を制御できないので、順序検証用に直接挿入する。
// DBに保存された値からモデルを構築して返すため、CreatedAtはクエリが比較する切り詰め済みの
// タイムスタンプと一致する。
func insertDraftRevision(t *testing.T, tx *sql.Tx, draftPageID model.DraftPageID, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, title string, createdAt time.Time) *model.DraftPageRevision {
	t.Helper()

	body := "body-" + title
	bodyHTML := "<p>" + title + "</p>"

	var id string
	var storedCreatedAt time.Time
	err := tx.QueryRowContext(context.Background(),
		`INSERT INTO draft_page_revisions (draft_page_id, space_id, space_member_id, title, body, body_html, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at`,
		string(draftPageID), string(spaceID), string(spaceMemberID), title, body, bodyHTML, createdAt,
	).Scan(&id, &storedCreatedAt)
	if err != nil {
		t.Fatalf("insertDraftRevision()のエラー = %v", err)
	}

	return &model.DraftPageRevision{
		ID:            model.DraftPageRevisionID(id),
		DraftPageID:   draftPageID,
		SpaceID:       spaceID,
		SpaceMemberID: spaceMemberID,
		Title:         title,
		Body:          body,
		BodyHTML:      bodyHTML,
		CreatedAt:     storedCreatedAt,
	}
}
