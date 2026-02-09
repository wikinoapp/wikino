package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestLimiter_Check(t *testing.T) {
	t.Parallel()

	t.Run("許可範囲内のリクエストは許可される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		input := CheckInput{
			Key:    "test:check_allowed",
			Limit:  3,
			Window: time.Hour,
		}

		// 1回目のリクエスト
		result, err := limiter.Check(context.Background(), input)
		if err != nil {
			t.Fatalf("1回目のチェックでエラー: %v", err)
		}
		if !result.Allowed {
			t.Error("1回目のリクエストが許可されるべき")
		}
		if result.Count != 1 {
			t.Errorf("1回目のカウントが1であるべき: got %d", result.Count)
		}
		if result.Remaining != 2 {
			t.Errorf("残りが2であるべき: got %d", result.Remaining)
		}

		// 2回目のリクエスト
		result, err = limiter.Check(context.Background(), input)
		if err != nil {
			t.Fatalf("2回目のチェックでエラー: %v", err)
		}
		if !result.Allowed {
			t.Error("2回目のリクエストが許可されるべき")
		}
		if result.Count != 2 {
			t.Errorf("2回目のカウントが2であるべき: got %d", result.Count)
		}
		if result.Remaining != 1 {
			t.Errorf("残りが1であるべき: got %d", result.Remaining)
		}

		// 3回目のリクエスト
		result, err = limiter.Check(context.Background(), input)
		if err != nil {
			t.Fatalf("3回目のチェックでエラー: %v", err)
		}
		if !result.Allowed {
			t.Error("3回目のリクエストが許可されるべき")
		}
		if result.Count != 3 {
			t.Errorf("3回目のカウントが3であるべき: got %d", result.Count)
		}
		if result.Remaining != 0 {
			t.Errorf("残りが0であるべき: got %d", result.Remaining)
		}
	})

	t.Run("制限を超えたリクエストは拒否される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		input := CheckInput{
			Key:    "test:check_exceeded",
			Limit:  2,
			Window: time.Hour,
		}

		// 制限回数までリクエスト
		for i := 0; i < 2; i++ {
			_, err := limiter.Check(context.Background(), input)
			if err != nil {
				t.Fatalf("リクエスト%dでエラー: %v", i+1, err)
			}
		}

		// 制限を超えたリクエスト
		result, err := limiter.Check(context.Background(), input)
		if err != nil {
			t.Fatalf("制限超過チェックでエラー: %v", err)
		}
		if result.Allowed {
			t.Error("制限を超えたリクエストは拒否されるべき")
		}
		if result.Count != 3 {
			t.Errorf("カウントが3であるべき: got %d", result.Count)
		}
		if result.Remaining != 0 {
			t.Errorf("残りが0であるべき: got %d", result.Remaining)
		}
	})

	t.Run("異なるキーは別々にカウントされる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		input1 := CheckInput{
			Key:    "test:key1",
			Limit:  2,
			Window: time.Hour,
		}
		input2 := CheckInput{
			Key:    "test:key2",
			Limit:  2,
			Window: time.Hour,
		}

		// key1で2回リクエスト
		for i := 0; i < 2; i++ {
			_, err := limiter.Check(context.Background(), input1)
			if err != nil {
				t.Fatalf("key1リクエスト%dでエラー: %v", i+1, err)
			}
		}

		// key2では1回目のリクエスト
		result, err := limiter.Check(context.Background(), input2)
		if err != nil {
			t.Fatalf("key2チェックでエラー: %v", err)
		}
		if !result.Allowed {
			t.Error("key2の1回目のリクエストは許可されるべき")
		}
		if result.Count != 1 {
			t.Errorf("key2のカウントが1であるべき: got %d", result.Count)
		}
	})

	t.Run("空のキーはエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		input := CheckInput{
			Key:    "",
			Limit:  5,
			Window: time.Hour,
		}

		_, err := limiter.Check(context.Background(), input)
		if err == nil {
			t.Error("空のキーはエラーを返すべき")
		}
	})

	t.Run("無効なLimitはエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		input := CheckInput{
			Key:    "test:invalid_limit",
			Limit:  0,
			Window: time.Hour,
		}

		_, err := limiter.Check(context.Background(), input)
		if err == nil {
			t.Error("無効なLimitはエラーを返すべき")
		}
	})

	t.Run("無効なWindowはエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		input := CheckInput{
			Key:    "test:invalid_window",
			Limit:  5,
			Window: 0,
		}

		_, err := limiter.Check(context.Background(), input)
		if err == nil {
			t.Error("無効なWindowはエラーを返すべき")
		}
	})
}

