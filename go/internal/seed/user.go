package seed

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/lib/pq"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// seedRole is the logical name by which a generator asks for an account. A
// generator names the administrator of a space or the collaborator writing
// alongside them, never the first or the second account, so that an account
// added to the seed leaves the generators that do not need its role untouched.
//
// [Ja] seedRole は、生成器がアカウントを求めるときに使う論理名。生成器が名指し
// するのは「1 人目」「2 人目」ではなく、そのスペースの管理者や、その横で書く
// 共同編集者になる。シードにアカウントを 1 つ足したとき、その役割を必要としない
// 生成器が変わらないようにするため。
type seedRole string

const (
	// roleOwner administers both spaces and holds every feature flag, so that
	// the Go version of a screen can be opened without touching the database by
	// hand. It signs in with a password alone.
	//
	// [Ja] roleOwner は両スペースを管理し、フィーチャーフラグを全件持つ。データ
	// ベースを手で触らずに画面の Go 版を開けるようにするため。サインインはパス
	// ワードだけで完了する。
	roleOwner seedRole = "owner"
	// roleCollaborator writes in seed-wiki without holding space:admin, and has
	// not joined seed-solo at all. It holds no feature flag and has two-factor
	// auth enabled, which makes the two roles together enough to compare a
	// screen before and after a flag, and to walk the two-factor step of the
	// sign-in flow.
	//
	// [Ja] roleCollaborator は space:admin を持たずに seed-wiki で書くアカウントで、
	// seed-solo には参加していない。フィーチャーフラグを 1 つも持たず 2 要素認証が
	// 有効になっており、2 つの役割を合わせると、フラグ有無による画面の比較と、
	// サインインの 2 要素認証ステップの通過の双方を確認できる。
	roleCollaborator seedRole = "collaborator"
	// roleGuest opens a space it has not joined. It holds every feature flag,
	// as roleOwner does, and has joined seed-wiki without space:admin, as
	// roleCollaborator has, but it is not a member of seed-solo. That
	// combination is the only one that reaches the Go version of a screen
	// through GuestPolicy: roleOwner holds the flags but has joined both
	// spaces, and roleCollaborator is outside seed-solo but holds no flag, so
	// Rails answers for it. Its sign-in completes with a password alone.
	//
	// [Ja] roleGuest は、自分が参加していないスペースを開くアカウント。roleOwner と
	// 同じくフィーチャーフラグを全件持ち、roleCollaborator と同じく space:admin 無しで
	// seed-wiki に参加しているが、seed-solo のメンバーではない。画面の Go 版へ
	// GuestPolicy を通って辿り着けるのは、この組み合わせだけである。roleOwner は
	// フラグを持つが両方のスペースに参加しており、roleCollaborator は seed-solo の
	// 外にいるがフラグを 1 つも持たないため Rails が応答する。サインインはパスワード
	// だけで完了する。
	roleGuest seedRole = "guest"
)

// allSeedRoles lists every role a generator names. The roster has to hold one
// account for each of them, which is what lets a generator ask for a role and
// get an account back.
//
// It is not the same list as contentAuthorRoles: joining the seed and writing
// its content are separate decisions, and an account can be added for the
// viewpoint it opens a screen from rather than to write anything. roleGuest is
// that account.
//
// [Ja] allSeedRoles は生成器が名指しする役割の一覧。名簿はこのそれぞれに 1 件ずつ
// アカウントを持つ必要があり、それによって生成器は役割を求めてアカウントを
// 受け取れる。
//
// contentAuthorRoles とは別の一覧になる。シードに加わることと、その内容を書くことは
// 別の判断であり、何かを書くためではなく、画面をどの視点から開くかのために足される
// アカウントもあるため。roleGuest がそれにあたる。
var allSeedRoles = []seedRole{roleOwner, roleCollaborator, roleGuest}

// contentAuthorRoles are the accounts that write into seed-wiki, in the order
// the generators hand the work round them. What each generator alternates for
// differs — a home screen with pages of its own, a suggestion whose creator may
// close it, a thread that reads as two members talking — but who takes part is
// the same fact every time, and a role added here joins every rotation at once.
//
// [Ja] contentAuthorRoles は seed-wiki へ書くアカウントを、生成器が仕事を回す順に
// 並べたもの。何のために交互にするのかは生成器ごとに違う (自分のページが並ぶホーム
// 画面、作成者ならクローズできる編集提案、2 人が話しているものとして読めるスレッド)
// が、誰が加わるかは毎回同じ事実であり、ここへ役割を足すとすべての交互担当が
// 一度に変わる。
var contentAuthorRoles = []seedRole{roleOwner, roleCollaborator}

// twoFactorSecret is the TOTP secret given to the accounts whose sign-in goes
// through the two-factor step. It is a fixed value so that the code
// cmd/devtotp mints stays valid across seeds, and it is safe to keep in the
// source because it only ever reaches a development database: the seed refuses
// to run anywhere else.
//
// [Ja] twoFactorSecret は、サインインが 2 要素認証ステップを通るアカウントに
// 与える TOTP の secret。cmd/devtotp が生成するコードがシードをまたいで有効で
// あり続けるよう固定値にしている。シードは開発環境以外での実行を拒否するため、
// この値が届く先は開発用データベースだけであり、ソースに置いても問題ない。
const twoFactorSecret = "JBSWY3DPEHPK3PXP"

