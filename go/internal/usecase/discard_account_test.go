package usecase

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// newDiscardAccountUsecaseはテスト用のデータベースと、渡したジョブ投入器を使うUseCaseを組み立てる。
func newDiscardAccountUsecase(db *sql.DB, inserter dispatcher.JobInserter) *DiscardAccountUsecase {
	queries := query.New(db)
	return NewDiscardAccountUsecase(
		db,
		repository.NewUserRepository(queries),
		repository.NewUserSessionRepository(queries),
		repository.NewSpaceRepository(queries),
		dispatcher.NewDispatcher(inserter, nil),
		validator.NewSettingsAccountDeletionCreateValidator(repository.NewSpaceRepository(queries)),
	)
}

// discardAccountTestPasswordは、アカウントの削除のテストのユーザーのパスワード
const discardAccountTestPassword = "discard-account-password"

// seedDiscardAccountUserは、パスワードとセッションを2つ持つユーザーを作成し、そのユーザーを返す。
func seedDiscardAccountUser(t *testing.T, db *sql.DB, suffix string) *model.User {
	t.Helper()

	atname := strings.ReplaceAll("discard_account_"+suffix, "-", "_")
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("discard-account-" + suffix + "@example.com").
		WithAtname(atname).
		Build()
	digest, err := auth.HashPassword(discardAccountTestPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	testutil.NewUserPasswordBuilderDB(t, db).WithUserID(userID).WithPasswordDigest(digest).Build()
	sessionRepo := repository.NewUserSessionRepository(query.New(db))
	for _, token := range []string{"discard-account-" + suffix + "-1", "discard-account-" + suffix + "-2"} {
		if _, err := sessionRepo.Create(context.Background(), repository.CreateInput{
			UserID:     userID,
			Token:      token,
			IPAddress:  "127.0.0.1",
			UserAgent:  "test",
			SignedInAt: time.Now(),
		}); err != nil {
			t.Fatalf("セッションの作成に失敗: %v", err)
		}
	}

	return &model.User{ID: userID, Atname: atname, Email: "discard-account-" + suffix + "@example.com"}
}

// userStateは、ユーザーが論理削除されているかと、残っているセッションの数を返す。
func userState(t *testing.T, db *sql.DB, userID model.UserID) (bool, int) {
	t.Helper()

	var discarded bool
	var sessions int
	if err := db.QueryRowContext(t.Context(), `
		SELECT u.discarded_at IS NOT NULL, (SELECT COUNT(*) FROM user_sessions s WHERE s.user_id = u.id)
		FROM users u WHERE u.id = $1`, string(userID)).Scan(&discarded, &sessions); err != nil {
		t.Fatalf("ユーザーの状態の取得に失敗: %v", err)
	}
	return discarded, sessions
}

func TestDiscardAccountUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	user := seedDiscardAccountUser(t, db, "success")
	other := seedDiscardAccountUser(t, db, "success-other")
	inserter := &mockJobInserter{}

	if err := newDiscardAccountUsecase(db, inserter).Execute(ctx, DiscardAccountInput{User: user, Atname: user.Atname}); err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	discarded, sessions := userState(t, db, user.ID)
	if !discarded {
		t.Error("ユーザーが論理削除されていない")
	}
	if sessions != 0 {
		t.Errorf("残っているセッション = %d件、期待値 = 0件", sessions)
	}
	args, ok := inserter.args.(dispatcher.DeleteAccountArgs)
	if !ok {
		t.Fatalf("投入したジョブの引数 = %T、DeleteAccountArgsを期待", inserter.args)
	}
	if args.UserID != string(user.ID) {
		t.Errorf("ジョブの引数のUserID = %q、期待値 = %q", args.UserID, user.ID)
	}

	// 論理削除したユーザーは、正しいパスワードでもログインできない
	q := query.New(db)
	signInValidator := validator.NewSignInCreateValidator(repository.NewUserRepository(q), repository.NewUserPasswordRepository(q), repository.NewUserTwoFactorAuthRepository(q))
	if _, err := signInValidator.Validate(ctx, validator.SignInCreateValidatorInput{
		Email:    user.Email,
		Password: discardAccountTestPassword,
	}); model.AsValidationError(err) == nil {
		t.Errorf("ログインの検証のエラー = %v、ValidationErrorを期待", err)
	}

	// ほかのユーザーは変わらない
	otherDiscarded, otherSessions := userState(t, db, other.ID)
	if otherDiscarded || otherSessions != 2 {
		t.Errorf("ほかのユーザーの状態 = 論理削除%v・セッション%d件、期待値 = 論理削除false・セッション2件", otherDiscarded, otherSessions)
	}
}

