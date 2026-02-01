package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreateAccountUsecase_Execute_Success(t *testing.T) {
	db := testutil.SetupTestDBWithoutTx(t)
	uc := NewCreateAccountUsecase(db)

	// メール確認完了済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("create-success@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("CA0001").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	// アカウントを作成
	output, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: ecID,
		Email:               "create-success@example.com",
		Atname:              "createsuccessuser",
		Password:            "password123",
		Locale:              model.LocaleJa,
		TimeZone:            "Asia/Tokyo",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if output.UserID == "" {
		t.Error("UserIDが空です")
	}

	// ユーザーが作成されたことを確認
	q := query.New(db)
	userRepo := repository.NewUserRepository(q)
	user, err := userRepo.FindByID(context.Background(), output.UserID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if user == nil {
		t.Fatal("ユーザーが見つかりません")
	}
	if user.Email != "create-success@example.com" {
		t.Errorf("Email = %v, want %v", user.Email, "create-success@example.com")
	}
	if user.Atname != "createsuccessuser" {
		t.Errorf("Atname = %v, want %v", user.Atname, "createsuccessuser")
	}
	if user.Locale != model.LocaleJa {
		t.Errorf("Locale = %v, want %v", user.Locale, model.LocaleJa)
	}
	if user.TimeZone != "Asia/Tokyo" {
		t.Errorf("TimeZone = %v, want %v", user.TimeZone, "Asia/Tokyo")
	}

	// パスワードが正しくハッシュ化されて保存されたことを確認
	userPasswordRepo := repository.NewUserPasswordRepository(q)
	userPassword, err := userPasswordRepo.FindByUserID(context.Background(), output.UserID)
	if err != nil {
		t.Fatalf("FindByUserID() error = %v", err)
	}
	if userPassword == nil {
		t.Fatal("ユーザーパスワードが見つかりません")
	}
	// bcryptでハッシュ化されたパスワードを検証
	if err := bcrypt.CompareHashAndPassword([]byte(userPassword.PasswordDigest), []byte("password123")); err != nil {
		t.Errorf("パスワードが正しくハッシュ化されていません: %v", err)
	}
}

func TestCreateAccountUsecase_Execute_EmailNotConfirmed(t *testing.T) {
	db := testutil.SetupTestDBWithoutTx(t)
	uc := NewCreateAccountUsecase(db)

	// メール確認が未完了のテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("not-confirmed@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("CA0002").
		WithStartedAt(time.Now()).
		Build() // BuildSucceeded()ではなくBuild()を使用

	// アカウント作成を試みるとエラーになる
	_, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: ecID,
		Email:               "not-confirmed@example.com",
		Atname:              "notconfirmeduser",
		Password:            "password123",
		Locale:              model.LocaleJa,
		TimeZone:            "Asia/Tokyo",
	})
	if !errors.Is(err, ErrEmailNotConfirmed) {
		t.Errorf("Execute() error = %v, want %v", err, ErrEmailNotConfirmed)
	}
}

func TestCreateAccountUsecase_Execute_EmailConfirmationNotFound(t *testing.T) {
	db := testutil.SetupTestDBWithoutTx(t)
	uc := NewCreateAccountUsecase(db)

	// 存在しないEmailConfirmationIDでアカウント作成を試みる
	_, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: "00000000-0000-0000-0000-000000000000",
		Email:               "notfound@example.com",
		Atname:              "notfounduser",
		Password:            "password123",
		Locale:              model.LocaleJa,
		TimeZone:            "Asia/Tokyo",
	})
	if !errors.Is(err, ErrEmailNotConfirmed) {
		t.Errorf("Execute() error = %v, want %v", err, ErrEmailNotConfirmed)
	}
}

func TestCreateAccountUsecase_Execute_AtnameAlreadyTaken(t *testing.T) {
	db := testutil.SetupTestDBWithoutTx(t)
	uc := NewCreateAccountUsecase(db)

	// 既存のユーザーを作成
	testutil.NewUserBuilderDB(t, db).
		WithEmail("existing@example.com").
		WithAtname("existinguser").
		Build()

	// メール確認完了済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("new@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("CA0003").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	// 既存のアットネームでアカウント作成を試みるとエラーになる
	_, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: ecID,
		Email:               "new@example.com",
		Atname:              "existinguser", // 既存のアットネーム
		Password:            "password123",
		Locale:              model.LocaleJa,
		TimeZone:            "Asia/Tokyo",
	})
	if !errors.Is(err, ErrAtnameAlreadyTaken) {
		t.Errorf("Execute() error = %v, want %v", err, ErrAtnameAlreadyTaken)
	}
}

func TestCreateAccountUsecase_Execute_EnglishLocale(t *testing.T) {
	db := testutil.SetupTestDBWithoutTx(t)
	uc := NewCreateAccountUsecase(db)

	// メール確認完了済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("english-user@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("CA0004").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	// 英語ロケールでアカウントを作成
	output, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: ecID,
		Email:               "english-user@example.com",
		Atname:              "englishuser",
		Password:            "password123",
		Locale:              model.LocaleEn,
		TimeZone:            "America/New_York",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// ユーザーが英語ロケールで作成されたことを確認
	q := query.New(db)
	userRepo := repository.NewUserRepository(q)
	user, err := userRepo.FindByID(context.Background(), output.UserID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if user.Locale != model.LocaleEn {
		t.Errorf("Locale = %v, want %v", user.Locale, model.LocaleEn)
	}
	if user.TimeZone != "America/New_York" {
		t.Errorf("TimeZone = %v, want %v", user.TimeZone, "America/New_York")
	}
}

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword() error = %v", err)
	}

	// ハッシュが空でないことを確認
	if hash == "" {
		t.Error("hashPassword() returned empty string")
	}

	// ハッシュが元のパスワードと異なることを確認
	if hash == password {
		t.Error("hashPassword() returned the same string as password")
	}

	// bcryptで検証できることを確認
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Errorf("bcrypt.CompareHashAndPassword() error = %v", err)
	}

	// 間違ったパスワードで検証が失敗することを確認
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrongpassword")); err == nil {
		t.Error("bcrypt.CompareHashAndPassword() should fail with wrong password")
	}
}
