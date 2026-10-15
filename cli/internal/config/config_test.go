package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

func TestDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "XDG_CONFIG_HOMEがあればその下", env: map[string]string{"XDG_CONFIG_HOME": "/xdg", "HOME": "/home/u"}, want: filepath.Join("/xdg", "wikino")},
		{name: "XDG_CONFIG_HOMEが無ければ~/.configの下", env: map[string]string{"HOME": "/home/u"}, want: filepath.Join("/home/u", ".config", "wikino")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Dir(envOf(tt.env))
			if err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}
			if got != tt.want {
				t.Errorf("ディレクトリ = %q、期待値 = %q", got, tt.want)
			}
		})
	}
}

func TestLoadSave(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "wikino")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("設定ファイルが無いときにエラーを期待しなかったが %v だった", err)
	}
	if len(cfg.Credentials) != 0 || cfg.Default.Host != "" {
		t.Errorf("空の設定を期待したが %+v だった", cfg)
	}

	want := testConfig()
	if err := want.Save(dir); err != nil {
		t.Fatalf("保存に失敗した: %v", err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("読み込みに失敗した: %v", err)
	}
	if got.Default != want.Default || len(got.Credentials) != len(want.Credentials) {
		t.Fatalf("読み直した設定 = %+v、期待値 = %+v", got, want)
	}
	for i := range want.Credentials {
		if got.Credentials[i] != want.Credentials[i] {
			t.Errorf("資格情報[%d] = %+v、期待値 = %+v", i, got.Credentials[i], want.Credentials[i])
		}
	}

	if runtime.GOOS == "windows" {
		return
	}
	for path, wantPerm := range map[string]os.FileMode{dir: 0o700, filepath.Join(dir, fileName): 0o600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != wantPerm {
			t.Errorf("%sのパーミッション = %o、期待値 = %o", path, info.Mode().Perm(), wantPerm)
		}
	}
}

func TestLoadInvalidFile(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "wikino")
	if err := WriteFileAtomic(dir, fileName, []byte("[default\n")); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	var invalid *InvalidFileError
	if !errors.As(err, &invalid) || invalid.Path != filepath.Join(dir, fileName) {
		t.Fatalf("設定ファイルの解析の失敗を期待したが %v だった", err)
	}

	var cliErr *output.Error
	if !errors.As(ConvertError(i18n.NewLocalizer(i18n.LangEn), err), &cliErr) || cliErr.ExitCode != output.ExitError {
		t.Fatalf("CLIのエラーへの変換を期待したが %v だった", err)
	}
	if !strings.HasPrefix(cliErr.Message, "Could not parse the configuration file ") || !strings.Contains(cliErr.Hint, invalid.Path) {
		t.Errorf("表示言語に翻訳されていない: %q / %q", cliErr.Message, cliErr.Hint)
	}

	other := errors.New("other")
	if got := ConvertError(i18n.NewLocalizer(i18n.LangEn), other); got != other {
		t.Errorf("ほかのエラー = %v、そのまま返すことを期待", got)
	}
}

func TestAddCredential(t *testing.T) {
	t.Parallel()

	cfg := &Config{}

	cfg.AddCredential(Credential{Host: DefaultHost, Space: "team", MemberID: "id-alice", MemberAtname: "alice", Store: "keyring"})
	if cfg.Default != (Default{Host: DefaultHost, Space: "team"}) {
		t.Errorf("最初のログインで既定の組にならなかった: %+v", cfg.Default)
	}
	if !cfg.Credentials[0].SpaceDefault {
		t.Error("スペースで最初のメンバーがスペースの既定にならなかった")
	}

	cfg.AddCredential(Credential{Host: DefaultHost, Space: "team", MemberID: "id-bot", MemberAtname: "bot", Store: "keyring"})
	cfg.AddCredential(Credential{Host: DefaultHost, Space: "solo", MemberID: "id-carol", MemberAtname: "carol", Store: "keyring"})
	if cfg.Default != (Default{Host: DefaultHost, Space: "team"}) {
		t.Errorf("既定の組がある後のログインで既定が変わった: %+v", cfg.Default)
	}
	if cfg.Credentials[1].SpaceDefault {
		t.Error("スペースの2人目のメンバーがスペースの既定になった")
	}
	if !cfg.Credentials[2].SpaceDefault {
		t.Error("別のスペースで最初のメンバーがスペースの既定にならなかった")
	}

	// 同じメンバーで再びログインすると、スペースの既定を保ったまま置き換える。
	replaced := cfg.AddCredential(Credential{Host: DefaultHost, Space: "TEAM", MemberID: "id-alice", MemberAtname: "alice2", Store: "file"})
	if replaced == nil || replaced.Store != "keyring" {
		t.Fatalf("置き換えた元の資格情報を期待したが %+v だった", replaced)
	}
	if len(cfg.Credentials) != 3 {
		t.Fatalf("資格情報の数 = %d、期待値 = 3", len(cfg.Credentials))
	}
	if got := cfg.Credentials[0]; got.MemberAtname != "alice2" || got.Store != "file" || !got.SpaceDefault {
		t.Errorf("置き換えた資格情報 = %+v", got)
	}
}

