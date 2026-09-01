package seed

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// validRoster is a roster that passes every check, which the tests below break
// one part of at a time.
//
// [Ja] validRoster はすべての検査を通る名簿。以下のテストは、これを 1 箇所ずつ
// 壊して確認する。
const validRoster = `
password = "seed-password"

[[users]]
role = "owner"
atname = "seeduser1"
name = "シードユーザー 1"
email = "seeduser1@example.com"
feature_flags = "all"
two_factor = false

[[users]]
role = "collaborator"
atname = "seeduser2"
name = "シードユーザー 2"
email = "seeduser2@example.com"
feature_flags = []
two_factor = true

[[users]]
role = "guest"
atname = "seeduser3"
name = "シードユーザー 3"
email = "seeduser3@example.com"
feature_flags = "all"
two_factor = false
`

func TestLoadUserRoster(t *testing.T) {
	t.Parallel()

	path := writeRoster(t, validRoster)

	roster, err := loadUserRoster(path)
	if err != nil {
		t.Fatalf("名簿の読み込みに失敗: %v", err)
	}

	// The path travels with the roster because a run reports which file it
	// read.
	//
	// [Ja] パスを名簿と一緒に持つのは、実行がどのファイルを読んだのかを報告する
	// ため。
	if roster.path != path {
		t.Errorf("名簿のパスが %q であることを期待したが %q だった", path, roster.path)
	}
	// What a run writes is the digest, so verifying it against the password
	// written in the file is what says the file's password was read.
	//
	// [Ja] 実行が書き込むのはダイジェストであるため、ファイルに書いたパスワードで
	// それを検証することが、ファイルのパスワードが読めていることの確認になる。
	if !auth.VerifyPassword(roster.passwordDigest, "seed-password") {
		t.Error("パスワードダイジェストが名簿のパスワードと一致しない")
	}
	if len(roster.users) != 3 {
		t.Fatalf("アカウントが 3 件であることを期待したが %d 件だった", len(roster.users))
	}

	owner := roster.users[0]
	if owner.role != roleOwner {
		t.Errorf("1 件目の役割が %s であることを期待したが %s だった", roleOwner, owner.role)
	}
	if owner.atname != "seeduser1" || owner.name != "シードユーザー 1" || owner.email != "seeduser1@example.com" {
		t.Errorf("1 件目の内容がファイルと一致しない: %+v", owner)
	}
	if !slices.Equal(owner.featureFlags, model.AllFeatureFlagNames) {
		t.Errorf("feature_flags が \"all\" のとき全フラグを期待したが %v だった", owner.featureFlags)
	}
	if owner.twoFactor {
		t.Error("two_factor = false のアカウントが 2 要素認証有効になっている")
	}

	collaborator := roster.users[1]
	if len(collaborator.featureFlags) != 0 {
		t.Errorf("feature_flags が空配列のときフラグ無しを期待したが %v だった", collaborator.featureFlags)
	}
	if !collaborator.twoFactor {
		t.Error("two_factor = true のアカウントが 2 要素認証無効になっている")
	}

	// The accounts are kept in the order the file writes them, which is what
	// lets the entries above be read off by position. The order also decides
	// the order a run creates them in and reports them in.
	//
	// [Ja] アカウントはファイルが書いた順のまま保持する。上の各件を位置で読み取れる
	// のはそのためであり、この順序は実行がアカウントを作成する順と報告する順も
	// 決める。
	guest := roster.users[2]
	if guest.role != roleGuest {
		t.Errorf("3 件目の役割が %s であることを期待したが %s だった", roleGuest, guest.role)
	}
}

