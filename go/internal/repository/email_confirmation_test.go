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
		if ec.UserID != nil {
			t.Errorf("新規登録の確認のUserID = %v、期待値 = nil", ec.UserID)
		}
	})
}

func TestEmailConfirmationRepository_開始ユーザーの保存と削除(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
	userID := testutil.NewUserBuilder(t, tx).Build()
	owned, err := repo.Create(ctx, CreateEmailConfirmationInput{
		UserID: &userID, Email: "owned@example.com", Event: model.EmailConfirmationEventEmailUpdate,
		Code: "OWN001", StartedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	unownedID := testutil.NewEmailConfirmationBuilder(t, tx).WithCode("OWN002").Build()
	confirmation, err := repo.FindByID(ctx, owned.ID)
	if err != nil || confirmation == nil || confirmation.UserID == nil || *confirmation.UserID != userID {
		t.Fatalf("開始ユーザーが保存されていない: 確認 = %v、エラー = %v", confirmation, err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = $1", string(userID)); err != nil {
		t.Fatal(err)
	}
	confirmation, err = repo.FindByID(ctx, owned.ID)
	if err != nil || confirmation != nil {
		t.Fatalf("削除したユーザーの確認が残っている: 確認 = %v、エラー = %v", confirmation, err)
	}
	unowned, err := repo.FindByID(ctx, unownedID)
	if err != nil || unowned == nil {
		t.Fatalf("所有者のない確認が削除された: エラー = %v", err)
	}
}

func TestEmailConfirmationRepository_ConsumeEmailUpdate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		event     model.EmailConfirmationEvent
		expired   bool
		ownerless bool
		otherUser bool
		code      string
		want      bool
	}{
		{name: "正しい確認を一度だけ消費する", event: model.EmailConfirmationEventEmailUpdate, code: "abc123", want: true},
		{name: "別ユーザー", event: model.EmailConfirmationEventEmailUpdate, code: "ABC123", otherUser: true},
		{name: "所有者なし", event: model.EmailConfirmationEventEmailUpdate, code: "ABC123", ownerless: true},
		{name: "期限切れ", event: model.EmailConfirmationEventEmailUpdate, code: "ABC123", expired: true},
		{name: "コード違い", event: model.EmailConfirmationEventEmailUpdate, code: "WRONG1"},
		{name: "新規登録", event: model.EmailConfirmationEventSignUp, code: "ABC123"},
		{name: "パスワードリセット", event: model.EmailConfirmationEventPasswordReset, code: "ABC123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, tx := testutil.SetupTx(t)
			ctx := context.Background()
			repo := NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
			userID := testutil.NewUserBuilder(t, tx).Build()
			builder := testutil.NewEmailConfirmationBuilder(t, tx).WithEvent(tc.event).WithCode("ABC123")
			if !tc.ownerless {
				builder = builder.WithUserID(userID)
			}
			if tc.expired {
				builder = builder.WithStartedAt(time.Now().Add(-16 * time.Minute))
			}
			id := builder.Build()
			if tc.otherUser {
				userID = testutil.NewUserBuilder(t, tx).WithAtname("consume_other").WithEmail("consume-other@example.com").Build()
			}
			confirmation, err := repo.ConsumeEmailUpdate(ctx, id, userID, tc.code)
			if err != nil || (confirmation != nil) != tc.want {
				t.Fatalf("ConsumeEmailUpdate() = %v、エラー = %v、成功の期待値 = %v", confirmation, err, tc.want)
			}
			if tc.want {
				if !confirmation.IsSucceeded() {
					t.Error("消費した確認が完了状態になっていない")
				}
				if again, err := repo.ConsumeEmailUpdate(ctx, id, userID, tc.code); err != nil || again != nil {
					t.Fatalf("確認を再利用できた: %v、エラー = %v", again, err)
				}
			} else {
				stored, err := repo.FindByID(ctx, id)
				if err != nil || stored == nil || stored.IsSucceeded() {
					t.Fatalf("拒否した確認の状態が変わった: %v、エラー = %v", stored, err)
				}
			}
		})
	}
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

func TestEmailConfirmationRepository_DeleteStartedBefore(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewEmailConfirmationRepository(q)

	// 他のテストが作る行を巻き込まないよう、どのテストも使わない過去の日時で作る
	cutoff := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)
	unsucceededID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("delete-unsucceeded@example.com").
		WithCode("DSB001").
		WithStartedAt(cutoff.Add(-time.Hour)).
		Build()
	succeededID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("delete-succeeded@example.com").
		WithCode("DSB002").
		WithStartedAt(cutoff.Add(-time.Hour)).
		WithSucceededAt(cutoff.Add(-time.Hour + time.Minute)).
		Build()
	atCutoffID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("delete-at-cutoff@example.com").
		WithCode("DSB003").
		WithStartedAt(cutoff).
		Build()

	deletedCount, err := repo.DeleteStartedBefore(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("DeleteStartedBefore()のエラー = %v", err)
	}
	if deletedCount != 2 {
		t.Errorf("削除した件数 = %d、期待値 = 2", deletedCount)
	}

	tests := []struct {
		name        string
		id          string
		wantDeleted bool
	}{
		{name: "cutoffより前に開始した未確認の行は削除される", id: unsucceededID, wantDeleted: true},
		{name: "cutoffより前に開始した確認済みの行は削除される", id: succeededID, wantDeleted: true},
		{name: "cutoffちょうどに開始した行は残る", id: atCutoffID, wantDeleted: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec, err := repo.FindByID(context.Background(), tt.id)
			if err != nil {
				t.Fatalf("FindByID()のエラー = %v", err)
			}
			if deleted := ec == nil; deleted != tt.wantDeleted {
				t.Errorf("削除されたか = %v、期待値 = %v", deleted, tt.wantDeleted)
			}
		})
	}
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