func TestDiscardAccountUsecase_Execute_アットネームが一致しなければ削除しない(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	user := seedDiscardAccountUser(t, db, "mismatch")
	inserter := &mockJobInserter{}

	err := newDiscardAccountUsecase(db, inserter).Execute(ctx, DiscardAccountInput{User: user, Atname: "someone_else"})
	if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("atname") {
		t.Fatalf("Execute()のエラー = %v、atnameのValidationErrorを期待", err)
	}

	discarded, sessions := userState(t, db, user.ID)
	if discarded || sessions != 2 {
		t.Errorf("ユーザーの状態 = 論理削除%v・セッション%d件、期待値 = 論理削除false・セッション2件", discarded, sessions)
	}
	if inserter.called {
		t.Error("物理削除のジョブが投入された")
	}
}

func TestDiscardAccountUsecase_Execute_参加しているスペースがあれば削除しない(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	user := seedDiscardAccountUser(t, db, "member")
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("discard-account-member").Build()
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(user.ID).Build()
	inserter := &mockJobInserter{}

	err := newDiscardAccountUsecase(db, inserter).Execute(ctx, DiscardAccountInput{User: user, Atname: user.Atname})
	if ve := model.AsValidationError(err); ve == nil || len(ve.Global) == 0 {
		t.Fatalf("Execute()のエラー = %v、フォーム全体のValidationErrorを期待", err)
	}

	discarded, sessions := userState(t, db, user.ID)
	if discarded || sessions != 2 {
		t.Errorf("ユーザーの状態 = 論理削除%v・セッション%d件、期待値 = 論理削除false・セッション2件", discarded, sessions)
	}
	if inserter.called {
		t.Error("物理削除のジョブが投入された")
	}
}

func TestDiscardAccountUsecase_Execute_ジョブを投入できなければ論理削除を取り消す(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	user := seedDiscardAccountUser(t, db, "enqueue-failure")

	if err := newDiscardAccountUsecase(db, rejectingJobInserter{}).Execute(ctx, DiscardAccountInput{User: user, Atname: user.Atname}); err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}

	discarded, sessions := userState(t, db, user.ID)
	if discarded {
		t.Error("ジョブを投入できなかったのに、ユーザーが論理削除されたまま")
	}
	// 消したセッションは戻らないため、ユーザーはログインし直してから削除をやり直す
	if sessions != 0 {
		t.Errorf("残っているセッション = %d件、期待値 = 0件", sessions)
	}
}

func TestGetAccountDeletionNewUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("get-account-deletion-new@example.com").
		WithAtname("get_account_deletion_new").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("get-account-deletion-new").Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	discardedSpaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("get-account-deletion-gone").WithDiscarded().Build()
	testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(discardedSpaceID).WithUserID(userID).Build()

	output, err := NewGetAccountDeletionNewUsecase(repository.NewSpaceRepository(queries)).
		Execute(t.Context(), GetAccountDeletionNewInput{UserID: userID})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if len(output.ActiveSpaces) != 1 || output.ActiveSpaces[0].ID != spaceID {
		t.Errorf("ActiveSpaces = %v、期待値 = 削除していないスペース%sだけ", output.ActiveSpaces, spaceID)
	}
}