// TestLoadUserRosterAcceptsSelectedFeatureFlags checks the third shape
// feature_flags can take: neither every flag nor none of them, but the ones an
// account is named to hold.
//
// [Ja] TestLoadUserRosterAcceptsSelectedFeatureFlags は feature_flags が取りうる
// 3 つ目の形を確認する。全件でも 0 件でもなく、そのアカウントが持つと名指しされた
// フラグである。
func TestLoadUserRosterAcceptsSelectedFeatureFlags(t *testing.T) {
	t.Parallel()

	body := strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = ["go_example"]`, 1)

	roster, err := loadUserRoster(writeRoster(t, body))
	if err != nil {
		t.Fatalf("名簿の読み込みに失敗: %v", err)
	}

	want := []model.FeatureFlagName{model.FeatureFlagExample}
	if !slices.Equal(roster.users[0].featureFlags, want) {
		t.Errorf("フィーチャーフラグが %v であることを期待したが %v だった", want, roster.users[0].featureFlags)
	}
}

// TestLoadUserRosterTrimsName checks that whitespace around a name does not
// travel into the account. The name is the one required string with no format
// of its own, so nothing else would catch a stray space, and it would show up
// on every screen the account appears on.
//
// [Ja] TestLoadUserRosterTrimsName は、名前の前後の空白がアカウントへ持ち込まれない
// ことを確認する。名前は自身の形式を持たない唯一の必須文字列であるため、紛れ込んだ
// 空白を他の検査が捕まえることはなく、そのアカウントが現れるすべての画面に出てしまう。
func TestLoadUserRosterTrimsName(t *testing.T) {
	t.Parallel()

	body := strings.Replace(validRoster, `name = "シードユーザー 1"`, `name = "  シードユーザー 1  "`, 1)

	roster, err := loadUserRoster(writeRoster(t, body))
	if err != nil {
		t.Fatalf("名簿の読み込みに失敗: %v", err)
	}

	want := "シードユーザー 1"
	if got := roster.users[0].name; got != want {
		t.Errorf("表示名が %q であることを期待したが %q だった", want, got)
	}
}

func TestLoadUserRosterRejectsMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed-users.toml")

	_, err := loadUserRoster(path)
	if err == nil {
		t.Fatal("名簿が無いときのエラーを期待したがnilだった")
	}

	// The message names the example, because copying it is what fixes this.
	//
	// [Ja] メッセージが見本を名指しするのは、それをコピーすることがこの状態の
	// 直し方であるため。
	if !strings.Contains(err.Error(), rosterExamplePath) {
		t.Errorf("エラーが %q を案内することを期待したが %q だった", rosterExamplePath, err)
	}
}

func TestLoadUserRosterRejectsInvalidRoster(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "TOMLとして読めないとき",
			body: "password = ",
			want: "読み込みに失敗",
		},
		{
			name: "知らないキーがあるとき",
			body: strings.Replace(validRoster, "two_factor = true", "two_factory = true", 1),
			want: "知らないキー",
		},
		{
			name: "パスワードが空のとき",
			body: strings.Replace(validRoster, `password = "seed-password"`, `password = ""`, 1),
			want: "password が空です",
		},
		{
			name: "パスワードにLFがあるとき",
			body: strings.Replace(validRoster, `password = "seed-password"`, `password = "line1\nline2"`, 1),
			want: "password に CR / LF は含められません",
		},
		{
			name: "パスワードにCRがあるとき",
			body: strings.Replace(validRoster, `password = "seed-password"`, `password = "line1\rline2"`, 1),
			want: "password に CR / LF は含められません",
		},
		{
			name: "パスワードがbcryptの上限を超えるとき",
			body: strings.Replace(validRoster, "seed-password", strings.Repeat("a", 73), 1),
			want: "password のハッシュ化に失敗",
		},
		{
			name: "アカウントが1件も無いとき",
			body: `password = "seed-password"`,
			want: "[[users]] が 1 件もありません",
		},
		{
			name: "必須項目が空のとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = ""`, 1),
			want: "atname が空です",
		},
		{
			name: "知らない役割を指定したとき",
			body: strings.Replace(validRoster, `role = "owner"`, `role = "onwer"`, 1),
			want: "生成器が知らない役割です",
		},
		{
			// The message names the value it refused, because the difference
			// between an accepted and a refused atname can be a single
			// character that reads the same in the file.
			//
			// [Ja] メッセージが拒否した値を名指しするのは、受理される atname と
			// されない atname の違いが、ファイル上では同じに見える 1 文字である
			// ことがあるため。
			name: "atnameに使えない文字があるとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = "seed-user1"`, 1),
			want: `atname "seed-user1" に使える文字は半角英数字とアンダースコアだけで、20 文字以内である必要があります`,
		},
		{
			name: "atnameが長すぎるとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = "123456789012345678901"`, 1),
			want: `atname "123456789012345678901" に使える文字は半角英数字とアンダースコアだけで、20 文字以内である必要があります`,
		},
		{
			name: "役割が重複しているとき",
			body: strings.Replace(validRoster, `role = "collaborator"`, `role = "owner"`, 1),
			want: "役割 owner の [[users]] が 2 件以上あります",
		},
		{
			name: "atnameが重複しているとき",
			body: strings.Replace(validRoster, `atname = "seeduser2"`, `atname = "seeduser1"`, 1),
			want: `atname "seeduser1" の [[users]] が 2 件以上あります`,
		},
		{
			// The column is citext, so these two reach the same row of the
			// unique index. The roster has to say so before the run empties
			// the database rather than after.
			//
			// [Ja] カラムが citext であるため、この 2 つは一意インデックスの同じ行に
			// 行き着く。名簿はそれを、実行がデータベースを空にした後ではなく前に
			// 告げる必要がある。
			name: "atnameが大文字小文字だけ違うとき",
			body: strings.Replace(validRoster, `atname = "seeduser2"`, `atname = "SeedUser1"`, 1),
			want: `atname "SeedUser1" の [[users]] が 2 件以上あります`,
		},
		{
			name: "メールアドレスの形式が不正なとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "invalid-email"`, 1),
			want: "email がメールアドレスの形式ではありません",
		},
		{
			// Parsing takes the address out of these two and leaves the rest
			// behind, but the roster stores what is written. An account made
			// from either would hold an address that the sign-in form cannot
			// submit, so the roster has to refuse them.
			//
			// [Ja] この 2 つは解釈するとアドレスだけが取り出され、残りは落ちるが、
			// 名簿が保存するのは書かれた文字列である。どちらから作ったアカウントも
			// サインインフォームからは送信できないアドレスを持つことになるため、
			// 名簿の側で拒否する必要がある。
			name: "メールアドレスの前後に空白があるとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "seeduser1@example.com "`, 1),
			want: "email にはアドレスだけを書いてください",
		},
		{
			name: "メールアドレスに表示名が付いているとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "シードユーザー 1 <seeduser1@example.com>"`, 1),
			want: "email にはアドレスだけを書いてください",
		},
		{
			name: "メールアドレスが重複しているとき",
			body: strings.Replace(validRoster, `email = "seeduser2@example.com"`, `email = "seeduser1@example.com"`, 1),
			want: `email "seeduser1@example.com" の [[users]] が 2 件以上あります`,
		},
		{
			name: "定義されていないフィーチャーフラグを指定したとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = ["go_exmaple"]`, 1),
			want: "定義されていないフィーチャーフラグです",
		},
		{
			name: "フィーチャーフラグが重複しているとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = ["go_example", "go_example"]`, 1),
			want: "2 回以上指定されています",
		},
		{
			name: "feature_flagsに知らない文字列を書いたとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = "every"`, 1),
			want: "feature_flags",
		},
		{
			name: "feature_flagsの配列要素が文字列でないとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = [1]`, 1),
			want: "feature_flags の要素はフィーチャーフラグ名の文字列である必要があります",
		},
		{
			name: "feature_flagsが文字列でも配列でもないとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = 1`, 1),
			want: `feature_flags は "all" かフィーチャーフラグ名の配列である必要があります`,
		},
		{
			name: "feature_flagsが無いとき",
			body: strings.Replace(validRoster, "feature_flags = \"all\"\n", "", 1),
			want: "feature_flags がありません",
		},
		{
			name: "two_factorが無いとき",
			body: strings.Replace(validRoster, "two_factor = false\n", "", 1),
			want: "two_factor がありません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := loadUserRoster(writeRoster(t, tt.body))
			if err == nil {
				t.Fatal("エラーを期待したがnilだった")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("エラーが %q を含むことを期待したが %q だった", tt.want, err)
			}
		})
	}
}

