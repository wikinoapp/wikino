package seed

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// rosterPath is the file a run reads its accounts from, and rosterExamplePath
// is the example committed in its place. Both are relative to the directory a
// run is started in, which is the Go module root (see the seed target in
// go/Makefile).
//
// The roster holds personal email addresses, so it is kept out of version
// control and the example is committed instead. The example says who exists in
// a development environment and what each account is there to show without
// naming anybody.
//
// [Ja] rosterPath は実行がアカウントを読み込むファイル、rosterExamplePath はその
// 代わりにコミットしている見本。どちらも実行を開始したディレクトリからの相対パス
// であり、それは Go モジュールのルートになる (go/Makefile の seed ターゲットを参照)。
//
// 名簿は個人のメールアドレスを持つためバージョン管理には入れず、代わりに見本を
// コミットしている。見本は誰かを名指しすることなく、開発環境にどんなアカウントが
// いて、それぞれが何を確認するためにいるのかを説明する。
const (
	rosterPath        = "seed-users.toml"
	rosterExamplePath = "seed-users.example.toml"
)

// allFeatureFlagsKeyword is what feature_flags holds for an account that is to
// be given every flag the application defines. Naming the flags one by one
// would leave that account behind as soon as a flag is added, which is exactly
// when an account that opens every screen is wanted.
//
// [Ja] allFeatureFlagsKeyword は、アプリケーションが定義するフラグを全件与える
// アカウントの feature_flags に書く値。フラグを 1 つずつ名指しすると、フラグが
// 追加された時点でそのアカウントが取り残される。すべての画面が開くアカウントが
// 必要になるのは、まさにそのときであるにもかかわらず。
const allFeatureFlagsKeyword = "all"

// rosterFile is the roster as it is written in the file.
//
// [Ja] rosterFile は、ファイルに書かれたままの名簿。
type rosterFile struct {
	// Password is shared by every account. The seed refuses to run outside a
	// development environment and the dev site itself sits behind basic auth,
	// so a password per account would buy nothing and cost one password
	// manager entry per account.
	//
	// [Ja] Password は全アカウントで共通。シードは開発環境以外での実行を拒否し、
	// dev サイト自体も Basic 認証の内側にあるため、アカウントごとに別のパスワードを
	// 持たせても得るものが無く、アカウントの数だけパスワード管理の項目が増える。
	Password string           `toml:"password"`
	Users    []rosterUserFile `toml:"users"`
}

// rosterUserFile is one [[users]] entry as it is written in the file.
//
// [Ja] rosterUserFile は、ファイルに書かれたままの [[users]] 1 件。
type rosterUserFile struct {
	Role         string               `toml:"role"`
	Atname       string               `toml:"atname"`
	Name         string               `toml:"name"`
	Email        string               `toml:"email"`
	FeatureFlags featureFlagSelection `toml:"feature_flags"`
	TwoFactor    *bool                `toml:"two_factor"`
}

// featureFlagSelection is the feature_flags value of one entry. It takes
// either the string "all" or a list of flag names, so that an account meant to
// open every screen behind a flag can say so, and an account meant to open
// some of them can name those.
//
// [Ja] featureFlagSelection は 1 件分の feature_flags の値。文字列 "all" か、
// フラグ名の配列のどちらかを取る。フラグの内側にあるすべての画面を開くための
// アカウントはそう書け、一部だけを開くためのアカウントはその名前を挙げられる
// ようにするため。
type featureFlagSelection struct {
	present bool
	all     bool
	names   []string
}

// UnmarshalTOML reads the two shapes feature_flags is allowed to take.
//
// [Ja] UnmarshalTOML は、feature_flags が取りうる 2 つの形を読む。
func (s *featureFlagSelection) UnmarshalTOML(data any) error {
	s.present = true

	switch value := data.(type) {
	case string:
		if value != allFeatureFlagsKeyword {
			return fmt.Errorf(
				"feature_flags に書ける文字列は %q だけです。一部のフラグを与えるときは [\"go_example\"] のように配列で書いてください",
				allFeatureFlagsKeyword,
			)
		}
		s.all = true

		return nil
	case []any:
		names := make([]string, 0, len(value))
		for _, item := range value {
			name, ok := item.(string)
			if !ok {
				return fmt.Errorf("feature_flags の要素はフィーチャーフラグ名の文字列である必要がありますが %v がありました", item)
			}
			names = append(names, name)
		}
		s.names = names

		return nil
	default:
		return fmt.Errorf("feature_flags は %q かフィーチャーフラグ名の配列である必要があります", allFeatureFlagsKeyword)
	}
}

