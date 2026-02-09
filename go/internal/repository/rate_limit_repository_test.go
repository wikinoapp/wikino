package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestRateLimitRepository_Increment(t *testing.T) {
	t.Parallel()

	t.Run("新規レコードが作成される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := NewRateLimitRepository(q)

		input := IncrementInput{
			Key:         "test:new_record",
			WindowStart: time.Now().UTC().Truncate(time.Hour),
		}

		result, err := repo.Increment(context.Background(), input)
		if err != nil {
			t.Fatalf("Incrementでエラー: %v", err)
		}
		if result.Count != 1 {
			t.Errorf("Countが1であるべき: got %d", result.Count)
		}
	})

	t.Run("既存レコードがインクリメントされる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := NewRateLimitRepository(q)

		input := IncrementInput{
			Key:         "test:increment",
			WindowStart: time.Now().UTC().Truncate(time.Hour),
		}

		// 1回目
		result, err := repo.Increment(context.Background(), input)
		if err != nil {
			t.Fatalf("1回目のIncrementでエラー: %v", err)
		}
		if result.Count != 1 {
			t.Errorf("1回目のCountが1であるべき: got %d", result.Count)
		}

		// 2回目
		result, err = repo.Increment(context.Background(), input)
		if err != nil {
			t.Fatalf("2回目のIncrementでエラー: %v", err)
		}
		if result.Count != 2 {
			t.Errorf("2回目のCountが2であるべき: got %d", result.Count)
		}

		// 3回目
		result, err = repo.Increment(context.Background(), input)
		if err != nil {
			t.Fatalf("3回目のIncrementでエラー: %v", err)
		}
		if result.Count != 3 {
			t.Errorf("3回目のCountが3であるべき: got %d", result.Count)
		}
	})

	t.Run("異なるキーは別々にカウントされる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := NewRateLimitRepository(q)

		windowStart := time.Now().UTC().Truncate(time.Hour)

		// key1で2回インクリメント
		input1 := IncrementInput{
			Key:         "test:key1",
			WindowStart: windowStart,
		}
		for i := 0; i < 2; i++ {
			_, err := repo.Increment(context.Background(), input1)
			if err != nil {
				t.Fatalf("key1のインクリメントでエラー: %v", err)
			}
		}

		// key2で1回インクリメント
		input2 := IncrementInput{
			Key:         "test:key2",
			WindowStart: windowStart,
		}
		result, err := repo.Increment(context.Background(), input2)
		if err != nil {
			t.Fatalf("key2のインクリメントでエラー: %v", err)
		}
		if result.Count != 1 {
			t.Errorf("key2のCountが1であるべき: got %d", result.Count)
		}
	})

	t.Run("異なるウィンドウは別々にカウントされる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := NewRateLimitRepository(q)

		key := "test:different_window"

		// ウィンドウ1で2回インクリメント
		window1 := time.Now().UTC().Truncate(time.Hour)
		input1 := IncrementInput{
			Key:         key,
			WindowStart: window1,
		}
		for i := 0; i < 2; i++ {
			_, err := repo.Increment(context.Background(), input1)
			if err != nil {
				t.Fatalf("ウィンドウ1のインクリメントでエラー: %v", err)
			}
		}

		// ウィンドウ2で1回インクリメント
		window2 := window1.Add(-time.Hour)
		input2 := IncrementInput{
			Key:         key,
			WindowStart: window2,
		}
		result, err := repo.Increment(context.Background(), input2)
		if err != nil {
			t.Fatalf("ウィンドウ2のインクリメントでエラー: %v", err)
		}
		if result.Count != 1 {
			t.Errorf("ウィンドウ2のCountが1であるべき: got %d", result.Count)
		}
	})
}

func TestRateLimitRepository_DeleteOldRecords(t *testing.T) {
	t.Parallel()

	t.Run("古いレコードが削除される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := NewRateLimitRepository(q)

		// レコードを作成
		input := IncrementInput{
			Key:         "test:delete_old",
			WindowStart: time.Now().UTC().Truncate(time.Hour),
		}
		_, err := repo.Increment(context.Background(), input)
		if err != nil {
			t.Fatalf("Incrementでエラー: %v", err)
		}

		// 削除を実行（現在のレコードは削除されない）
		cutoff := time.Now().UTC().Add(-2 * time.Hour)
		err = repo.DeleteOldRecords(context.Background(), cutoff)
		if err != nil {
			t.Errorf("DeleteOldRecordsでエラー: %v", err)
		}
	})
}

func TestRateLimitRepository_WithTx(t *testing.T) {
	t.Parallel()

	t.Run("トランザクション付きリポジトリが正常に動作する", func(t *testing.T) {
		t.Parallel()

		db, tx := testutil.SetupTx(t)
		q := query.New(db)
		repo := NewRateLimitRepository(q)

		// WithTxでトランザクション付きリポジトリを作成
		repoWithTx := repo.WithTx(tx)

		input := IncrementInput{
			Key:         "test:with_tx",
			WindowStart: time.Now().UTC().Truncate(time.Hour),
		}

		result, err := repoWithTx.Increment(context.Background(), input)
		if err != nil {
			t.Fatalf("WithTxでのIncrementでエラー: %v", err)
		}
		if result.Count != 1 {
			t.Errorf("Countが1であるべき: got %d", result.Count)
		}
	})
}
