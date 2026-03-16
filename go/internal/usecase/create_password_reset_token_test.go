package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/password_reset"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/worker"
)

func TestCreatePasswordResetTokenUsecase_Execute(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	// テストユーザーを作成
	_ = testutil.NewUserBuilderDB(t, db).
		WithEmail("reset-test@example.com").
		WithAtname("reset_test_user").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	userRepo := repository.NewUserRepository(q)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(q)
	inserter := &mockInserter{}
	uc := NewCreatePasswordResetTokenUsecase(cfg, db, userRepo, passwordResetTokenRepo, inserter)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := CreatePasswordResetTokenInput{
		Email:  "reset-test@example.com",
		Locale: "ja",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.TokenID == "" {
		t.Error("TokenID が空です")
	}

	// エンキューが呼ばれたことを確認
	if !inserter.called {
		t.Error("Insert が呼ばれていません")
	}

	// SendPasswordResetArgs の検証
	resetArgs, ok := inserter.args.(worker.SendPasswordResetArgs)
	if !ok {
		t.Fatalf("args の型が SendPasswordResetArgs ではありません: %T", inserter.args)
	}
	if resetArgs.Email != "reset-test@example.com" {
		t.Errorf("Email = %s, want reset-test@example.com", resetArgs.Email)
	}
	if resetArgs.Locale != "ja" {
		t.Errorf("Locale = %s, want ja", resetArgs.Locale)
	}
	if resetArgs.AppURL == "" {
		t.Error("AppURL が空です")
	}
	if resetArgs.ResetURL == "" {
		t.Error("ResetURL が空です")
	}
}

func TestCreatePasswordResetTokenUsecase_Execute_DeletesExistingUnusedTokens(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	// テストユーザーを作成
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("delete-existing@example.com").
		WithAtname("delete_existing_user").
		Build()

	// 既存の未使用トークンを作成
	existingTokenDigest := "existing_token_digest_to_delete"
	testutil.NewPasswordResetTokenBuilderDB(t, db).
		WithUserID(userID).
		WithTokenDigest(existingTokenDigest).
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	userRepo := repository.NewUserRepository(q)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(q)
	inserter := &mockInserter{}
	uc := NewCreatePasswordResetTokenUsecase(cfg, db, userRepo, passwordResetTokenRepo, inserter)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := CreatePasswordResetTokenInput{
		Email:  "delete-existing@example.com",
		Locale: "ja",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.TokenID == "" {
		t.Error("TokenID が空です")
	}

	// 既存のトークンが削除されていることを確認
	existingToken, err := passwordResetTokenRepo.FindByTokenDigest(ctx, existingTokenDigest)
	if err != nil {
		t.Fatalf("FindByTokenDigest() error = %v", err)
	}
	if existingToken != nil {
		t.Error("既存の未使用トークンが削除されていません")
	}
}

func TestCreatePasswordResetTokenUsecase_Execute_EnglishLocale(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	// テストユーザーを作成
	_ = testutil.NewUserBuilderDB(t, db).
		WithEmail("english-reset@example.com").
		WithAtname("english_reset_user").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	userRepo := repository.NewUserRepository(q)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(q)
	inserter := &mockInserter{}
	uc := NewCreatePasswordResetTokenUsecase(cfg, db, userRepo, passwordResetTokenRepo, inserter)

	ctx := i18n.SetLocale(context.Background(), "en")
	input := CreatePasswordResetTokenInput{
		Email:  "english-reset@example.com",
		Locale: "en",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.TokenID == "" {
		t.Error("TokenID が空です")
	}

	// SendPasswordResetArgs の検証（英語）
	resetArgs, ok := inserter.args.(worker.SendPasswordResetArgs)
	if !ok {
		t.Fatalf("args の型が SendPasswordResetArgs ではありません: %T", inserter.args)
	}
	if resetArgs.Locale != "en" {
		t.Errorf("Locale = %s, want en", resetArgs.Locale)
	}
}

func TestCreatePasswordResetTokenUsecase_Execute_TokenIsHashedInDB(t *testing.T) {
	db := testutil.GetTestDB()
	q := query.New(db)

	// テストユーザーを作成
	_ = testutil.NewUserBuilderDB(t, db).
		WithEmail("hash-test@example.com").
		WithAtname("hash_test_user").
		Build()

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	userRepo := repository.NewUserRepository(q)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepository(q)
	inserter := &mockInserter{}
	uc := NewCreatePasswordResetTokenUsecase(cfg, db, userRepo, passwordResetTokenRepo, inserter)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := CreatePasswordResetTokenInput{
		Email:  "hash-test@example.com",
		Locale: "ja",
	}

	_, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// エンキューされたResetURLからトークンを取得
	resetArgs := inserter.args.(worker.SendPasswordResetArgs)

	// ResetURLからトークンを抽出（"https://wikino.app/password/edit?token=xxx" の形式）
	resetURL := resetArgs.ResetURL
	// URLからtokenパラメータを取得
	tokenStart := len("https://wikino.app/password/edit?token=")
	if len(resetURL) <= tokenStart {
		t.Fatalf("ResetURL が不正な形式です: %s", resetURL)
	}
	plainToken := resetURL[tokenStart:]

	// トークンをハッシュ化
	tokenDigest := password_reset.HashToken(plainToken)

	// ハッシュ化されたトークンでDBから検索
	token, err := passwordResetTokenRepo.FindByTokenDigest(ctx, tokenDigest)
	if err != nil {
		t.Fatalf("FindByTokenDigest() error = %v", err)
	}
	if token == nil {
		t.Fatal("トークンがDBに保存されていません")
	}

	// DBに保存されているのはハッシュ化されたトークンであることを確認
	if token.TokenDigest != tokenDigest {
		t.Errorf("TokenDigest = %s, want %s", token.TokenDigest, tokenDigest)
	}

	// 平文トークンがDBに保存されていないことを確認（ハッシュ化されているはず）
	if token.TokenDigest == plainToken {
		t.Error("平文トークンがDBに保存されています（ハッシュ化されていません）")
	}
}