// resolve turns the selection into the flags to grant, and reports a name that
// the application does not define. A flag the roster misspells would otherwise
// leave the account without the screen it was written to open, and nothing
// would say why.
//
// [Ja] resolve は選択を、付与するフラグへ変換する。アプリケーションが定義して
// いない名前は報告する。名簿が綴りを誤ったフラグは、そうしないと、そのアカウント
// から本来開くはずの画面を落としたまま、理由を何も告げないため。
func (s featureFlagSelection) resolve() ([]model.FeatureFlagName, error) {
	if s.all {
		return slices.Clone(model.AllFeatureFlagNames), nil
	}

	flags := make([]model.FeatureFlagName, 0, len(s.names))
	seen := make(map[model.FeatureFlagName]bool, len(s.names))
	for _, name := range s.names {
		flag := model.FeatureFlagName(name)
		if !slices.Contains(model.AllFeatureFlagNames, flag) {
			return nil, fmt.Errorf(
				"feature_flags の %q は定義されていないフィーチャーフラグです。指定できるのは %s です",
				name, joinFeatureFlagNames(model.AllFeatureFlagNames),
			)
		}
		if seen[flag] {
			return nil, fmt.Errorf("feature_flags の %q が 2 回以上指定されています", name)
		}
		seen[flag] = true
		flags = append(flags, flag)
	}

	return flags, nil
}

// rosterUser is one account the roster names.
//
// [Ja] rosterUser は、名簿が挙げるアカウント 1 件。
type rosterUser struct {
	role         seedRole
	atname       string
	name         string
	email        string
	featureFlags []model.FeatureFlagName
	twoFactor    bool
}

// userRoster is the accounts a run creates. Who exists is configuration rather
// than code: the addresses are personal, and an account is added to look at a
// screen from a viewpoint the existing ones cannot take. Which account does
// what stays in the code, which reaches for them by role.
//
// [Ja] userRoster は実行が作成するアカウント。誰がいるのかはコードではなく設定と
// する。アドレスが個人のものであることと、アカウントが足されるのは、既存の
// アカウントでは取れない視点から画面を見るためであることによる。どのアカウントが
// 何をするのかはコードに残り、コードはアカウントを役割で引く。
type userRoster struct {
	// path is the file the roster was read from. A run reports it beside the
	// database it is about to empty, so that both of the things it was pointed
	// at can be read off one line.
	//
	// [Ja] path は名簿を読み込んだファイル。実行はこれを、これから空にする
	// データベースと並べて報告する。実行が何を向いているのかを 1 行で読み取れる
	// ようにするため。
	path string
	// passwordDigest is the shared password as the accounts store it. The
	// plaintext is not carried past reading: what a run writes is the digest,
	// and hashing once while the roster is read is what lets it be dropped.
	//
	// [Ja] passwordDigest は、アカウントが保存する形にした共通パスワード。平文は
	// 読み込みの先へは持ち越さない。実行が書き込むのはダイジェストであり、名簿の
	// 読み込み時に一度ハッシュ化していることが、平文を落とせる理由になる。
	passwordDigest string
	users          []rosterUser
}

// loadUserRoster reads the roster from path.
//
// A missing file is an error rather than a fall back to the example. The
// example carries placeholder addresses, and signing in as an account that
// nobody reads mail for is not something to arrive at by accident.
//
// [Ja] loadUserRoster は path から名簿を読む。
//
// ファイルが無い場合は、見本へフォールバックせずエラーとする。見本が持つのは
// 仮のアドレスであり、誰もメールを読まないアカウントでサインインする状態へ、
// 気付かないまま辿り着いてよいものではないため。
func loadUserRoster(path string) (*userRoster, error) {
	file, err := loadRosterFile(path)
	if err != nil {
		return nil, err
	}

	roster, err := file.toUserRoster(path)
	if err != nil {
		return nil, fmt.Errorf("開発用ユーザーの名簿 %s: %w", path, err)
	}

	return roster, nil
}

