package seed

import (
	"fmt"
)

// Credentialsは、シードが作成したアカウント1件がサインインに使う値。
type Credentials struct {
	Email    string
	Password string
}

// FindCredentialsは、名簿がroleのアカウントへ与える資格情報を返す。
//
// ブラウザ確認が自前の環境変数を読むのではなくここへ尋ねるのは、サインインする
// アカウントを、シードが作成したアカウントそのものにするため。供給元が2つあると
// 片方だけを変えられてしまい、どちらか一方が単体で間違っているわけでもないまま、
// シード直後のサインインが壊れる。
func FindCredentials(role string) (*Credentials, error) {
	return findCredentials(rosterPath, role)
}

// findCredentialsはpathの名簿を読み、roleの資格情報を返す。パスを受け取る
// のは、テストが自前の名簿を指せるようにするためで、呼び出し側には実行が読むのと
// 同じ1つのファイルが渡る。
func findCredentials(path string, role string) (*Credentials, error) {
	file, err := loadRosterFile(path)
	if err != nil {
		return nil, err
	}

	// 尋ねられた1件だけでなく、名簿全体を検査する。これはシードが読むファイル
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

	// 生成器が名指しする役割は、検査を通った名簿がすべて持っている。そのため
	// ここに残るのは、存在しない役割を指定した場合だけになる。役割を並べることが、
	// 書き間違いを、意図していた名前へ戻す手がかりになる。
	return nil, fmt.Errorf("役割 %qのアカウントは名簿にありません。指定できるのは %sです", role, joinSeedRoles(allSeedRoles))
}