func TestAddCredentialWithoutDefaultMakesLoggedInMemberDefault(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	cfg.AddCredential(Credential{Host: DefaultHost, Space: "team", MemberID: "id-alice", MemberAtname: "alice", Store: "keyring"})
	cfg.AddCredential(Credential{Host: DefaultHost, Space: "team", MemberID: "id-bot", MemberAtname: "bot", Store: "keyring"})
	cfg.AddCredential(Credential{Host: DefaultHost, Space: "solo", MemberID: "id-carol", MemberAtname: "carol", Store: "keyring"})
	cfg.SetDefault(&cfg.Credentials[2])
	carol := cfg.Credentials[2]
	cfg.RemoveCredential(&carol)
	if cfg.Default != (Default{}) {
		t.Fatalf("既定の組を消しても既定が残った: %+v", cfg.Default)
	}

	// 既定が無い状態でスペースの既定でないメンバーがログインすると、そのメンバーが既定の組になる。
	cfg.AddCredential(Credential{Host: DefaultHost, Space: "team", MemberID: "id-bot", MemberAtname: "bot", Store: "keyring"})
	alice, bot := cfg.Credentials[0], cfg.Credentials[1]
	if !cfg.IsDefault(&bot) {
		t.Errorf("ログインしたメンバーが既定の組にならなかった: %+v / %+v", cfg.Default, bot)
	}
	if alice.SpaceDefault {
		t.Error("元のメンバーがスペースの既定のまま残った")
	}
}

func TestRemoveCredential(t *testing.T) {
	t.Parallel()

	cfg := testConfig()

	cfg.RemoveCredential(&cfg.Credentials[1])
	if cfg.Default.Space != "team" || len(cfg.Credentials) != 3 {
		t.Errorf("既定でない資格情報を消して既定が変わった: %+v", cfg)
	}

	cfg.RemoveCredential(&cfg.Credentials[0])
	if cfg.Default != (Default{}) {
		t.Errorf("既定の組を消しても既定が残った: %+v", cfg.Default)
	}
	if len(cfg.FindCredentials(DefaultHost, "team")) != 0 {
		t.Error("資格情報が消えていない")
	}
}

func TestSetDefault(t *testing.T) {
	t.Parallel()

	cfg := testConfig()

	cfg.SetDefault(&cfg.Credentials[1])
	if cfg.Default != (Default{Host: DefaultHost, Space: "team"}) {
		t.Errorf("既定の組 = %+v", cfg.Default)
	}
	if cfg.Credentials[0].SpaceDefault || !cfg.Credentials[1].SpaceDefault {
		t.Errorf("スペースの既定のメンバーが切り替わっていない: %+v", cfg.Credentials[:2])
	}
	if !cfg.IsDefault(&cfg.Credentials[1]) || cfg.IsDefault(&cfg.Credentials[2]) {
		t.Error("IsDefaultの判定が違う")
	}

	cfg.SetDefault(&cfg.Credentials[3])
	if cfg.Default != (Default{Host: devHost, Space: "dev"}) {
		t.Errorf("別の接続先に切り替えた既定の組 = %+v", cfg.Default)
	}
	if !cfg.Credentials[1].SpaceDefault {
		t.Error("別のスペースに切り替えて、元のスペースの既定のメンバーが外れた")
	}
}