// loadRosterFile reads the roster at path as it is written, without checking
// what it holds.
//
// Reading the file is the step shared by the two things the roster is read
// for: the accounts a run creates, and the credentials the browser
// verification signs in with. Keeping it in one function is what makes both of
// them fail the same way, and say the same thing, when the file is missing or
// misspelled.
//
// [Ja] loadRosterFile は path の名簿を、書かれたままの形で読む。中身が何であるかの
// 検査は行わない。
//
// ファイルを読むことは、名簿が読まれる 2 つの目的 (実行が作成するアカウントと、
// ブラウザ確認がサインインに使う資格情報) が共有する手順である。1 つの関数にまとめて
// いることが、ファイルが無いときや書き間違いがあるときに、両者が同じ形で失敗し、
// 同じことを告げる理由になる。
func loadRosterFile(path string) (rosterFile, error) {
	var file rosterFile

	meta, err := toml.DecodeFile(path, &file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return rosterFile{}, fmt.Errorf("開発用ユーザーの名簿 %s がありません。%s をコピーして作成してください", path, rosterExamplePath)
		}

		return rosterFile{}, fmt.Errorf("開発用ユーザーの名簿 %s の読み込みに失敗: %w", path, err)
	}

	// A key the decoder did not use is a typo: the value written next to it
	// never reaches the account, and the run goes on to create that account
	// without whatever the key was meant to give it.
	//
	// [Ja] デコーダが使わなかったキーは書き間違いである。その隣に書かれた値は
	// アカウントへ届かず、実行はそのキーが与えるはずだったものを欠いたまま、
	// そのアカウントを作りに行く。
	if keys := meta.Undecoded(); len(keys) > 0 {
		return rosterFile{}, fmt.Errorf("開発用ユーザーの名簿 %s に知らないキーがあります: %s", path, joinTOMLKeys(keys))
	}

	return file, nil
}

// toUserRoster checks the roster over and returns what the generators work
// from.
//
// [Ja] toUserRoster は名簿を検査し、生成器が使う形にして返す。
func (f rosterFile) toUserRoster(path string) (*userRoster, error) {
	users, err := f.validate()
	if err != nil {
		return nil, err
	}

	// Hash the shared password while the roster is still being read. Besides
	// rejecting an input bcrypt cannot handle before the database is touched,
	// this lets every account use the same prepared digest instead of discovering
	// a hashing failure after its user row has already been inserted.
	//
	// [Ja] 名簿を読み込んでいる間に共通パスワードをハッシュ化する。bcrypt が処理
	// できない入力をデータベースへ触る前に拒否できるだけでなく、各アカウントが準備
	// 済みの同じダイジェストを使えるため、ユーザー行を INSERT した後でハッシュ化の
	// 失敗が判明することも防げる。
	passwordDigest, err := auth.HashPassword(f.Password)
	if err != nil {
		return nil, fmt.Errorf("password のハッシュ化に失敗: %w", err)
	}

	return &userRoster{
		path:           path,
		passwordDigest: passwordDigest,
		users:          users,
	}, nil
}

