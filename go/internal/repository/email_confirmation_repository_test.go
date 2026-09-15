package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestEmailConfirmationRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewEmailConfirmationRepository(q)

	t.Run("メール確認情報を作成できる", func(t *testing.T) {
		now := time.Now()
		input := CreateEmailConfirmationInput{
			Email:     "create@example.com",
			Event:     model.EmailConfirmationEventSignUp,
			Code:      "ABC123",
			StartedAt: now,
		}

		ec, err := repo.Create(context.Background(), input)
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		if ec == nil {
			t.Fatal("Create()がnilを返した、期待値 = メールアドレス確認")
		}
		if ec.Email != "create@example.com" {
			t.Errorf("ec.Email = %v、期待値 = create@example.com", ec.Email)
		}
		if ec.Event != model.EmailConfirmationEventSignUp {
			t.Errorf("ec.Event = %v、期待値 = %v", ec.Event, model.EmailConfirmationEventSignUp)
		}
		if ec.Code != "ABC123" {
			t.Errorf("ec.Code = %v、期待値 = ABC123", ec.Code)
		}
		if ec.SucceededAt != nil {
			t.Errorf("ec.SucceededAt = %v、期待値 = nil", ec.SucceededAt)
		}
	})
}

func TestEmailConfirmationRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewEmailConfirmationRepository(q)

	// テストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("findbyid@example.com").
		WithCode("XYZ789").
		Build()

	t.Run("IDでメール確認情報を取得できる", func(t *testing.T) {
		ec, err := repo.FindByID(context.Background(), ecID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if ec == nil {
			t.Fatal("FindByID()がnilを返した、期待値 = メールアドレス確認")
		}
		if ec.ID != ecID {
			t.Errorf("ec.ID = %v、期待値 = %v", ec.ID, ecID)
		}
		if ec.Email != "findbyid@example.com" {
			t.Errorf("ec.Email = %v、期待値 = findbyid@example.com", ec.Email)
		}
		if ec.Code != "XYZ789" {
			t.Errorf("ec.Code = %v、期待値 = XYZ789", ec.Code)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		ec, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if ec != nil {
			t.Errorf("FindByID() = %v、期待値 = nil", ec)
		}
	})
}

func TestEmailConfirmationRepository_FindActiveByEmailAndEvent(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewEmailConfirmationRepository(q)

	// 有効なメール確認を作成 (15分以内)
	testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("active@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("ACTIVE1").
		WithStartedAt(time.Now()).
		Build()

	t.Run("有効なメール確認情報を取得できる", func(t *testing.T) {
		ec, err := repo.FindActiveByEmailAndEvent(context.Background(), "active@example.com", model.EmailConfirmationEventSignUp)
		if err != nil {
			t.Fatalf("FindActiveByEmailAndEvent()のエラー = %v", err)
		}
		if ec == nil {
			t.Fatal("FindActiveByEmailAndEvent()がnilを返した、期待値 = メールアドレス確認")
		}
		if ec.Email != "active@example.com" {
			t.Errorf("ec.Email = %v、期待値 = active@example.com", ec.Email)
		}
		if ec.Code != "ACTIVE1" {
			t.Errorf("ec.Code = %v、期待値 = ACTIVE1", ec.Code)
		}
	})

	t.Run("期限切れのメール確認はnilを返す", func(t *testing.T) {
		// 16分前のメール確認を作成 (期限切れ)
		testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("expired@example.com").
			WithEvent(model.EmailConfirmationEventSignUp).
			WithCode("EXPIRED").
			WithStartedAt(time.Now().Add(-16 * time.Minute)).
			Build()

		ec, err := repo.FindActiveByEmailAndEvent(context.Background(), "expired@example.com", model.EmailConfirmationEventSignUp)
		if err != nil {
			t.Fatalf("FindActiveByEmailAndEvent()のエラー = %v", err)
		}
		if ec != nil {
			t.Errorf("FindActiveByEmailAndEvent() = %v、期待値 = nil (期限切れ)", ec)
		}
	})

	t.Run("確認済みのメール確認はnilを返す", func(t *testing.T) {
		testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("succeeded@example.com").
			WithEvent(model.EmailConfirmationEventSignUp).
			WithCode("SUCCEED").
			WithStartedAt(time.Now()).
			BuildSucceeded()

		ec, err := repo.FindActiveByEmailAndEvent(context.Background(), "succeeded@example.com", model.EmailConfirmationEventSignUp)
		if err != nil {
			t.Fatalf("FindActiveByEmailAndEvent()のエラー = %v", err)
		}
		if ec != nil {
			t.Errorf("FindActiveByEmailAndEvent() = %v、期待値 = nil (確認済み)", ec)
		}
	})

	t.Run("異なるイベント種別はnilを返す", func(t *testing.T) {
		testutil.NewEmailConfirmationBuilder(t, tx).
			WithEmail("different-event@example.com").
			WithEvent(model.EmailConfirmationEventPasswordReset).
			WithCode("DIFEVT").
			WithStartedAt(time.Now()).
			Build()

		ec, err := repo.FindActiveByEmailAndEvent(context.Background(), "different-event@example.com", model.EmailConfirmationEventSignUp)
		if err != nil {
			t.Fatalf("FindActiveByEmailAndEvent()のエラー = %v", err)
		}
		if ec != nil {
			t.Errorf("FindActiveByEmailAndEvent() = %v、期待値 = nil (イベント種別が異なる)", ec)
		}
	})

	t.Run("存在しないメールアドレスはnilを返す", func(t *testing.T) {
		ec, err := repo.FindActiveByEmailAndEvent(context.Background(), "nonexistent@example.com", model.EmailConfirmationEventSignUp)
		if err != nil {
			t.Fatalf("FindActiveByEmailAndEvent()のエラー = %v", err)
		}
		if ec != nil {
			t.Errorf("FindActiveByEmailAndEvent() = %v、期待値 = nil", ec)
		}
	})
}

func TestEmailConfirmationRepository_Succeed(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewEmailConfirmationRepository(q)

	// テストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("succeed@example.com").
		WithCode("SUCC01").
		Build()

	t.Run("メール確認を完了状態に更新できる", func(t *testing.T) {
		err := repo.Succeed(context.Background(), ecID)
		if err != nil {
			t.Fatalf("Succeed()のエラー = %v", err)
		}

		// 更新後は取得できないことを確認 (succeeded_atがnot nullになるため)
		ec, err := repo.FindActiveByEmailAndEvent(context.Background(), "succeed@example.com", model.EmailConfirmationEventSignUp)
		if err != nil {
			t.Fatalf("FindActiveByEmailAndEvent()のエラー = %v", err)
		}
		if ec != nil {
			t.Errorf("Succeed()で確認済みになっていない: FindActiveByEmailAndEvent() = %v、期待値 = nil", ec)
		}

		// IDで取得すると、succeeded_atが設定されていることを確認
		ecByID, err := repo.FindByID(context.Background(), ecID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if ecByID == nil {
			t.Fatal("FindByID()がnilを返した、期待値 = メールアドレス確認")
		}
		if ecByID.SucceededAt == nil {
			t.Error("ecByID.SucceededAt = nil、期待値 = nilではない")
		}
	})
}

func TestEmailConfirmation_IsExpired(t *testing.T) {
	t.Parallel()

	t.Run("15分以内は期限切れではない", func(t *testing.T) {
		ec := &model.EmailConfirmation{
			StartedAt: time.Now().Add(-14 * time.Minute),
		}
		if ec.IsExpired() {
			t.Error("IsExpired() = true、期待値 = false (14分経過)")
		}
	})

	t.Run("15分を超えると期限切れ", func(t *testing.T) {
		ec := &model.EmailConfirmation{
			StartedAt: time.Now().Add(-16 * time.Minute),
		}
		if !ec.IsExpired() {
			t.Error("IsExpired() = false、期待値 = true (16分経過)")
		}
	})
}

func TestEmailConfirmation_IsSucceeded(t *testing.T) {
	t.Parallel()

	t.Run("SucceededAtがnilの場合は未完了", func(t *testing.T) {
		ec := &model.EmailConfirmation{
			SucceededAt: nil,
		}
		if ec.IsSucceeded() {
			t.Error("IsSucceeded() = true、期待値 = false")
		}
	})

	t.Run("SucceededAtが設定されている場合は完了", func(t *testing.T) {
		now := time.Now()
		ec := &model.EmailConfirmation{
			SucceededAt: &now,
		}
		if !ec.IsSucceeded() {
			t.Error("IsSucceeded() = false、期待値 = true")
		}
	})
}
