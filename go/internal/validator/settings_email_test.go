package validator_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

const settingsEmailPassword = "password123"

// settingsEmailUserはパスワードを持つユーザーを作る
func settingsEmailUser(t *testing.T, tx *sql.Tx, atname string) model.UserID {
	t.Helper()

	digest, err := auth.HashPassword(settingsEmailPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	return testutil.NewUserBuilder(t, tx).
		WithEmail(atname + "@example.com").
		WithAtname(atname).
		BuildWithPassword(digest)
}

func TestSettingsEmailUpdateValidator_Validate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		newEmail   string
		password   string
		wantFields []string
	}{
		{name: "正しい入力", newEmail: "settings-email-new@example.com", password: settingsEmailPassword},
		{name: "新しいメールアドレスが空", newEmail: "", password: settingsEmailPassword, wantFields: []string{"new_email"}},
		{name: "新しいメールアドレスの形式が誤り", newEmail: "invalid-email", password: settingsEmailPassword, wantFields: []string{"new_email"}},
		{name: "表示名付きのメールアドレス", newEmail: "Name <settings-email-named@example.com>", password: settingsEmailPassword, wantFields: []string{"new_email"}},
		{name: "パスワードが空", newEmail: "settings-email-new@example.com", password: "", wantFields: []string{"password"}},
		{name: "パスワードが違う", newEmail: "settings-email-new@example.com", password: "wrong-password", wantFields: []string{"password"}},
		{name: "登録済みのメールアドレス", newEmail: "settings_email_taken@example.com", password: settingsEmailPassword, wantFields: []string{"new_email"}},
		{name: "自分の今のメールアドレス", newEmail: "settings_email_self@example.com", password: settingsEmailPassword, wantFields: []string{"new_email"}},
		{name: "パスワードが違うときは登録済みかを示さない", newEmail: "settings_email_taken@example.com", password: "wrong-password", wantFields: []string{"password"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			userID := settingsEmailUser(t, tx, "settings_email_self")
			testutil.NewUserBuilder(t, tx).
				WithEmail("settings_email_taken@example.com").
				WithAtname("settings_email_taken").
				Build()

			v := validator.NewSettingsEmailUpdateValidator(repository.NewUserRepository(q), repository.NewUserPasswordRepository(q))
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			err := v.Validate(ctx, validator.SettingsEmailUpdateValidatorInput{
				UserID:   userID,
				NewEmail: tc.newEmail,
				Password: tc.password,
			})

			if len(tc.wantFields) == 0 {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				return
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、ValidationErrorを期待", err)
			}
			for _, field := range []string{"new_email", "password"} {
				want := false
				for _, f := range tc.wantFields {
					want = want || f == field
				}
				if got := ve.HasFieldError(field); got != want {
					t.Errorf("%qのフィールドエラーがあるか = %v、期待値 = %v (%v)", field, got, want, ve.Fields)
				}
			}
		})
	}
}