// validate checks the roster over and returns the accounts it holds.
//
// It stops short of the password digest, which is what separates it from
// toUserRoster. Reading the roster to hand one account's credentials to the
// browser verification wants the password as it is written, and hashing it
// there would buy nothing for the wait it costs.
//
// [Ja] validate は名簿を検査し、そこに書かれているアカウントを返す。
//
// パスワードダイジェストの手前で止まる点が toUserRoster との違いになる。1 件分の
// 資格情報をブラウザ確認へ渡すために名簿を読むときに要るのは書かれたままのパスワード
// であり、そこでハッシュ化しても、かかる待ち時間に見合うものが無いため。
func (f rosterFile) validate() ([]rosterUser, error) {
	if f.Password == "" {
		return nil, errors.New("password が空です。全アカウント共通のサインインパスワードを書いてください")
	}
	if strings.ContainsAny(f.Password, "\r\n") {
		return nil, errors.New("password に CR / LF は含められません。改行を含まないパスワードを書いてください")
	}
	if len(f.Users) == 0 {
		return nil, errors.New("[[users]] が 1 件もありません")
	}

	users := make([]rosterUser, 0, len(f.Users))
	roles := make(map[seedRole]bool, len(f.Users))
	atnames := make(map[string]bool, len(f.Users))
	emails := make(map[string]bool, len(f.Users))

	for i, entry := range f.Users {
		user, err := entry.toRosterUser()
		if err != nil {
			return nil, fmt.Errorf("%d 件目の [[users]]: %w", i+1, err)
		}

		// The role is what a generator names an account by, the atname goes
		// into a URL and the email is what the browser verification signs in
		// with. A value written twice makes one of the two accounts
		// unreachable through whichever of the three it shares.
		//
		// Two atnames that differ only in case are the same atname: the column
		// is citext, so the unique index the second account would be written
		// against does not tell them apart. Comparing them the same way here
		// keeps that collision from surfacing as a failed insert after the
		// database has been emptied. Emails are compared as written, which is
		// how both the column and the sign-in lookup compare them.
		//
		// [Ja] 役割は生成器がアカウントを名指しする名前、atname は URL に入る名前、
		// メールアドレスはブラウザ確認がサインインに使う名前である。同じ値を 2 度
		// 書くと、共有した名前ではどちらか一方のアカウントへ辿り着けなくなる。
		//
		// 大文字小文字だけが違う 2 つの atname は同じ atname である。カラムが citext
		// であり、2 件目のアカウントが書き込まれる一意インデックスがそれらを区別
		// しないため。ここでも同じ方法で比較することで、この衝突がデータベースを
		// 空にした後の INSERT 失敗として表面化することを防ぐ。メールアドレスは
		// 書かれたとおりに比較する。カラムもサインイン時の引き当ても、そう比較する
		// ため。
		if roles[user.role] {
			return nil, fmt.Errorf("役割 %s の [[users]] が 2 件以上あります", user.role)
		}
		atnameKey := strings.ToLower(user.atname)
		if atnames[atnameKey] {
			return nil, fmt.Errorf("atname %q の [[users]] が 2 件以上あります (大文字小文字の違いは同じ atname として扱われます)", user.atname)
		}
		if emails[user.email] {
			return nil, fmt.Errorf("email %q の [[users]] が 2 件以上あります", user.email)
		}
		roles[user.role] = true
		atnames[atnameKey] = true
		emails[user.email] = true

		users = append(users, user)
	}

	// A role the generators name but the roster does not hold would not
	// surface until the generator that needs it runs, which is after the run
	// has emptied the database.
	//
	// [Ja] 生成器が名指ししているのに名簿に無い役割は、それを必要とする生成器が
	// 走るまで表面化せず、それは実行がデータベースを空にした後になる。
	for _, role := range allSeedRoles {
		if !roles[role] {
			return nil, fmt.Errorf("役割 %s の [[users]] がありません。生成器がこの役割を名指しするため、1 件必要です", role)
		}
	}

	return users, nil
}

