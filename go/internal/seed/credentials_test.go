package seed

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFindCredentials(t *testing.T) {
	t.Parallel()

	path := writeRoster(t, validRoster)

	tests := []struct {
		name      string
		role      seedRole
		wantEmail string
	}{
		{name: "管理者", role: roleOwner, wantEmail: "seeduser1@example.com"},
		{name: "共同編集者", role: roleCollaborator, wantEmail: "seeduser2@example.com"},
		{name: "非メンバー", role: roleGuest, wantEmail: "seeduser3@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			credentials, err := findCredentials(path, string(tt.role))
			if err != nil {
				t.Fatalf("資格情報の取得に失敗: %v", err)
			}

			if credentials.Email != tt.wantEmail {
				t.Errorf("メールアドレスが %q であることを期待したが %q だった", tt.wantEmail, credentials.Email)
			}
			// The password is shared by every account, so what this checks is
			// that the plaintext of the roster comes back rather than the
			// digest a run writes.
			//
			// [Ja] パスワードは全アカウント共通であるため、ここで確認しているのは、
			// 実行が書き込むダイジェストではなく名簿の平文が返ることになる。
			if credentials.Password != "seed-password" {
				t.Errorf("パスワードが名簿に書いた値であることを期待したが %q だった", credentials.Password)
			}
		})
	}
}

// TestFindCredentialsRejectsUnknownRole covers the one way a lookup fails
// without the roster being at fault: a role nobody holds. Every role the
// generators name is held by a roster that passes the checks, so a lookup that
// finds nothing was asked for a name that does not exist.
//
// [Ja] TestFindCredentialsRejectsUnknownRole は、名簿に問題が無いまま引きが失敗する
// 唯一の場合を扱う。誰も持っていない役割を尋ねた場合である。生成器が名指しする役割は
// 検査を通った名簿がすべて持っているため、何も見つからない引きは、存在しない名前を
// 尋ねられたことを意味する。
func TestFindCredentialsRejectsUnknownRole(t *testing.T) {
	t.Parallel()

	_, err := findCredentials(writeRoster(t, validRoster), "onwer")
	if err == nil {
		t.Fatal("知らない役割のエラーを期待したがnilだった")
	}

	// The message lists the roles, because a misspelling is told apart from a
	// role that was never added only by seeing what does exist.
	//
	// [Ja] メッセージが役割を並べるのは、書き間違いと、そもそも足されていない役割の
	// 区別が、実際に存在するものを見て初めて付くため。
	if !strings.Contains(err.Error(), string(roleOwner)) {
		t.Errorf("エラーが指定できる役割を並べることを期待したが %q だった", err)
	}
}

// TestFindCredentialsRejectsInvalidRoster checks that the whole roster is
// looked over, not only the entry asked for. The browser verification signs in
// as an account the seed created, and a roster the seed would refuse holds no
// such account.
//
// [Ja] TestFindCredentialsRejectsInvalidRoster は、尋ねられた 1 件だけでなく名簿
// 全体が検査されることを確認する。ブラウザ確認がサインインするのはシードが作成した
// アカウントであり、シードが拒否する名簿には、そのアカウントが存在しない。
func TestFindCredentialsRejectsInvalidRoster(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "尋ねた役割とは別の役割が重複しているとき",
			body: strings.Replace(validRoster, `role = "guest"`, `role = "collaborator"`, 1),
			want: "役割 collaborator の [[users]] が 2 件以上あります",
		},
		{
			name: "知らないキーがあるとき",
			body: strings.Replace(validRoster, "two_factor = true", "two_factory = true", 1),
			want: "知らないキー",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := findCredentials(writeRoster(t, tt.body), string(roleOwner))
			if err == nil {
				t.Fatal("エラーを期待したがnilだった")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("エラーが %q を含むことを期待したが %q だった", tt.want, err)
			}
		})
	}
}

func TestFindCredentialsRejectsMissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "seed-users.toml")

	_, err := findCredentials(path, string(roleOwner))
	if err == nil {
		t.Fatal("名簿が無いときのエラーを期待したがnilだった")
	}

	// A developer who has not set the roster up meets this error through the
	// browser verification as much as through the seed, so it has to point at
	// the same file to copy.
	//
	// [Ja] 名簿を用意していない開発者は、シードからと同じくブラウザ確認からもこの
	// エラーに出会うため、コピー元として同じファイルを案内する必要がある。
	if !strings.Contains(err.Error(), rosterExamplePath) {
		t.Errorf("エラーが %q を案内することを期待したが %q だった", rosterExamplePath, err)
	}
}