func TestSettingsEmailConfirmationUpdateValidator_Validate(t *testing.T) {
	t.Parallel()

	succeededAt := time.Now()
	testCases := []struct {
		name        string
		event       model.EmailConfirmationEvent
		startedAt   time.Time
		succeededAt *time.Time
		email       string
		code        string
		wantField   string
		wantGlobal  bool
		ownerless   bool
		otherUser   bool
		invalidID   bool
	}{
		{name: "所有者のない既存の確認", event: model.EmailConfirmationEventEmailUpdate, code: "ABC123", ownerless: true, wantGlobal: true},
		{name: "別のユーザーの確認", event: model.EmailConfirmationEventEmailUpdate, code: "ABC123", otherUser: true, wantGlobal: true},
		{name: "不正な確認ID", event: model.EmailConfirmationEventEmailUpdate, code: "ABC123", invalidID: true, wantGlobal: true},
		{name: "パスワードリセットの確認では確定できない", event: model.EmailConfirmationEventPasswordReset, code: "ABC123", wantGlobal: true},
		{name: "正しい確認コード", event: model.EmailConfirmationEventEmailUpdate, code: "ABC123"},
		{name: "大文字小文字を区別しない", event: model.EmailConfirmationEventEmailUpdate, code: "abc123"},
		{name: "確認コードが空", event: model.EmailConfirmationEventEmailUpdate, code: "", wantField: "code"},
		{name: "確認コードの長さが誤り", event: model.EmailConfirmationEventEmailUpdate, code: "ABC", wantField: "code"},
		{name: "確認コードが違う", event: model.EmailConfirmationEventEmailUpdate, code: "XYZ999", wantField: "code"},
		{name: "期限切れ", event: model.EmailConfirmationEventEmailUpdate, startedAt: time.Now().Add(-16 * time.Minute), code: "ABC123", wantGlobal: true},
		{name: "新規登録の確認では確定できない", event: model.EmailConfirmationEventSignUp, code: "ABC123", wantGlobal: true},
		{name: "完了済みの確認", event: model.EmailConfirmationEventEmailUpdate, succeededAt: &succeededAt, code: "ABC123", wantGlobal: true},
		{name: "待つ間にほかのユーザーが登録したメールアドレス", event: model.EmailConfirmationEventEmailUpdate, email: "settings_confirm_taken@example.com", code: "ABC123", wantGlobal: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			testutil.NewUserBuilder(t, tx).
				WithEmail("settings_confirm_taken@example.com").
				WithAtname("settings_confirm_taken").
				Build()

			userID := testutil.NewUserBuilder(t, tx).WithAtname("settings_confirm_self").WithEmail("settings-confirm-self@example.com").Build()

			email := tc.email
			if email == "" {
				email = "settings-confirm-new@example.com"
			}
			startedAt := tc.startedAt
			if startedAt.IsZero() {
				startedAt = time.Now()
			}
			builder := testutil.NewEmailConfirmationBuilder(t, tx).
				WithEmail(email).
				WithEvent(tc.event).
				WithCode("ABC123").
				WithStartedAt(startedAt)
			if !tc.ownerless {
				builder = builder.WithUserID(userID)
			}
			if tc.succeededAt != nil {
				builder = builder.WithSucceededAt(*tc.succeededAt)
			}
			confirmationID := builder.Build()
			if tc.otherUser {
				userID = testutil.NewUserBuilder(t, tx).WithAtname("settings_confirm_b").WithEmail("settings-confirm-b@example.com").Build()
			}
			if tc.invalidID {
				confirmationID = "invalid-id"
			}

			v := validator.NewSettingsEmailConfirmationUpdateValidator(repository.NewEmailConfirmationRepository(q), repository.NewUserRepository(q))
			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			confirmation, err := v.Validate(ctx, validator.SettingsEmailConfirmationUpdateValidatorInput{
				UserID:              userID,
				EmailConfirmationID: confirmationID,
				Code:                tc.code,
			})

			if tc.wantField == "" && !tc.wantGlobal {
				if err != nil {
					t.Fatalf("Validate()のエラー = %v", err)
				}
				if confirmation == nil || confirmation.ID != confirmationID {
					t.Errorf("確認 = %+v、ID = %q の確認を期待", confirmation, confirmationID)
				}
				return
			}
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、ValidationErrorを期待", err)
			}
			if tc.wantField != "" && !ve.HasFieldError(tc.wantField) {
				t.Errorf("%qのフィールドエラーが無い (%v)", tc.wantField, ve.Fields)
			}
			if tc.wantGlobal && len(ve.Global) == 0 {
				t.Errorf("フォーム全体のエラーが無い (%v)", ve.Fields)
			}
		})
	}
}

func TestSettingsEmailConfirmationUpdateValidator_Validate_存在しない確認(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	v := validator.NewSettingsEmailConfirmationUpdateValidator(repository.NewEmailConfirmationRepository(q), repository.NewUserRepository(q))

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	_, err := v.Validate(ctx, validator.SettingsEmailConfirmationUpdateValidatorInput{
		EmailConfirmationID: "00000000-0000-0000-0000-000000000000",
		Code:                "ABC123",
	})
	ve := model.AsValidationError(err)
	if ve == nil || len(ve.Global) == 0 {
		t.Errorf("エラー = %v、フォーム全体のValidationErrorを期待", err)
	}
}