// toRosterUser checks one entry over and returns it in the form the generators
// work from.
//
// [Ja] toRosterUser は 1 件分を検査し、生成器が使う形にして返す。
func (e rosterUserFile) toRosterUser() (rosterUser, error) {
	for _, field := range []struct {
		key   string
		value string
	}{
		{key: "role", value: e.Role},
		{key: "atname", value: e.Atname},
		{key: "name", value: e.Name},
		{key: "email", value: e.Email},
	} {
		if strings.TrimSpace(field.value) == "" {
			return rosterUser{}, fmt.Errorf("%s が空です", field.key)
		}
	}

	role := seedRole(e.Role)
	if !slices.Contains(allSeedRoles, role) {
		return rosterUser{}, fmt.Errorf("role %q は生成器が知らない役割です。指定できるのは %s です", e.Role, joinSeedRoles(allSeedRoles))
	}
	// The atname is checked against the rule the application applies to every
	// account, rather than a copy of it here, so that an atname the roster
	// accepts is one an account can hold. Checking it while the roster is read
	// keeps an invalid account from being found only after the database has
	// been emptied.
	//
	// [Ja] atname は、ここに置いた写しではなくアプリケーションがすべてのアカウント
	// に課している規則で検査する。名簿が受理する atname が、アカウントが実際に
	// 持てる atname であるようにするため。名簿の読み込み時に検査することで、
	// データベースを空にした後で初めて不正なアカウントが見つかることを防ぐ。
	if !validator.IsValidAtname(e.Atname) {
		return rosterUser{}, fmt.Errorf(
			"atname %q に使える文字は半角英数字とアンダースコアだけで、%d 文字以内である必要があります",
			e.Atname, validator.AtnameMaxLength,
		)
	}
	// An email accepted by the roster must also be the address that signs the
	// account in. The roster stores what is written rather than what parsing
	// makes of it, while surrounding whitespace and a display name parse away:
	// an account written either way would hold an address that the sign-in
	// form, whose field drops the same whitespace, cannot submit. Requiring the
	// written form to be the address itself keeps the seed from finishing with
	// an account nobody can reach.
	//
	// [Ja] 名簿が受理するメールアドレスは、そのアカウントをサインインさせる
	// アドレスそのものである必要がある。名簿が保存するのは解釈した結果ではなく
	// 書かれた文字列である一方、前後の空白や表示名は解釈の過程で落ちる。どちらの
	// 書き方をしたアカウントも、同じ空白を落とすサインインフォームからは送信できない
	// アドレスを持つことになる。書かれた文字列がアドレスそのものであることを求める
	// ことで、誰も辿り着けないアカウントを作ったままシードが正常終了することを防ぐ。
	address, canonical := validator.CanonicalEmail(e.Email)
	if !canonical {
		if address == "" {
			return rosterUser{}, errors.New("email がメールアドレスの形式ではありません")
		}

		return rosterUser{}, fmt.Errorf(
			"email にはアドレスだけを書いてください (前後の空白や表示名は含められません)。%q のつもりであれば、そう書き直してください",
			address,
		)
	}
	if !e.FeatureFlags.present {
		return rosterUser{}, errors.New("feature_flags がありません")
	}
	if e.TwoFactor == nil {
		return rosterUser{}, errors.New("two_factor がありません")
	}

	flags, err := e.FeatureFlags.resolve()
	if err != nil {
		return rosterUser{}, err
	}

	// The name is the one required string with no format of its own to check
	// it against: the atname's rule rejects whitespace and the email has to be
	// written as the address itself, while a name is whatever it is. Trimming
	// it is what keeps a stray space in the file from becoming a display name
	// that reads as indented on every screen the account appears on.
	//
	// [Ja] name は、照らし合わせる形式を自身では持たない唯一の必須文字列である。
	// atname は規則が空白を弾き、email は書かれた文字列がアドレスそのものである
	// ことを求められるが、名前は書かれたものが何であれ名前になる。トリムすること
	// で、ファイルに紛れ込んだ空白が、そのアカウントが現れるすべての画面で字下げ
	// されて見える表示名になることを防ぐ。
	return rosterUser{
		role:         role,
		atname:       e.Atname,
		name:         strings.TrimSpace(e.Name),
		email:        e.Email,
		featureFlags: flags,
		twoFactor:    *e.TwoFactor,
	}, nil
}

// joinTOMLKeys lists keys for an error message.
//
// [Ja] joinTOMLKeys は、エラーメッセージ用にキーを並べる。
func joinTOMLKeys(keys []toml.Key) string {
	ss := make([]string, 0, len(keys))
	for _, key := range keys {
		ss = append(ss, key.String())
	}

	return strings.Join(ss, ", ")
}

// joinSeedRoles lists roles for an error message.
//
// [Ja] joinSeedRoles は、エラーメッセージ用に役割を並べる。
func joinSeedRoles(roles []seedRole) string {
	ss := make([]string, 0, len(roles))
	for _, role := range roles {
		ss = append(ss, string(role))
	}

	return strings.Join(ss, ", ")
}

// joinFeatureFlagNames lists feature flag names for an error message.
//
// [Ja] joinFeatureFlagNames は、エラーメッセージ用にフィーチャーフラグ名を並べる。
func joinFeatureFlagNames(names []model.FeatureFlagName) string {
	ss := make([]string, 0, len(names))
	for _, name := range names {
		ss = append(ss, string(name))
	}

	return strings.Join(ss, ", ")
}
