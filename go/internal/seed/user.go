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

// seedRoleは、生成器がアカウントを求めるときに使う論理名。生成器が名指し
// するのは「1人目」「2人目」ではなく、そのスペースの管理者や、その横で書く
// 共同編集者になる。シードにアカウントを1つ足したとき、その役割を必要としない
// 生成器が変わらないようにするため。
type seedRole string

const (
	// roleOwnerは両スペースを管理し、フィーチャーフラグを全件持つ。データ
	// ベースを手で触らずに画面のGo版を開けるようにするため。サインインはパス
	// ワードだけで完了する。
	roleOwner seedRole = "owner"
	// roleCollaboratorはspace:adminを持たずにseed-wikiで書くアカウントで、
	// seed-soloには参加していない。フィーチャーフラグを1つも持たず2要素認証が
	// 有効になっており、2つの役割を合わせると、フラグ有無による画面の比較と、
	// サインインの2要素認証ステップの通過の双方を確認できる。
	roleCollaborator seedRole = "collaborator"
	// roleGuestは、自分が参加していないスペースを開くアカウント。roleOwnerと
	// 同じくフィーチャーフラグを全件持ち、roleCollaboratorと同じくspace:admin無しで
	// seed-wikiに参加しているが、seed-soloのメンバーではない。画面のGo版へ
	// GuestPolicyを通って辿り着けるのは、この組み合わせだけである。roleOwnerは
	// フラグを持つが両方のスペースに参加しており、roleCollaboratorはseed-soloの
	// 外にいるがフラグを1つも持たないためRailsが応答する。サインインはパスワード
	// だけで完了する。
	roleGuest seedRole = "guest"
)

// allSeedRolesは生成器が名指しする役割の一覧。名簿はこのそれぞれに1件ずつ
// アカウントを持つ必要があり、それによって生成器は役割を求めてアカウントを
// 受け取れる。
//
// contentAuthorRolesとは別の一覧になる。シードに加わることと、その内容を書くことは
// 別の判断であり、何かを書くためではなく、画面をどの視点から開くかのために足される
// アカウントもあるため。roleGuestがそれにあたる。
var allSeedRoles = []seedRole{roleOwner, roleCollaborator, roleGuest}

// contentAuthorRolesはseed-wikiへ書くアカウントを、生成器が仕事を回す順に
// 並べたもの。何のために交互にするのかは生成器ごとに違う (自分のページが並ぶホーム
// 画面、作成者ならクローズできる編集提案、2人が話しているものとして読めるスレッド)
// が、誰が加わるかは毎回同じ事実であり、ここへ役割を足すとすべての交互担当が
// 一度に変わる。
var contentAuthorRoles = []seedRole{roleOwner, roleCollaborator}

// twoFactorSecretは、サインインが2要素認証ステップを通るアカウントに
// 与えるTOTPのsecret。cmd/devtotpが生成するコードがシードをまたいで有効で
// あり続けるよう固定値にしている。シードは開発環境以外での実行を拒否するため、
// この値が届く先は開発用データベースだけであり、ソースに置いても問題ない。
const twoFactorSecret = "JBSWY3DPEHPK3PXP"

// recoveryCodeCountはRails版が2要素認証の有効化時に発行するコード数に
// 合わせている。リカバリーコード画面が現実的な件数で表示されるようにするため。
const recoveryCodeCount = 10

// seededUsersはシードが作成したアカウントを、後続の生成器のために保持する。
type seededUsers struct {
	byRole map[seedRole]*model.User
}

// userは、その役割で作成したアカウントを返す。シードがその役割のアカウントを
// 作っていない場合はnilを返す。
func (u *seededUsers) user(role seedRole) *model.User {
	return u.byRole[role]
}

// requireNameは、その役割のアカウントが画面上で名乗る表示名を返す。シードが
// その役割のアカウントを作っていない場合は、その役割を名指しするエラーを返す。
//
// 自身が書き込まれるスペースに参加していないアカウントを名指しするテキストは、
// 名前をこちらで求める。そのスペースのメンバーシップは名前を持っておらず、名前が
// 入るべき場所が空いたままの文は、ページへ書き込んでよいものではないため。
func (u *seededUsers) requireName(role seedRole) (string, error) {
	user := u.user(role)
	if user == nil {
		return "", fmt.Errorf("役割 %sのアカウントが作成されていない", role)
	}

	return user.Name, nil
}

// generateUsersは名簿が挙げるアカウントを作成する。それはブラウザ確認で
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

// createUserはユーザーと、サインインに使うパスワードダイジェストを作成する。
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
		return nil, fmt.Errorf("ユーザー %sの作成に失敗: %w", atname, err)
	}

	if _, err := repository.NewUserPasswordRepository(queries).Create(ctx, repository.CreateUserPasswordInput{
		UserID:         user.ID,
		PasswordDigest: passwordDigest,
	}); err != nil {
		return nil, fmt.Errorf("ユーザー %sのパスワードの作成に失敗: %w", atname, err)
	}

	return user, nil
}

// enableFeatureFlagsは、名簿がそのユーザーへ与えるフラグを付与する。
//
// Repositoryではなくここで行を書くのは、フラグの作成はRails側が担当しており、
// Go側にCreateが存在しないため。シードのために追加すると、シードだけが呼ぶ
// 本番コードをInfrastructure層に残すことになる。
func enableFeatureFlags(ctx context.Context, dbtx query.DBTX, userID model.UserID, names []model.FeatureFlagName) error {
	for _, name := range names {
		if _, err := dbtx.ExecContext(
			ctx,
			`INSERT INTO feature_flags (user_id, name) VALUES ($1, $2)`,
			string(userID), string(name),
		); err != nil {
			return fmt.Errorf("フィーチャーフラグ %sの付与に失敗: %w", name, err)
		}
	}

	return nil
}

// enableTwoFactorAuthはユーザーの2要素認証を有効にする。リカバリーコードも
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
