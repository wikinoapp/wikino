package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestPageRepository_LockByIDs(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("page-lock-by-ids").Build()
	otherSpaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("page-lock-by-ids-other").Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).Build()
	otherTopicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(otherSpaceID).Build()
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("対象").Build()
	otherID := testutil.NewPageBuilderDB(t, db).WithSpaceID(otherSpaceID).WithTopicID(otherTopicID).WithNumber(1).WithTitle("別スペース").Build()
	discardedID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(2).WithTitle("廃棄済み").Build()
	if _, err := db.ExecContext(t.Context(), "UPDATE pages SET discarded_at = $1 WHERE id = $2 AND space_id = $3", time.Now(), discardedID, spaceID); err != nil {
		t.Fatal(err)
	}
	_, tx := testutil.SetupTx(t)
	if err := NewPageRepository(query.New(db)).WithTx(tx).LockByIDs(t.Context(), []model.PageID{pageID, otherID, discardedID}, spaceID); err != nil {
		t.Fatalf("LockByIDs()のエラー = %v", err)
	}
	_, probe := testutil.SetupTx(t)
	for _, page := range []struct {
		id      model.PageID
		spaceID model.SpaceID
	}{{id: otherID, spaceID: otherSpaceID}, {id: discardedID, spaceID: spaceID}} {
		if _, err := probe.ExecContext(t.Context(), "SELECT id FROM pages WHERE id = $1 AND space_id = $2 FOR NO KEY UPDATE NOWAIT", page.id, page.spaceID); err != nil {
			t.Fatalf("対象外のページがロックされています: %v", err)
		}
	}
	_, err := probe.ExecContext(t.Context(), "SELECT id FROM pages WHERE id = $1 AND space_id = $2 FOR NO KEY UPDATE NOWAIT", pageID, spaceID)
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "55P03" {
		t.Errorf("対象ページのロックのエラー = %v、期待値 = lock_not_available", err)
	}
}
