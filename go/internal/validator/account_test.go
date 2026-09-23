package validator_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestAccountCreateValidator_Validate_FormatValidation(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := query.New(db).WithTx(tx)
	userRepo := repository.NewUserRepository(queries)

	v := validator.NewAccountCreateValidator(userRepo)

	tests := []struct {
		name          string
		atname        string
		password      string
		wantErrors    bool
		expectedField string
	}{
		{
			name:       "正しいリクエスト",
			atname:     "testuser",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:          "空のアットネーム",
			atname:        "",
			password:      "password123",
			wantErrors:    true,
			expectedField: "atname",
		},
		{
			name:          "空のパスワード",
			atname:        "testuser",
			password:      "",
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:       "両方とも空",
			atname:     "",
			password:   "",
			wantErrors: true,
		},
		{
			name:          "アットネームが長すぎる",
			atname:        "verylongusernameover20",
			password:      "password123",
			wantErrors:    true,
			expectedField: "atname",
		},
		{
			name:          "不正な文字を含むアットネーム",
			atname:        "test-user!@",
			password:      "password123",
			wantErrors:    true,
			expectedField: "atname",
		},
		{
			name:          "パスワードが短すぎる",
			atname:        "testuser",
			password:      "short",
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:          "パスワードが長すぎる",
			atname:        "testuser",
			password:      strings.Repeat("a", 129),
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:          "マルチバイト文字を含むパスワード",
			atname:        "testuser",
			password:      "パスワード123",
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:          "スペースを含むパスワード",
			atname:        "testuser",
			password:      "pass word",
			wantErrors:    true,
			expectedField: "password",
		},
		{
			name:       "アンダースコアを含むアットネーム",
			atname:     "test_user",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:       "数字を含むアットネーム",
			atname:     "user123",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:       "アットネームがちょうど20文字",
			atname:     "12345678901234567890",
			password:   "password123",
			wantErrors: false,
		},
		{
			name:       "パスワードがちょうど8文字",
			atname:     "testuser8",
			password:   "12345678",
			wantErrors: false,
		},
		{
			name:       "パスワードがちょうど128文字",
			atname:     "testuser",
			password:   strings.Repeat("a", 128),
			wantErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = i18n.SetLocale(ctx, i18n.LangJa)

			err := v.Validate(ctx, validator.AccountCreateValidatorInput{
				Atname:   tt.atname,
				Password: tt.password,
			})

			if tt.wantErrors {
				ve := model.AsValidationError(err)
				if ve == nil {
					t.Error("ValidationErrorを期待したが、nilだった")
					return
				}
				if !ve.HasErrors() {
					t.Error("エラーが無い")
				}
				if tt.expectedField != "" && !ve.HasFieldError(tt.expectedField) {
					t.Errorf("%qのフィールドエラーが無い", tt.expectedField)
				}
			} else {
				if err != nil {
					t.Errorf("予期しないエラー: %v", err)
				}
			}
		})
	}
}

