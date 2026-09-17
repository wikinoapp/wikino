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

// rosterPathは実行がアカウントを読み込むファイル、rosterExamplePathはその
// 代わりにコミットしている見本。どちらも実行を開始したディレクトリからの相対パス
// であり、それはGoモジュールのルートになる (go/Makefileのseedターゲットを参照)。
//
// 名簿は個人のメールアドレスを持つためバージョン管理には入れず、代わりに見本を
// コミットしている。見本は誰かを名指しすることなく、開発環境にどんなアカウントが
// いて、それぞれが何を確認するためにいるのかを説明する。
const (
	rosterPath        = "seed-users.toml"
	rosterExamplePath = "seed-users.example.toml"
)

// allFeatureFlagsKeywordは、アプリケーションが定義するフラグを全件与える
// アカウントのfeature_flagsに書く値。フラグを1つずつ名指しすると、フラグが
// 追加された時点でそのアカウントが取り残される。すべての画面が開くアカウントが
// 必要になるのは、まさにそのときであるにもかかわらず。
const allFeatureFlagsKeyword = "all"

// rosterFileは、ファイルに書かれたままの名簿。
type rosterFile struct {
	// Passwordは全アカウントで共通。シードは開発環境以外での実行を拒否し、
	// devサイト自体もBasic認証の内側にあるため、アカウントごとに別のパスワードを
	// 持たせても得るものが無く、アカウントの数だけパスワード管理の項目が増える。
	Password string           `toml:"password"`
	Users    []rosterUserFile `toml:"users"`
}

// rosterUserFileは、ファイルに書かれたままの [[users]] 1件。
type rosterUserFile struct {
	Role         string               `toml:"role"`
	Atname       string               `toml:"atname"`
	Name         string               `toml:"name"`
	Email        string               `toml:"email"`
	FeatureFlags featureFlagSelection `toml:"feature_flags"`
	TwoFactor    *bool                `toml:"two_factor"`
}

// featureFlagSelectionは1件分のfeature_flagsの値。文字列 "all" か、
// フラグ名の配列のどちらかを取る。フラグの内側にあるすべての画面を開くための
// アカウントはそう書け、一部だけを開くためのアカウントはその名前を挙げられる
// ようにするため。
type featureFlagSelection struct {
	present bool
	all     bool
	names   []string
}

// UnmarshalTOMLは、feature_flagsが取りうる2つの形を読む。
func (s *featureFlagSelection) UnmarshalTOML(data any) error {
	s.present = true

	switch value := data.(type) {
	case string:
		if value != allFeatureFlagsKeyword {
			return fmt.Errorf(
				"feature_flagsに書ける文字列は %qだけです。一部のフラグを与えるときは [\"go_example\"] のように配列で書いてください",
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
				return fmt.Errorf("feature_flagsの要素はフィーチャーフラグ名の文字列である必要がありますが %vがありました", item)
			}
			names = append(names, name)
		}
		s.names = names

		return nil
	default:
		return fmt.Errorf("feature_flagsは %qかフィーチャーフラグ名の配列である必要があります", allFeatureFlagsKeyword)
	}
}

// resolveは選択を、付与するフラグへ変換する。アプリケーションが定義して
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
				"feature_flagsの %qは定義されていないフィーチャーフラグです。指定できるのは %sです",
				name, joinFeatureFlagNames(model.AllFeatureFlagNames),
			)
		}
		if seen[flag] {
			return nil, fmt.Errorf("feature_flagsの %qが2回以上指定されています", name)
		}
		seen[flag] = true
		flags = append(flags, flag)
	}

	return flags, nil
}

// rosterUserは、名簿が挙げるアカウント1件。
type rosterUser struct {
	role         seedRole
	atname       string
	name         string
	email        string
	featureFlags []model.FeatureFlagName
	twoFactor    bool
}

// userRosterは実行が作成するアカウント。誰がいるのかはコードではなく設定と
// する。アドレスが個人のものであることと、アカウントが足されるのは、既存の
// アカウントでは取れない視点から画面を見るためであることによる。どのアカウントが
// 何をするのかはコードに残り、コードはアカウントを役割で引く。
type userRoster struct {
	// pathは名簿を読み込んだファイル。実行はこれを、これから空にする
	// データベースと並べて報告する。実行が何を向いているのかを1行で読み取れる
	// ようにするため。
	path string
	// passwordDigestは、アカウントが保存する形にした共通パスワード。平文は
	// 読み込みの先へは持ち越さない。実行が書き込むのはダイジェストであり、名簿の
	// 読み込み時に一度ハッシュ化していることが、平文を落とせる理由になる。
	passwordDigest string
	users          []rosterUser
}

