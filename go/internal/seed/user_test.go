package seed

import (
	"context"
	"database/sql"
	"io"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// testRosterPassword is the plaintext testRoster hashes. The roster itself
// carries only the digest, so a test that needs the password to verify against
// holds it here.
//
// [Ja] testRosterPassword は testRoster がハッシュ化する平文。名簿が持つのは
// ダイジェストだけであるため、検証にパスワードを必要とするテストはここで保持する。
const testRosterPassword = "seed-password"

// testRoster is the roster the generator tests work from. It names the same
// roles the seed does, so that what a generator reaches for by role is present
// here too, with values of its own so that a test does not depend on what the
// development roster happens to hold.
//
// [Ja] testRoster は生成器のテストが使う名簿。シードと同じ役割を挙げているため、
// 生成器が役割で求めるものはここにも揃っている。値はテスト独自のものにしており、
// 開発用の名簿がたまたま持っている内容にテストが依存しないようにしている。
func testRoster(t *testing.T) *userRoster {
	t.Helper()

	passwordDigest, err := auth.HashPassword(testRosterPassword)
	if err != nil {
		t.Fatalf("テスト用パスワードのハッシュ化に失敗: %v", err)
	}

	return &userRoster{
		path:           "seed-users.toml",
		passwordDigest: passwordDigest,
		users: []rosterUser{
			{
				role:         roleOwner,
				atname:       "seedowner",
				name:         "シードオーナー",
				email:        "seed-owner@example.com",
				featureFlags: model.AllFeatureFlagNames,
			},
			{
				role:      roleCollaborator,
				atname:    "seedcollaborator",
				name:      "シードコラボレーター",
				email:     "seed-collaborator@example.com",
				twoFactor: true,
			},
			{
				role:         roleGuest,
				atname:       "seedguest",
				name:         "シードゲスト",
				email:        "seed-guest@example.com",
				featureFlags: model.AllFeatureFlagNames,
			},
		},
	}
}

func TestGenerateUsers(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	roster := testRoster(t)

	users, err := generateUsers(ctx, tx, io.Discard, roster)
	if err != nil {
		t.Fatalf("ユーザー生成に失敗: %v", err)
	}

	for _, want := range roster.users {
		user := users.user(want.role)
		if user == nil {
			t.Fatalf("役割 %s のアカウントが作成されていない", want.role)
		}

		if user.Email != want.email {
			t.Errorf("役割 %s のメールアドレスが %q であることを期待したが %q だった", want.role, want.email, user.Email)
		}
		if user.Atname != want.atname {
			t.Errorf("役割 %s のアットネームが %q であることを期待したが %q だった", want.role, want.atname, user.Atname)
		}
		if user.Name != want.name {
			t.Errorf("役割 %s の名前が %q であることを期待したが %q だった", want.role, want.name, user.Name)
		}
		wantDescription := "ブラウザ確認用のシードユーザーです。"
		if user.Description != wantDescription {
			t.Errorf("役割 %s の説明が %q であることを期待したが %q だった", want.role, wantDescription, user.Description)
		}

		// The generated digest must verify against the password from the roster.
		// A sign-in client using those same credentials can therefore authenticate
		// immediately after the seed finishes.
		//
		// [Ja] 生成したダイジェストは名簿のパスワードそのもので検証できる必要がある。
		// これにより、同じ資格情報を使うサインインクライアントはシード完了直後に認証
		// できる。
		var digest string
		if err := tx.QueryRowContext(ctx, `SELECT password_digest FROM user_passwords WHERE user_id = $1`, string(user.ID)).Scan(&digest); err != nil {
			t.Fatalf("役割 %s のパスワード取得に失敗: %v", want.role, err)
		}
		if !auth.VerifyPassword(digest, testRosterPassword) {
			t.Errorf("役割 %s のパスワードダイジェストが名簿のパスワードと一致しない", want.role)
		}

		assertFeatureFlags(ctx, t, tx, user.ID, want.featureFlags)

		if want.twoFactor {
			assertTwoFactorAuthEnabled(ctx, t, tx, user.ID)
		} else {
			assertTwoFactorAuthAbsent(ctx, t, tx, user.ID)
		}
	}
}

// assertFeatureFlags checks that the user holds exactly the given flags.
//
// [Ja] assertFeatureFlags は、ユーザーが与えられたフラグだけを持つことを確認する。
func assertFeatureFlags(ctx context.Context, t *testing.T, tx *sql.Tx, userID model.UserID, want []model.FeatureFlagName) {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `SELECT name FROM feature_flags WHERE user_id = $1`, string(userID))
	if err != nil {
		t.Fatalf("フィーチャーフラグの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	got := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("フィーチャーフラグ名の読み取りに失敗: %v", err)
		}
		got[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("フィーチャーフラグの走査に失敗: %v", err)
	}

	if len(got) != len(want) {
		t.Errorf("フィーチャーフラグが %d 件であることを期待したが %d 件だった", len(want), len(got))
	}
	for _, name := range want {
		if !got[string(name)] {
			t.Errorf("フィーチャーフラグ %s が付与されていない", name)
		}
	}
}

// assertTwoFactorAuthEnabled checks that the user can clear the two-factor
// step, both with a generated code and with a recovery code.
//
// [Ja] assertTwoFactorAuthEnabled は、生成コードとリカバリーコードのどちらでも
// ユーザーが 2 要素認証ステップを通過できることを確認する。
func assertTwoFactorAuthEnabled(ctx context.Context, t *testing.T, tx *sql.Tx, userID model.UserID) {
	t.Helper()

	var (
		secret        string
		enabled       bool
		recoveryCodes []string
	)
	err := tx.QueryRowContext(
		ctx,
		`SELECT secret, enabled, recovery_codes FROM user_two_factor_auths WHERE user_id = $1`,
		string(userID),
	).Scan(&secret, &enabled, pq.Array(&recoveryCodes))
	if err != nil {
		t.Fatalf("二要素認証設定の取得に失敗: %v", err)
	}

	if !enabled {
		t.Error("二要素認証が有効であることを期待したが無効だった")
	}
	if len(recoveryCodes) != recoveryCodeCount {
		t.Errorf("リカバリーコードが %d 件であることを期待したが %d 件だった", recoveryCodeCount, len(recoveryCodes))
	}
	if len(recoveryCodes) > 0 {
		twoFactorAuthRepo := repository.NewUserTwoFactorAuthRepository(query.New(tx))
		recoveryValidator := validator.NewSignInTwoFactorRecoveryCreateValidator(twoFactorAuthRepo)
		if _, err := recoveryValidator.Validate(ctx, validator.SignInTwoFactorRecoveryCreateValidatorInput{
			UserID:       userID,
			RecoveryCode: recoveryCodes[0],
		}); err != nil {
			t.Errorf("シードしたリカバリーコードが認証経路で受理されなかった: %v", err)
		}
	}

	// A secret that cmd/devtotp cannot turn into a code would leave the
	// account unable to sign in at all.
	//
	// [Ja] cmd/devtotp がコードに変換できない secret だと、そのアカウントは
	// そもそもサインインできなくなる。
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("TOTPコードの生成に失敗: %v", err)
	}
	if !totp.Validate(code, secret) {
		t.Error("生成したTOTPコードが検証を通らなかった")
	}
}

// assertTwoFactorAuthAbsent checks that the user has no two-factor auth
// settings at all. An account whose sign-in is meant to complete with a
// password alone stops showing that path the moment it grows a row here.
//
// [Ja] assertTwoFactorAuthAbsent は、ユーザーが 2 要素認証の設定をまったく
// 持たないことを確認する。パスワードだけでサインインが完了するはずのアカウントは、
// ここに行ができた時点でその経路を見せなくなる。
func assertTwoFactorAuthAbsent(ctx context.Context, t *testing.T, tx *sql.Tx, userID model.UserID) {
	t.Helper()

	var count int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT count(*) FROM user_two_factor_auths WHERE user_id = $1`,
		string(userID),
	).Scan(&count); err != nil {
		t.Fatalf("二要素認証設定の件数取得に失敗: %v", err)
	}
	if count != 0 {
		t.Errorf("二要素認証が無いことを期待したが %d 件の設定があった", count)
	}
}
