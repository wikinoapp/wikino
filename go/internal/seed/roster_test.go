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

// validRosterはすべての検査を通る名簿。以下のテストは、これを1箇所ずつ
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

	// パスを名簿と一緒に持つのは、実行がどのファイルを読んだのかを報告する
	// ため。
	if roster.path != path {
		t.Errorf("名簿のパスが%qであることを期待したが%qだった", path, roster.path)
	}
	// 実行が書き込むのはダイジェストであるため、ファイルに書いたパスワードで
	// それを検証することが、ファイルのパスワードが読めていることの確認になる。
	if !auth.VerifyPassword(roster.passwordDigest, "seed-password") {
		t.Error("パスワードダイジェストが名簿のパスワードと一致しない")
	}
	if len(roster.users) != 3 {
		t.Fatalf("アカウントが3件であることを期待したが%d件だった", len(roster.users))
	}

	owner := roster.users[0]
	if owner.role != roleOwner {
		t.Errorf("1件目の役割が%sであることを期待したが%sだった", roleOwner, owner.role)
	}
	if owner.atname != "seeduser1" || owner.name != "シードユーザー 1" || owner.email != "seeduser1@example.com" {
		t.Errorf("1件目の内容がファイルと一致しない: %+v", owner)
	}
	if !slices.Equal(owner.featureFlags, model.AllFeatureFlagNames) {
		t.Errorf("feature_flagsが \"all\" のとき全フラグを期待したが%vだった", owner.featureFlags)
	}
	if owner.twoFactor {
		t.Error("two_factor = falseのアカウントが2要素認証有効になっている")
	}

	collaborator := roster.users[1]
	if len(collaborator.featureFlags) != 0 {
		t.Errorf("feature_flagsが空配列のときフラグ無しを期待したが%vだった", collaborator.featureFlags)
	}
	if !collaborator.twoFactor {
		t.Error("two_factor = trueのアカウントが2要素認証無効になっている")
	}

	// アカウントはファイルが書いた順のまま保持する。上の各件を位置で読み取れる
	// のはそのためであり、この順序は実行がアカウントを作成する順と報告する順も
	// 決める。
	guest := roster.users[2]
	if guest.role != roleGuest {
		t.Errorf("3件目の役割が%sであることを期待したが%sだった", roleGuest, guest.role)
	}
}

// TestLoadUserRosterAcceptsSelectedFeatureFlagsはfeature_flagsが取りうる
// 3つ目の形を確認する。全件でも0件でもなく、そのアカウントが持つと名指しされた
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
		t.Errorf("フィーチャーフラグが%vであることを期待したが%vだった", want, roster.users[0].featureFlags)
	}
}

// TestLoadUserRosterTrimsNameは、名前の前後の空白がアカウントへ持ち込まれない
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
		t.Errorf("表示名が%qであることを期待したが%qだった", want, got)
	}
}

func TestLoadUserRosterRejectsMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed-users.toml")

	_, err := loadUserRoster(path)
	if err == nil {
		t.Fatal("名簿が無いときのエラーを期待したがnilだった")
	}

	// メッセージが見本を名指しするのは、それをコピーすることがこの状態の
	// 直し方であるため。
	if !strings.Contains(err.Error(), rosterExamplePath) {
		t.Errorf("エラーが%qを案内することを期待したが%qだった", rosterExamplePath, err)
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
			want: "passwordが空です",
		},
		{
			name: "パスワードにLFがあるとき",
			body: strings.Replace(validRoster, `password = "seed-password"`, `password = "line1\nline2"`, 1),
			want: "passwordにCR / LFは含められません",
		},
		{
			name: "パスワードにCRがあるとき",
			body: strings.Replace(validRoster, `password = "seed-password"`, `password = "line1\rline2"`, 1),
			want: "passwordにCR / LFは含められません",
		},
		{
			name: "パスワードがbcryptの上限を超えるとき",
			body: strings.Replace(validRoster, "seed-password", strings.Repeat("a", 73), 1),
			want: "passwordのハッシュ化に失敗",
		},
		{
			name: "アカウントが1件も無いとき",
			body: `password = "seed-password"`,
			want: "[[users]] が1件もありません",
		},
		{
			name: "必須項目が空のとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = ""`, 1),
			want: "atnameが空です",
		},
		{
			name: "知らない役割を指定したとき",
			body: strings.Replace(validRoster, `role = "owner"`, `role = "onwer"`, 1),
			want: "生成器が知らない役割です",
		},
		{
			// メッセージが拒否した値を名指しするのは、受理されるatnameと
			// されないatnameの違いが、ファイル上では同じに見える1文字である
			// ことがあるため。
			name: "atnameに使えない文字があるとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = "seed-user1"`, 1),
			want: `atname "seed-user1"に使える文字は半角英数字とアンダースコアだけで、20文字以内である必要があります`,
		},
		{
			name: "atnameが長すぎるとき",
			body: strings.Replace(validRoster, `atname = "seeduser1"`, `atname = "123456789012345678901"`, 1),
			want: `atname "123456789012345678901"に使える文字は半角英数字とアンダースコアだけで、20文字以内である必要があります`,
		},
		{
			name: "役割が重複しているとき",
			body: strings.Replace(validRoster, `role = "collaborator"`, `role = "owner"`, 1),
			want: "役割 ownerの [[users]] が2件以上あります",
		},
		{
			name: "atnameが重複しているとき",
			body: strings.Replace(validRoster, `atname = "seeduser2"`, `atname = "seeduser1"`, 1),
			want: `atname "seeduser1"の [[users]] が2件以上あります`,
		},
		{
			// カラムがcitextであるため、この2つは一意インデックスの同じ行に
			// 行き着く。名簿はそれを、実行がデータベースを空にした後ではなく前に
			// 告げる必要がある。
			name: "atnameが大文字小文字だけ違うとき",
			body: strings.Replace(validRoster, `atname = "seeduser2"`, `atname = "SeedUser1"`, 1),
			want: `atname "SeedUser1"の [[users]] が2件以上あります`,
		},
		{
			name: "メールアドレスの形式が不正なとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "invalid-email"`, 1),
			want: "emailがメールアドレスの形式ではありません",
		},
		{
			// この2つは解釈するとアドレスだけが取り出され、残りは落ちるが、
			// 名簿が保存するのは書かれた文字列である。どちらから作ったアカウントも
			// サインインフォームからは送信できないアドレスを持つことになるため、
			// 名簿の側で拒否する必要がある。
			name: "メールアドレスの前後に空白があるとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "seeduser1@example.com "`, 1),
			want: "emailにはアドレスだけを書いてください",
		},
		{
			name: "メールアドレスに表示名が付いているとき",
			body: strings.Replace(validRoster, `email = "seeduser1@example.com"`, `email = "シードユーザー 1 <seeduser1@example.com>"`, 1),
			want: "emailにはアドレスだけを書いてください",
		},
		{
			name: "メールアドレスが重複しているとき",
			body: strings.Replace(validRoster, `email = "seeduser2@example.com"`, `email = "seeduser1@example.com"`, 1),
			want: `email "seeduser1@example.com"の [[users]] が2件以上あります`,
		},
		{
			name: "定義されていないフィーチャーフラグを指定したとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = ["go_exmaple"]`, 1),
			want: "定義されていないフィーチャーフラグです",
		},
		{
			name: "フィーチャーフラグが重複しているとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = ["go_example", "go_example"]`, 1),
			want: "2回以上指定されています",
		},
		{
			name: "feature_flagsに知らない文字列を書いたとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = "every"`, 1),
			want: "feature_flags",
		},
		{
			name: "feature_flagsの配列要素が文字列でないとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = [1]`, 1),
			want: "feature_flagsの要素はフィーチャーフラグ名の文字列である必要があります",
		},
		{
			name: "feature_flagsが文字列でも配列でもないとき",
			body: strings.Replace(validRoster, `feature_flags = "all"`, `feature_flags = 1`, 1),
			want: `feature_flagsは "all"かフィーチャーフラグ名の配列である必要があります`,
		},
		{
			name: "feature_flagsが無いとき",
			body: strings.Replace(validRoster, "feature_flags = \"all\"\n", "", 1),
			want: "feature_flagsがありません",
		},
		{
			name: "two_factorが無いとき",
			body: strings.Replace(validRoster, "two_factor = false\n", "", 1),
			want: "two_factorがありません",
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
				t.Errorf("エラーが%qを含むことを期待したが%qだった", tt.want, err)
			}
		})
	}
}