// loadUserRosterはpathから名簿を読む。
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

// loadRosterFileはpathの名簿を、書かれたままの形で読む。中身が何であるかの
// 検査は行わない。
//
// ファイルを読むことは、名簿が読まれる2つの目的 (実行が作成するアカウントと、
// ブラウザ確認がサインインに使う資格情報) が共有する手順である。1つの関数にまとめて
// いることが、ファイルが無いときや書き間違いがあるときに、両者が同じ形で失敗し、
// 同じことを告げる理由になる。
func loadRosterFile(path string) (rosterFile, error) {
	var file rosterFile

	meta, err := toml.DecodeFile(path, &file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return rosterFile{}, fmt.Errorf("開発用ユーザーの名簿 %sがありません。%sをコピーして作成してください", path, rosterExamplePath)
		}

		return rosterFile{}, fmt.Errorf("開発用ユーザーの名簿 %sの読み込みに失敗: %w", path, err)
	}

	// デコーダが使わなかったキーは書き間違いである。その隣に書かれた値は
	// アカウントへ届かず、実行はそのキーが与えるはずだったものを欠いたまま、
	// そのアカウントを作りに行く。
	if keys := meta.Undecoded(); len(keys) > 0 {
		return rosterFile{}, fmt.Errorf("開発用ユーザーの名簿 %sに知らないキーがあります: %s", path, joinTOMLKeys(keys))
	}

	return file, nil
}

// toUserRosterは名簿を検査し、生成器が使う形にして返す。
func (f rosterFile) toUserRoster(path string) (*userRoster, error) {
	users, err := f.validate()
	if err != nil {
		return nil, err
	}

	// 名簿を読み込んでいる間に共通パスワードをハッシュ化する。bcryptが処理
	// できない入力をデータベースへ触る前に拒否できるだけでなく、各アカウントが準備
	// 済みの同じダイジェストを使えるため、ユーザー行をINSERTした後でハッシュ化の
	// 失敗が判明することも防げる。
	passwordDigest, err := auth.HashPassword(f.Password)
	if err != nil {
		return nil, fmt.Errorf("passwordのハッシュ化に失敗: %w", err)
	}

	return &userRoster{
		path:           path,
		passwordDigest: passwordDigest,
		users:          users,
	}, nil
}

// validateは名簿を検査し、そこに書かれているアカウントを返す。
//
// パスワードダイジェストの手前で止まる点がtoUserRosterとの違いになる。1件分の
// 資格情報をブラウザ確認へ渡すために名簿を読むときに要るのは書かれたままのパスワード
// であり、そこでハッシュ化しても、かかる待ち時間に見合うものが無いため。
func (f rosterFile) validate() ([]rosterUser, error) {
	if f.Password == "" {
		return nil, errors.New("passwordが空です。全アカウント共通のサインインパスワードを書いてください")
	}
	if strings.ContainsAny(f.Password, "\r\n") {
		return nil, errors.New("passwordにCR / LFは含められません。改行を含まないパスワードを書いてください")
	}
	if len(f.Users) == 0 {
		return nil, errors.New("[[users]] が1件もありません")
	}

	users := make([]rosterUser, 0, len(f.Users))
	roles := make(map[seedRole]bool, len(f.Users))
	atnames := make(map[string]bool, len(f.Users))
	emails := make(map[string]bool, len(f.Users))

	for i, entry := range f.Users {
		user, err := entry.toRosterUser()
		if err != nil {
			return nil, fmt.Errorf("%d件目の [[users]]: %w", i+1, err)
		}

		// 役割は生成器がアカウントを名指しする名前、atnameはURLに入る名前、
		// メールアドレスはブラウザ確認がサインインに使う名前である。同じ値を2度
		// 書くと、共有した名前ではどちらか一方のアカウントへ辿り着けなくなる。
		//
		// 大文字小文字だけが違う2つのatnameは同じatnameである。カラムがcitext
		// であり、2件目のアカウントが書き込まれる一意インデックスがそれらを区別
		// しないため。ここでも同じ方法で比較することで、この衝突がデータベースを
		// 空にした後のINSERT失敗として表面化することを防ぐ。メールアドレスは
		// 書かれたとおりに比較する。カラムもサインイン時の引き当ても、そう比較する
		// ため。
		if roles[user.role] {
			return nil, fmt.Errorf("役割 %sの [[users]] が2件以上あります", user.role)
		}
		atnameKey := strings.ToLower(user.atname)
		if atnames[atnameKey] {
			return nil, fmt.Errorf("atname %qの [[users]] が2件以上あります (大文字小文字の違いは同じatnameとして扱われます)", user.atname)
		}
		if emails[user.email] {
			return nil, fmt.Errorf("email %qの [[users]] が2件以上あります", user.email)
		}
		roles[user.role] = true
		atnames[atnameKey] = true
		emails[user.email] = true

		users = append(users, user)
	}

	// 生成器が名指ししているのに名簿に無い役割は、それを必要とする生成器が
	// 走るまで表面化せず、それは実行がデータベースを空にした後になる。
	for _, role := range allSeedRoles {
		if !roles[role] {
			return nil, fmt.Errorf("役割 %sの [[users]] がありません。生成器がこの役割を名指しするため、1件必要です", role)
		}
	}

	return users, nil
}