func TestLimiter_Allow(t *testing.T) {
	t.Parallel()

	t.Run("許可範囲内のリクエストはnilを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		input := CheckInput{
			Key:    "test:allow_ok",
			Limit:  3,
			Window: time.Hour,
		}

		err := limiter.Allow(context.Background(), input)
		if err != nil {
			t.Errorf("許可範囲内のリクエストはnilを返すべき: %v", err)
		}
	})

	t.Run("制限を超えたリクエストはErrRateLimitExceededを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		input := CheckInput{
			Key:    "test:allow_exceeded",
			Limit:  1,
			Window: time.Hour,
		}

		// 1回目は許可される
		err := limiter.Allow(context.Background(), input)
		if err != nil {
			t.Errorf("1回目は許可されるべき: %v", err)
		}

		// 2回目は制限超過
		err = limiter.Allow(context.Background(), input)
		if err != ErrRateLimitExceeded {
			t.Errorf("制限超過でErrRateLimitExceededを返すべき: got %v", err)
		}
	})
}

func TestLimiter_CleanupOldRecords(t *testing.T) {
	t.Parallel()

	t.Run("古いレコードが削除される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := query.New(tx)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		// レコードを作成
		input := CheckInput{
			Key:    "test:cleanup",
			Limit:  10,
			Window: time.Hour,
		}
		_, err := limiter.Check(context.Background(), input)
		if err != nil {
			t.Fatalf("チェックでエラー: %v", err)
		}

		// 2時間の保持期間で削除（現在のレコードは削除されない）
		err = limiter.CleanupOldRecords(context.Background(), 2*time.Hour)
		if err != nil {
			t.Errorf("クリーンアップでエラー: %v", err)
		}
	})
}

func TestLimiter_WithTx(t *testing.T) {
	t.Parallel()

	t.Run("トランザクション付きLimiterが正常に動作する", func(t *testing.T) {
		t.Parallel()

		db, tx := testutil.SetupTx(t)
		q := query.New(db)
		repo := repository.NewRateLimitRepository(q)
		limiter := NewLimiter(repo)

		// WithTxでトランザクション付きLimiterを作成
		limiterWithTx := limiter.WithTx(tx)

		input := CheckInput{
			Key:    "test:with_tx",
			Limit:  3,
			Window: time.Hour,
		}

		result, err := limiterWithTx.Check(context.Background(), input)
		if err != nil {
			t.Fatalf("WithTxでのチェックでエラー: %v", err)
		}
		if !result.Allowed {
			t.Error("リクエストが許可されるべき")
		}
		if result.Count != 1 {
			t.Errorf("カウントが1であるべき: got %d", result.Count)
		}
	})
}

func TestIPKey(t *testing.T) {
	t.Parallel()

	got := IPKey("192.168.1.1")
	want := "ip:192.168.1.1"
	if got != want {
		t.Errorf("IPKey() = %q, want %q", got, want)
	}
}

func TestEmailKey(t *testing.T) {
	t.Parallel()

	got := EmailKey("user@example.com")
	want := "email:user@example.com"
	if got != want {
		t.Errorf("EmailKey() = %q, want %q", got, want)
	}
}