// recoveryCodeCount matches the number of codes the Rails version issues when
// two-factor auth is enabled, so the recovery code screen shows a realistic
// list.
//
// [Ja] recoveryCodeCount は Rails 版が 2 要素認証の有効化時に発行するコード数に
// 合わせている。リカバリーコード画面が現実的な件数で表示されるようにするため。
const recoveryCodeCount = 10

// seededUsers holds the accounts the seed created, for the generators that
// come after it.
//
// [Ja] seededUsers はシードが作成したアカウントを、後続の生成器のために保持する。
type seededUsers struct {
	byRole map[seedRole]*model.User
}

// user returns the account created for the role, or nil when the seed creates
// no account for it.
//
// [Ja] user は、その役割で作成したアカウントを返す。シードがその役割のアカウントを
// 作っていない場合は nil を返す。
func (u *seededUsers) user(role seedRole) *model.User {
	return u.byRole[role]
}

// generateUsers creates the accounts the roster names, which are the accounts
// the browser verification signs in as.
//
// [Ja] generateUsers は名簿が挙げるアカウントを作成する。それはブラウザ確認で
// サインインするアカウントである。
func generateUsers(ctx context.Context, dbtx query.DBTX, out io.Writer, roster *userRoster) (*seededUsers, error) {
	bar := newProgress(out, "ユーザー", len(roster.users))
	defer bar.finish()

	users := &seededUsers{byRole: make(map[seedRole]*model.User, len(roster.users))}

	for _, account := range roster.users {
		user, err := createUser(ctx, dbtx, account, roster.passwordDigest)
		if err != nil {
			return nil, err
		}

		if err := enableFeatureFlags(ctx, dbtx, user.ID, account.featureFlags); err != nil {
			return nil, err
		}
		if account.twoFactor {
			if err := enableTwoFactorAuth(ctx, dbtx, user.ID); err != nil {
				return nil, err
			}
		}

		users.byRole[account.role] = user
		bar.advance()
	}

	return users, nil
}

// createUser creates a user together with the password digest that lets it
// sign in.
//
// [Ja] createUser はユーザーと、サインインに使うパスワードダイジェストを作成する。
func createUser(ctx context.Context, dbtx query.DBTX, account rosterUser, passwordDigest string) (*model.User, error) {
	queries := query.New(dbtx)
	atname := account.atname

	user, err := repository.NewUserRepository(queries).Create(ctx, repository.CreateUserInput{
		Email:       account.email,
		Atname:      atname,
		Name:        account.name,
		Description: "ブラウザ確認用のシードユーザーです。",
		Locale:      model.LocaleJa,
		TimeZone:    "Asia/Tokyo",
		JoinedAt:    time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("ユーザー %s の作成に失敗: %w", atname, err)
	}

	if _, err := repository.NewUserPasswordRepository(queries).Create(ctx, repository.CreateUserPasswordInput{
		UserID:         user.ID,
		PasswordDigest: passwordDigest,
	}); err != nil {
		return nil, fmt.Errorf("ユーザー %s のパスワードの作成に失敗: %w", atname, err)
	}

	return user, nil
}

// enableFeatureFlags grants the user the flags the roster gives it.
//
// The rows are written here rather than through a repository because flags are
// created from the Rails side and no Create exists on the Go side. Adding one
// for the seed would leave production code in the Infrastructure layer that
// only the seed calls.
//
// [Ja] enableFeatureFlags は、名簿がそのユーザーへ与えるフラグを付与する。
//
// Repository ではなくここで行を書くのは、フラグの作成は Rails 側が担当しており、
// Go 側に Create が存在しないため。シードのために追加すると、シードだけが呼ぶ
// 本番コードを Infrastructure 層に残すことになる。
func enableFeatureFlags(ctx context.Context, dbtx query.DBTX, userID model.UserID, names []model.FeatureFlagName) error {
	for _, name := range names {
		if _, err := dbtx.ExecContext(
			ctx,
			`INSERT INTO feature_flags (user_id, name) VALUES ($1, $2)`,
			string(userID), string(name),
		); err != nil {
			return fmt.Errorf("フィーチャーフラグ %s の付与に失敗: %w", name, err)
		}
	}

	return nil
}

// enableTwoFactorAuth turns on two-factor auth for the user, with recovery
// codes so that the recovery path can be walked as well.
//
// [Ja] enableTwoFactorAuth はユーザーの 2 要素認証を有効にする。リカバリーコードも
// 併せて登録し、リカバリー経路も確認できるようにする。
func enableTwoFactorAuth(ctx context.Context, dbtx query.DBTX, userID model.UserID) error {
	codes := make([]string, 0, recoveryCodeCount)
	for i := 1; i <= recoveryCodeCount; i++ {
		codes = append(codes, fmt.Sprintf("seedcd%02d", i))
	}

	now := time.Now()
	if _, err := dbtx.ExecContext(
		ctx,
		`INSERT INTO user_two_factor_auths
           (user_id, secret, enabled, enabled_at, recovery_codes, created_at, updated_at)
         VALUES ($1, $2, true, $3, $4, $5, $6)`,
		string(userID), twoFactorSecret, now, pq.Array(codes), now, now,
	); err != nil {
		return fmt.Errorf("二要素認証設定の作成に失敗: %w", err)
	}

	return nil
}