// toRosterUserは1件分を検査し、生成器が使う形にして返す。
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
			return rosterUser{}, fmt.Errorf("%sが空です", field.key)
		}
	}

	role := seedRole(e.Role)
	if !slices.Contains(allSeedRoles, role) {
		return rosterUser{}, fmt.Errorf("role %qは生成器が知らない役割です。指定できるのは %sです", e.Role, joinSeedRoles(allSeedRoles))
	}
	// atnameは、ここに置いた写しではなくアプリケーションがすべてのアカウント
	// に課している規則で検査する。名簿が受理するatnameが、アカウントが実際に
	// 持てるatnameであるようにするため。名簿の読み込み時に検査することで、
	// データベースを空にした後で初めて不正なアカウントが見つかることを防ぐ。
	if !validator.IsValidAtname(e.Atname) {
		return rosterUser{}, fmt.Errorf(
			"atname %qに使える文字は半角英数字とアンダースコアだけで、%d文字以内である必要があります",
			e.Atname, validator.AtnameMaxLength,
		)
	}
	// 名簿が受理するメールアドレスは、そのアカウントをサインインさせる
	// アドレスそのものである必要がある。名簿が保存するのは解釈した結果ではなく
	// 書かれた文字列である一方、前後の空白や表示名は解釈の過程で落ちる。どちらの
	// 書き方をしたアカウントも、同じ空白を落とすサインインフォームからは送信できない
	// アドレスを持つことになる。書かれた文字列がアドレスそのものであることを求める
	// ことで、誰も辿り着けないアカウントを作ったままシードが正常終了することを防ぐ。
	address, canonical := validator.CanonicalEmail(e.Email)
	if !canonical {
		if address == "" {
			return rosterUser{}, errors.New("emailがメールアドレスの形式ではありません")
		}

		return rosterUser{}, fmt.Errorf(
			"emailにはアドレスだけを書いてください (前後の空白や表示名は含められません)。%qのつもりであれば、そう書き直してください",
			address,
		)
	}
	if !e.FeatureFlags.present {
		return rosterUser{}, errors.New("feature_flagsがありません")
	}
	if e.TwoFactor == nil {
		return rosterUser{}, errors.New("two_factorがありません")
	}

	flags, err := e.FeatureFlags.resolve()
	if err != nil {
		return rosterUser{}, err
	}

	// nameは、照らし合わせる形式を自身では持たない唯一の必須文字列である。
	// atnameは規則が空白を弾き、emailは書かれた文字列がアドレスそのものである
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

// joinTOMLKeysは、エラーメッセージ用にキーを並べる。
func joinTOMLKeys(keys []toml.Key) string {
	ss := make([]string, 0, len(keys))
	for _, key := range keys {
		ss = append(ss, key.String())
	}

	return strings.Join(ss, ", ")
}

// joinSeedRolesは、エラーメッセージ用に役割を並べる。
func joinSeedRoles(roles []seedRole) string {
	ss := make([]string, 0, len(roles))
	for _, role := range roles {
		ss = append(ss, string(role))
	}

	return strings.Join(ss, ", ")
}

// joinFeatureFlagNamesは、エラーメッセージ用にフィーチャーフラグ名を並べる。
func joinFeatureFlagNames(names []model.FeatureFlagName) string {
	ss := make([]string, 0, len(names))
	for _, name := range names {
		ss = append(ss, string(name))
	}

	return strings.Join(ss, ", ")
}