// TestLoadUserRosterRejectsMissingRole is the one invalid roster that cannot be
// made by editing a value: a role has to be removed for it. It is the failure a
// run would otherwise hit after emptying the database.
//
// The entry removed is the last one the file writes, so the role named below is
// the last of validRoster rather than any particular one.
//
// [Ja] TestLoadUserRosterRejectsMissingRole は、値の書き換えでは作れない唯一の
// 不正な名簿。役割ごと取り除く必要があるため。これは、そうしなければ実行が
// データベースを空にした後に踏む失敗である。
//
// 取り除くのはファイルが最後に書いている 1 件であるため、以下で名指しする役割は
// 特定の役割ではなく validRoster の末尾の役割になる。
func TestLoadUserRosterRejectsMissingRole(t *testing.T) {
	t.Parallel()

	body := validRoster[:strings.LastIndex(validRoster, "[[users]]")]

	_, err := loadUserRoster(writeRoster(t, body))
	if err == nil {
		t.Fatal("役割が欠けているときのエラーを期待したがnilだった")
	}
	if !strings.Contains(err.Error(), string(roleGuest)) {
		t.Errorf("エラーが役割 %s を名指しすることを期待したが %q だった", roleGuest, err)
	}
}

// TestLoadUserRosterAcceptsExampleFile checks the file the public repository
// carries. It is what a developer copies to seed-users.toml, and what a role
// added to the seed has to be written into as well; loading it here is what
// says so when it has not been.
//
// [Ja] TestLoadUserRosterAcceptsExampleFile は公開リポジトリが持つファイルを確認
// する。開発者が seed-users.toml へコピーするのはこれであり、シードへ足した役割を
// 書き込む先でもある。ここで読み込むことが、それが行われていないときにそう告げる
// ことになる。
func TestLoadUserRosterAcceptsExampleFile(t *testing.T) {
	t.Parallel()

	roster, err := loadUserRoster(filepath.Join("..", "..", rosterExamplePath))
	if err != nil {
		t.Fatalf("%s の読み込みに失敗: %v", rosterExamplePath, err)
	}

	if len(roster.users) != len(allSeedRoles) {
		t.Fatalf("見本のアカウントが %d 件であることを期待したが %d 件だった", len(allSeedRoles), len(roster.users))
	}

	if !auth.VerifyPassword(roster.passwordDigest, "password") {
		t.Errorf("見本のパスワードが %q であることを期待したが一致しなかった", "password")
	}

	usersByRole := make(map[seedRole]rosterUser, len(roster.users))
	for _, user := range roster.users {
		usersByRole[user.role] = user
	}

	owner := usersByRole[roleOwner]
	if owner.atname != "seeduser1" || owner.name != "シードユーザー 1" || owner.email != "seeduser1@example.com" {
		t.Errorf("見本の owner の内容が期待と一致しない: %+v", owner)
	}
	if !slices.Equal(owner.featureFlags, model.AllFeatureFlagNames) {
		t.Errorf("見本の owner が全フィーチャーフラグを持つことを期待したが %v だった", owner.featureFlags)
	}
	if owner.twoFactor {
		t.Error("見本の owner が 2 要素認証無効であることを期待した")
	}

	collaborator := usersByRole[roleCollaborator]
	if collaborator.atname != "seeduser2" || collaborator.name != "シードユーザー 2" || collaborator.email != "seeduser2@example.com" {
		t.Errorf("見本の collaborator の内容が期待と一致しない: %+v", collaborator)
	}
	if len(collaborator.featureFlags) != 0 {
		t.Errorf("見本の collaborator がフィーチャーフラグを持たないことを期待したが %v だった", collaborator.featureFlags)
	}
	if !collaborator.twoFactor {
		t.Error("見本の collaborator が 2 要素認証有効であることを期待した")
	}

	guest := usersByRole[roleGuest]
	if guest.atname != "seeduser3" || guest.name != "シードユーザー 3" || guest.email != "seeduser3@example.com" {
		t.Errorf("見本の guest の内容が期待と一致しない: %+v", guest)
	}
	// The guest reaches the Go version of a screen of a space it has not
	// joined, which it can only do while it holds the flags those screens sit
	// behind.
	//
	// [Ja] guest は、自分が参加していないスペースの画面の Go 版へ辿り着く。それが
	// できるのは、あれらの画面が隠れているフラグを持っている間だけである。
	if !slices.Equal(guest.featureFlags, model.AllFeatureFlagNames) {
		t.Errorf("見本の guest が全フィーチャーフラグを持つことを期待したが %v だった", guest.featureFlags)
	}
	if guest.twoFactor {
		t.Error("見本の guest が 2 要素認証無効であることを期待した")
	}
}

// writeRoster writes a roster file into a directory of the test's own and
// returns its path.
//
// [Ja] writeRoster はテスト専用のディレクトリへ名簿ファイルを書き、そのパスを返す。
func writeRoster(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "seed-users.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("名簿の書き込みに失敗: %v", err)
	}

	return path
}
