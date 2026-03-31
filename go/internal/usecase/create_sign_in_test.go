package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestCreateSignInUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("正常系: サインインしてセッションを作成できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)

		password := "testpassword123"
		passwordDigest, err := auth.HashPassword(password)
		if err != nil {
			t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
		}

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("signin-uc@example.com").
			WithAtname("signinuc").
			BuildWithPassword(passwordDigest)

		userRepo := repository.NewUserRepository(q)
		userPasswordRepo := repository.NewUserPasswordRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

		signInValidator := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)
		uc := NewCreateSignInUsecase(signInValidator, userSessionRepo)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		output, err := uc.Execute(ctx, CreateSignInInput{
			Email:     "signin-uc@example.com",
			Password:  password,
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("Execute() returned nil, want output")
		}
		if output.Token == "" {
			t.Error("Execute() returned empty token")
		}
		if output.TwoFactorRequired {
			t.Error("Execute() returned TwoFactorRequired = true, want false")
		}
		if output.UserID != userID {
			t.Errorf("Execute() UserID = %v, want %v", output.UserID, userID)
		}

		// DBに保存されていることを確認
		session, err := userSessionRepo.FindByToken(ctx, output.Token)
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
		}
		if session == nil {
			t.Fatal("FindByToken() returned nil, want session")
		}
		if session.UserID != userID {
			t.Errorf("session.UserID = %v, want %v", session.UserID, userID)
		}
		if session.IPAddress != "192.168.1.1" {
			t.Errorf("session.IPAddress = %v, want 192.168.1.1", session.IPAddress)
		}
		if session.UserAgent != "Mozilla/5.0" {
			t.Errorf("session.UserAgent = %v, want Mozilla/5.0", session.UserAgent)
		}
	})

	t.Run("正常系: 各呼び出しで異なるトークンが生成される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)

		password := "testpassword123"
		passwordDigest, err := auth.HashPassword(password)
		if err != nil {
			t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
		}

		_ = testutil.NewUserBuilder(t, tx).
			WithEmail("signin-unique@example.com").
			WithAtname("signinunique").
			BuildWithPassword(passwordDigest)

		userRepo := repository.NewUserRepository(q)
		userPasswordRepo := repository.NewUserPasswordRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

		signInValidator := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)
		uc := NewCreateSignInUsecase(signInValidator, userSessionRepo)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		input := CreateSignInInput{
			Email:     "signin-unique@example.com",
			Password:  password,
			IPAddress: "192.168.1.2",
			UserAgent: "TestAgent",
		}

		output1, err := uc.Execute(ctx, input)
		if err != nil {
			t.Fatalf("Execute() first call error = %v", err)
		}

		output2, err := uc.Execute(ctx, input)
		if err != nil {
			t.Fatalf("Execute() second call error = %v", err)
		}

		if output1.Token == output2.Token {
			t.Error("Execute() returned same token for different calls")
		}
	})

	t.Run("異常系: バリデーションエラー", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)

		userRepo := repository.NewUserRepository(q)
		userPasswordRepo := repository.NewUserPasswordRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

		signInValidator := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)
		uc := NewCreateSignInUsecase(signInValidator, userSessionRepo)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		output, err := uc.Execute(ctx, CreateSignInInput{
			Email:     "",
			Password:  "",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})

		if output != nil {
			t.Error("expected nil output")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if !ve.HasFieldError("email") {
			t.Error("expected email field error")
		}
	})

	t.Run("異常系: ユーザーが見つからない場合", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)

		userRepo := repository.NewUserRepository(q)
		userPasswordRepo := repository.NewUserPasswordRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

		signInValidator := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)
		uc := NewCreateSignInUsecase(signInValidator, userSessionRepo)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		output, err := uc.Execute(ctx, CreateSignInInput{
			Email:     "nonexistent@example.com",
			Password:  "password123",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})

		if output != nil {
			t.Error("expected nil output")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if len(ve.Global) == 0 {
			t.Error("expected global error")
		}
	})

	t.Run("異常系: パスワードが正しくない場合", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)

		password := "testpassword123"
		passwordDigest, err := auth.HashPassword(password)
		if err != nil {
			t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
		}

		_ = testutil.NewUserBuilder(t, tx).
			WithEmail("signin-wrong@example.com").
			WithAtname("signinwrong").
			BuildWithPassword(passwordDigest)

		userRepo := repository.NewUserRepository(q)
		userPasswordRepo := repository.NewUserPasswordRepository(q)
		userSessionRepo := repository.NewUserSessionRepository(q)
		userTwoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(q)

		signInValidator := validator.NewSignInCreateValidator(userRepo, userPasswordRepo, userTwoFactorAuthRepo)
		uc := NewCreateSignInUsecase(signInValidator, userSessionRepo)

		ctx := context.Background()
		ctx = i18n.SetLocale(ctx, "ja")

		output, err := uc.Execute(ctx, CreateSignInInput{
			Email:     "signin-wrong@example.com",
			Password:  "wrongpassword",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})

		if output != nil {
			t.Error("expected nil output")
		}
		ve := model.AsValidationError(err)
		if ve == nil {
			t.Fatal("expected ValidationError, got nil")
		}
		if len(ve.Global) == 0 {
			t.Error("expected global error")
		}
	})
}