// TestLoadUserRosterRejectsMissingRoleは、値の書き換えでは作れない唯一の
// 不正な名簿。役割ごと取り除く必要があるため。これは、そうしなければ実行が
// データベースを空にした後に踏む失敗である。
//
// 取り除くのはファイルが最後に書いている1件であるため、以下で名指しする役割は
// 特定の役割ではなくvalidRosterの末尾の役割になる。
func TestLoadUserRosterRejectsMissingRole(t *testing.T) {
	t.Parallel()

	body := validRoster[:strings.LastIndex(validRoster, "[[users]]")]

	_, err := loadUserRoster(writeRoster(t, body))
	if err == nil {
		t.Fatal("役割が欠けているときのエラーを期待したがnilだった")
	}
	if !strings.Contains(err.Error(), string(roleGuest)) {
		t.Errorf("エラーが役割%sを名指しすることを期待したが%qだった", roleGuest, err)
	}
}

// TestLoadUserRosterAcceptsExampleFileは公開リポジトリが持つファイルを確認
// する。開発者がseed-users.tomlへコピーするのはこれであり、シードへ足した役割を
// 書き込む先でもある。ここで読み込むことが、それが行われていないときにそう告げる
// ことになる。
func TestLoadUserRosterAcceptsExampleFile(t *testing.T) {
	t.Parallel()

	roster, err := loadUserRoster(filepath.Join("..", "..", rosterExamplePath))
	if err != nil {
		t.Fatalf("%sの読み込みに失敗: %v", rosterExamplePath, err)
	}

	if len(roster.users) != len(allSeedRoles) {
		t.Fatalf("見本のアカウントが%d件であることを期待したが%d件だった", len(allSeedRoles), len(roster.users))
	}

	if !auth.VerifyPassword(roster.passwordDigest, "password") {
		t.Errorf("見本のパスワードが%qであることを期待したが一致しなかった", "password")
	}

	usersByRole := make(map[seedRole]rosterUser, len(roster.users))
	for _, user := range roster.users {
		usersByRole[user.role] = user
	}

	owner := usersByRole[roleOwner]
	if owner.atname != "seeduser1" || owner.name != "シードユーザー1" || owner.email != "seeduser1@example.com" {
		t.Errorf("見本のownerの内容が期待と一致しない: %+v", owner)
	}
	if !slices.Equal(owner.featureFlags, model.AllFeatureFlagNames) {
		t.Errorf("見本のownerが全フィーチャーフラグを持つことを期待したが%vだった", owner.featureFlags)
	}
	if owner.twoFactor {
		t.Error("見本のownerが2要素認証無効であることを期待した")
	}

	collaborator := usersByRole[roleCollaborator]
	if collaborator.atname != "seeduser2" || collaborator.name != "シードユーザー2" || collaborator.email != "seeduser2@example.com" {
		t.Errorf("見本のcollaboratorの内容が期待と一致しない: %+v", collaborator)
	}
	if len(collaborator.featureFlags) != 0 {
		t.Errorf("見本のcollaboratorがフィーチャーフラグを持たないことを期待したが%vだった", collaborator.featureFlags)
	}
	if !collaborator.twoFactor {
		t.Error("見本のcollaboratorが2要素認証有効であることを期待した")
	}

	guest := usersByRole[roleGuest]
	if guest.atname != "seeduser3" || guest.name != "シードユーザー3" || guest.email != "seeduser3@example.com" {
		t.Errorf("見本のguestの内容が期待と一致しない: %+v", guest)
	}
	// guestは、自分が参加していないスペースの画面のGo版へ辿り着く。それが
	// できるのは、あれらの画面が隠れているフラグを持っている間だけである。
	if !slices.Equal(guest.featureFlags, model.AllFeatureFlagNames) {
		t.Errorf("見本のguestが全フィーチャーフラグを持つことを期待したが%vだった", guest.featureFlags)
	}
	if guest.twoFactor {
		t.Error("見本のguestが2要素認証無効であることを期待した")
	}
}

// writeRosterはテスト専用のディレクトリへ名簿ファイルを書き、そのパスを返す。
func writeRoster(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "seed-users.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("名簿の書き込みに失敗: %v", err)
	}

	return path
}