func TestAccountCreateValidator_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := query.New(db).WithTx(tx)
	userRepo := repository.NewUserRepository(queries)

	v := validator.NewAccountCreateValidator(userRepo)

	tests := []struct {
		name            string
		atname          string
		password        string
		locale          string
		expectedMessage string
	}{
		{
			name:            "アットネームが未入力 (ja)",
			atname:          "",
			password:        "password123",
			locale:          "ja",
			expectedMessage: "アットネームを入力してください",
		},
		{
			name:            "アットネームが未入力 (en)",
			atname:          "",
			password:        "password123",
			locale:          "en",
			expectedMessage: "Please enter a username",
		},
		{
			name:            "アットネームが長すぎる (ja)",
			atname:          "verylongusernameover20",
			password:        "password123",
			locale:          "ja",
			expectedMessage: "アットネームは20文字以内で入力してください",
		},
		{
			name:            "アットネームが長すぎる (en)",
			atname:          "verylongusernameover20",
			password:        "password123",
			locale:          "en",
			expectedMessage: "Username must be 20 characters or less",
		},
		{
			name:            "アットネームの形式が不正 (ja)",
			atname:          "test-user!",
			password:        "password123",
			locale:          "ja",
			expectedMessage: "アットネームは英数字とアンダースコアのみ使用できます",
		},
		{
			name:            "アットネームの形式が不正 (en)",
			atname:          "test-user!",
			password:        "password123",
			locale:          "en",
			expectedMessage: "Username can only contain letters, numbers, and underscores",
		},
		{
			name:            "パスワードが未入力 (ja)",
			atname:          "testuser",
			password:        "",
			locale:          "ja",
			expectedMessage: "パスワードを入力してください",
		},
		{
			name:            "パスワードが未入力 (en)",
			atname:          "testuser",
			password:        "",
			locale:          "en",
			expectedMessage: "Please enter a password",
		},
		{
			name:            "パスワードが短すぎる (ja)",
			atname:          "testuser",
			password:        "short",
			locale:          "ja",
			expectedMessage: "パスワードは8文字以上で入力してください",
		},
		{
			name:            "パスワードが短すぎる (en)",
			atname:          "testuser",
			password:        "short",
			locale:          "en",
			expectedMessage: "Password must be at least 8 characters",
		},
		{
			name:            "パスワードが長すぎる (ja)",
			atname:          "testuser",
			password:        strings.Repeat("a", 129),
			locale:          "ja",
			expectedMessage: "パスワードは128文字以内で入力してください",
		},
		{
			name:            "パスワードが長すぎる (en)",
			atname:          "testuser",
			password:        strings.Repeat("a", 129),
			locale:          "en",
			expectedMessage: "Password must be at most 128 characters",
		},
		{
			name:            "パスワードに不正な文字 (ja)",
			atname:          "testuser",
			password:        "パスワード123",
			locale:          "ja",
			expectedMessage: "パスワードには印字可能なASCII文字のみ使用できます",
		},
		{
			name:            "パスワードに不正な文字 (en)",
			atname:          "testuser",
			password:        "パスワード123",
			locale:          "en",
			expectedMessage: "Password can only contain printable ASCII characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.locale == "ja" {
				ctx = i18n.SetLocale(ctx, i18n.LangJa)
			} else {
				ctx = i18n.SetLocale(ctx, i18n.LangEn)
			}

			err := v.Validate(ctx, validator.AccountCreateValidatorInput{
				Atname:   tt.atname,
				Password: tt.password,
			})

			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatal("ValidationErrorを期待したが、nilだった")
			}

			// エラーメッセージが含まれているか確認
			found := false
			for _, messages := range ve.Fields {
				for _, msg := range messages {
					if strings.Contains(msg, tt.expectedMessage) {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("エラーにメッセージ%qが見つからない", tt.expectedMessage)
			}
		})
	}
}

func TestAccountCreateValidator_Validate_Success(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := query.New(db).WithTx(tx)
	userRepo := repository.NewUserRepository(queries)

	v := validator.NewAccountCreateValidator(userRepo)

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	err := v.Validate(ctx, validator.AccountCreateValidatorInput{
		Atname:   "newuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}
}

func TestAccountCreateValidator_Validate_AtnameAlreadyTaken(t *testing.T) {
	t.Parallel()

	db, tx := testutil.SetupTx(t)
	queries := query.New(db).WithTx(tx)
	userRepo := repository.NewUserRepository(queries)

	// 既存のユーザーを作成
	now := time.Now()
	_, err := userRepo.Create(t.Context(), repository.CreateUserInput{
		Email:       "existing@example.com",
		Atname:      "existinguser",
		Name:        "",
		Description: "",
		Locale:      model.LocaleJa,
		TimeZone:    "Asia/Tokyo",
		JoinedAt:    now,
	})
	if err != nil {
		t.Fatalf("既存ユーザーの作成に失敗: %v", err)
	}

	v := validator.NewAccountCreateValidator(userRepo)

	ctx := i18n.SetLocale(t.Context(), i18n.LangJa)

	err = v.Validate(ctx, validator.AccountCreateValidatorInput{
		Atname:   "existinguser",
		Password: "password123",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("ValidationErrorを期待したが、nilだった")
	}
	if !ve.HasErrors() {
		t.Error("フォームのエラーにエラーが含まれていない")
	}
	if !ve.HasFieldError("atname") {
		t.Error("atnameのフィールドエラーが無い")
	}
}

func TestIsValidAtname(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		atname string
		want   bool
	}{
		{name: "英数字とアンダースコア", atname: "seed_user1", want: true},
		{name: "大文字を含む", atname: "SeedUser1", want: true},
		{name: "上限ちょうど", atname: strings.Repeat("a", validator.AtnameMaxLength), want: true},
		{name: "上限を1文字超える", atname: strings.Repeat("a", validator.AtnameMaxLength+1), want: false},
		{name: "空文字列", atname: "", want: false},
		{name: "ハイフンを含む", atname: "seed-user1", want: false},
		{name: "空白を含む", atname: "seed user1", want: false},
		{name: "日本語を含む", atname: "シードユーザー", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := validator.IsValidAtname(tt.atname); got != tt.want {
				t.Errorf("IsValidAtname(%q) = %vであることを期待したが%vだった", tt.atname, tt.want, got)
			}
		})
	}
}
