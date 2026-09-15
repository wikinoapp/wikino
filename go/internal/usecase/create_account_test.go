package usecase

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestCreateAccountUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	userRepo := repository.NewUserRepository(q)
	userPasswordRepo := repository.NewUserPasswordRepository(q)
	createValidator := validator.NewAccountCreateValidator(userRepo)
	uc := NewCreateAccountUsecase(db, emailConfirmationRepo, userRepo, userPasswordRepo, createValidator)

	// メール確認完了済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("create-success@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("CA0001").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	// アカウントを作成
	output, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: string(ecID),
		Atname:              "createsuccessuser",
		Password:            "password123",
		Locale:              model.LocaleJa,
		TimeZone:            "Asia/Tokyo",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}
	if output.UserID == "" {
		t.Error("UserIDが空です")
	}

	// ユーザーが作成されたことを確認
	user, err := userRepo.FindByID(context.Background(), output.UserID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if user == nil {
		t.Fatal("ユーザーが見つかりません")
	}
	if user.Email != "create-success@example.com" {
		t.Errorf("Email = %v、期待値 = %v", user.Email, "create-success@example.com")
	}
	if user.Atname != "createsuccessuser" {
		t.Errorf("Atname = %v、期待値 = %v", user.Atname, "createsuccessuser")
	}
	if user.Locale != model.LocaleJa {
		t.Errorf("Locale = %v、期待値 = %v", user.Locale, model.LocaleJa)
	}
	if user.TimeZone != "Asia/Tokyo" {
		t.Errorf("TimeZone = %v、期待値 = %v", user.TimeZone, "Asia/Tokyo")
	}

	// パスワードが正しくハッシュ化されて保存されたことを確認
	userPassword, err := userPasswordRepo.FindByUserID(context.Background(), output.UserID)
	if err != nil {
		t.Fatalf("FindByUserID()のエラー = %v", err)
	}
	if userPassword == nil {
		t.Fatal("ユーザーパスワードが見つかりません")
	}
	// bcryptでハッシュ化されたパスワードを検証
	if err := bcrypt.CompareHashAndPassword([]byte(userPassword.PasswordDigest), []byte("password123")); err != nil {
		t.Errorf("パスワードが正しくハッシュ化されていません: %v", err)
	}
}

func TestCreateAccountUsecase_Execute_EnglishLocale(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	userRepo := repository.NewUserRepository(q)
	userPasswordRepo := repository.NewUserPasswordRepository(q)
	createValidator := validator.NewAccountCreateValidator(userRepo)
	uc := NewCreateAccountUsecase(db, emailConfirmationRepo, userRepo, userPasswordRepo, createValidator)

	// メール確認完了済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("english-user@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("CA0004").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	// 英語ロケールでアカウントを作成
	output, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: string(ecID),
		Atname:              "englishuser",
		Password:            "password123",
		Locale:              model.LocaleEn,
		TimeZone:            "America/New_York",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v、期待値 = nil", err)
	}

	// ユーザーが英語ロケールで作成されたことを確認
	user, err := userRepo.FindByID(context.Background(), output.UserID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if user.Locale != model.LocaleEn {
		t.Errorf("Locale = %v、期待値 = %v", user.Locale, model.LocaleEn)
	}
	if user.TimeZone != "America/New_York" {
		t.Errorf("TimeZone = %v、期待値 = %v", user.TimeZone, "America/New_York")
	}
}

func TestCreateAccountUsecase_Execute_EmailConfirmationNotFound(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	userRepo := repository.NewUserRepository(q)
	userPasswordRepo := repository.NewUserPasswordRepository(q)
	createValidator := validator.NewAccountCreateValidator(userRepo)
	uc := NewCreateAccountUsecase(db, emailConfirmationRepo, userRepo, userPasswordRepo, createValidator)

	_, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: "00000000-0000-0000-0000-000000000000",
		Atname:              "testuser",
		Password:            "password123",
		Locale:              model.LocaleJa,
		TimeZone:            "Asia/Tokyo",
	})

	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatal("AppErrorを期待したが、nilだった")
	}
	if ae.Code != model.AppErrCodeResourceNotFound {
		t.Errorf("Code = %v、期待値 = %v", ae.Code, model.AppErrCodeResourceNotFound)
	}
}

func TestCreateAccountUsecase_Execute_EmailNotConfirmed(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	userRepo := repository.NewUserRepository(q)
	userPasswordRepo := repository.NewUserPasswordRepository(q)
	createValidator := validator.NewAccountCreateValidator(userRepo)
	uc := NewCreateAccountUsecase(db, emailConfirmationRepo, userRepo, userPasswordRepo, createValidator)

	// メール確認未完了のテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("unconfirmed@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("CA0005").
		WithStartedAt(time.Now()).
		Build()

	_, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: string(ecID),
		Atname:              "testuser",
		Password:            "password123",
		Locale:              model.LocaleJa,
		TimeZone:            "Asia/Tokyo",
	})

	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatal("AppErrorを期待したが、nilだった")
	}
	if ae.Code != model.AppErrCodeConflict {
		t.Errorf("Code = %v、期待値 = %v", ae.Code, model.AppErrCodeConflict)
	}
}

func TestCreateAccountUsecase_Execute_ValidationError(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	q := query.New(db)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	userRepo := repository.NewUserRepository(q)
	userPasswordRepo := repository.NewUserPasswordRepository(q)
	createValidator := validator.NewAccountCreateValidator(userRepo)
	uc := NewCreateAccountUsecase(db, emailConfirmationRepo, userRepo, userPasswordRepo, createValidator)

	// メール確認完了済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("validation-test@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("CA0006").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	_, err := uc.Execute(context.Background(), CreateAccountInput{
		EmailConfirmationID: string(ecID),
		Atname:              "",
		Password:            "password123",
		Locale:              model.LocaleJa,
		TimeZone:            "Asia/Tokyo",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("ValidationErrorを期待したが、nilだった")
	}
	if !ve.HasFieldError("atname") {
		t.Error("atnameのフィールドエラーが無い")
	}
}

func TestHashPassword(t *testing.T) {
	t.Parallel()

	password := "testpassword123"

	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword()のエラー = %v", err)
	}

	// ハッシュが空でないことを確認
	if hash == "" {
		t.Error("hashPassword()が空文字列を返した")
	}

	// ハッシュが元のパスワードと異なることを確認
	if hash == password {
		t.Error("hashPassword()がパスワードと同じ文字列を返した")
	}

	// bcryptで検証できることを確認
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Errorf("bcrypt.CompareHashAndPassword()のエラー = %v", err)
	}

	// 間違ったパスワードで検証が失敗することを確認
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrongpassword")); err == nil {
		t.Error("誤ったパスワードなのにbcrypt.CompareHashAndPassword()が失敗しなかった")
	}
}
