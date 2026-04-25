package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCleanupRateLimitsUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := query.New(tx)
	rateLimitRepo := repository.NewRateLimitRepository(q)
	limiter := ratelimit.NewLimiter(rateLimitRepo)
	uc := NewCleanupRateLimitsUsecase(limiter)

	ctx := context.Background()

	// Rate Limitレコードを作成
	_, err := limiter.Check(ctx, ratelimit.CheckInput{
		Key:    "test:cleanup_usecase",
		Limit:  10,
		Window: time.Hour,
	})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	// 正常に実行できること
	err = uc.Execute(ctx, CleanupRateLimitsInput{
		RetentionHours: 24,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestCleanupRateLimitsUsecase_Execute_DefaultRetention(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := query.New(tx)
	rateLimitRepo := repository.NewRateLimitRepository(q)
	limiter := ratelimit.NewLimiter(rateLimitRepo)
	uc := NewCleanupRateLimitsUsecase(limiter)

	ctx := context.Background()

	// RetentionHoursが0の場合、デフォルト24時間が使われること
	err := uc.Execute(ctx, CleanupRateLimitsInput{
		RetentionHours: 0,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
