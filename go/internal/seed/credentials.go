package seed

import (
	"fmt"
)

// Credentials are what one seeded account signs in with.
//
// [Ja] Credentials は、シードが作成したアカウント 1 件がサインインに使う値。
type Credentials struct {
	Email    string
	Password string
}

// FindCredentials returns the credentials the roster gives the account with
// role.
//
// The browser verification asks for them here rather than reading an
// environment of its own, so that the account it signs in as is the account the
// seed created. Two sources let one of them be changed alone, and the sign-in
// that follows a seed breaks without either of them being wrong on its own.
//
// [Ja] FindCredentials は、名簿が role のアカウントへ与える資格情報を返す。
//
// ブラウザ確認が自前の環境変数を読むのではなくここへ尋ねるのは、サインインする
// アカウントを、シードが作成したアカウントそのものにするため。供給元が 2 つあると
// 片方だけを変えられてしまい、どちらか一方が単体で間違っているわけでもないまま、
// シード直後のサインインが壊れる。
func FindCredentials(role string) (*Credentials, error) {
	return findCredentials(rosterPath, role)
}

// findCredentials reads the roster at path and returns the credentials for
// role. It takes the path so that tests can point it at a roster of their own,
// while callers get the one file a run reads.
//
// [Ja] findCredentials は path の名簿を読み、role の資格情報を返す。パスを受け取る
// のは、テストが自前の名簿を指せるようにするためで、呼び出し側には実行が読むのと
// 同じ 1 つのファイルが渡る。
func findCredentials(path string, role string) (*Credentials, error) {
	file, err := loadRosterFile(path)
	if err != nil {
		return nil, err
	}

	// The whole roster is checked over, not only the entry asked for. It is
	// the file the seed reads: credentials taken out of a roster the seed
	// would refuse belong to an account the database does not hold, and
	// signing in with them fails at the form instead of here, where what is
	// wrong with the file can be said.
	//
	// [Ja] 尋ねられた 1 件だけでなく、名簿全体を検査する。これはシードが読むファイル
	// であり、シードが拒否する名簿から取り出した資格情報は、データベースに存在しない
	// アカウントのものである。それを使ったサインインは、ファイルの何が問題なのかを
	// 告げられるここではなく、フォームで失敗することになる。
	users, err := file.validate()
	if err != nil {
		return nil, fmt.Errorf("開発用ユーザーの名簿 %s: %w", path, err)
	}

	for _, user := range users {
		if user.role == seedRole(role) {
			return &Credentials{Email: user.email, Password: file.Password}, nil
		}
	}

	// Every role the generators name is held by the checked roster, so what is
	// left here is a role that does not exist. Listing the roles is what turns
	// a misspelling back into the name that was meant.
	//
	// [Ja] 生成器が名指しする役割は、検査を通った名簿がすべて持っている。そのため
	// ここに残るのは、存在しない役割を指定した場合だけになる。役割を並べることが、
	// 書き間違いを、意図していた名前へ戻す手がかりになる。
	return nil, fmt.Errorf("役割 %q のアカウントは名簿にありません。指定できるのは %s です", role, joinSeedRoles(allSeedRoles))
}
