package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

const devHost = "http://localhost:4000"

// testConfigは、2つのスペースに資格情報を持つ設定を返す。
// スペースteamには2人のメンバーがいて、aliceがそのスペースの既定のメンバー。
func testConfig() *Config {
	return &Config{
		Default: Default{Host: DefaultHost, Space: "team"},
		Credentials: []Credential{
			{Host: DefaultHost, Space: "team", MemberID: "id-alice", MemberAtname: "alice", Store: "file", SpaceDefault: true},
			{Host: DefaultHost, Space: "team", MemberID: "id-bot", MemberAtname: "bot", Store: "file"},
			{Host: DefaultHost, Space: "solo", MemberID: "id-carol", MemberAtname: "carol", Store: "file", SpaceDefault: true},
			{Host: devHost, Space: "dev", MemberID: "id-dave", MemberAtname: "dave", Store: "file", SpaceDefault: true},
		},
	}
}

func envOf(env map[string]string) func(string) string {
	return func(key string) string { return env[key] }
}

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     func() *Config
		flags      Flags
		env        map[string]string
		wantHost   string
		wantSpace  string
		wantMember string
		wantToken  string
	}{
		{
			name:       "指定が無ければ既定の組を使う",
			config:     testConfig,
			wantHost:   DefaultHost,
			wantSpace:  "team",
			wantMember: "alice",
		},
		{
			name:       "--memberでスペースの別のメンバーを選ぶ",
			config:     testConfig,
			flags:      Flags{Member: "bot"},
			wantHost:   DefaultHost,
			wantSpace:  "team",
			wantMember: "bot",
		},
		{
			name:       "--memberは大文字と小文字を区別しない",
			config:     testConfig,
			flags:      Flags{Member: "BOT"},
			wantHost:   DefaultHost,
			wantSpace:  "team",
			wantMember: "bot",
		},
		{
			name:       "--spaceは大文字と小文字を区別しない",
			config:     testConfig,
			flags:      Flags{Space: "Solo"},
			wantHost:   DefaultHost,
			wantSpace:  "Solo",
			wantMember: "carol",
		},
		{
			name:       "--spaceはWIKINO_SPACEより優先する",
			config:     testConfig,
			flags:      Flags{Space: "solo"},
			env:        map[string]string{EnvSpace: "team"},
			wantHost:   DefaultHost,
			wantSpace:  "solo",
			wantMember: "carol",
		},
		{
			name:       "WIKINO_SPACEは既定のスペースより優先する",
			config:     testConfig,
			env:        map[string]string{EnvSpace: "solo"},
			wantHost:   DefaultHost,
			wantSpace:  "solo",
			wantMember: "carol",
		},
		{
			name:       "--memberはWIKINO_MEMBERより優先する",
			config:     testConfig,
			flags:      Flags{Member: "alice"},
			env:        map[string]string{EnvMember: "bot"},
			wantHost:   DefaultHost,
			wantSpace:  "team",
			wantMember: "alice",
		},
		{
			name:       "WIKINO_MEMBERはスペースの既定のメンバーより優先する",
			config:     testConfig,
			env:        map[string]string{EnvMember: "bot"},
			wantHost:   DefaultHost,
			wantSpace:  "team",
			wantMember: "bot",
		},
		{
			name:       "既定の接続先と違う接続先では既定のスペースを使わず、資格情報のある唯一のスペースを使う",
			config:     testConfig,
			flags:      Flags{Host: devHost},
			wantHost:   devHost,
			wantSpace:  "dev",
			wantMember: "dave",
		},
		{
			name:       "--hostはWIKINO_HOSTより優先する",
			config:     testConfig,
			flags:      Flags{Host: devHost + "/"},
			env:        map[string]string{EnvHost: "https://other.example.com"},
			wantHost:   devHost,
			wantSpace:  "dev",
			wantMember: "dave",
		},
		{
			name: "既定が無くても資格情報が1つならそれを使う",
			config: func() *Config {
				return &Config{Credentials: []Credential{
					{Host: DefaultHost, Space: "solo", MemberID: "id-carol", MemberAtname: "carol"},
				}}
			},
			wantHost:   DefaultHost,
			wantSpace:  "solo",
			wantMember: "carol",
		},
		{
			name:      "WIKINO_TOKENは保存した資格情報より優先する",
			config:    testConfig,
			env:       map[string]string{EnvToken: "secret", EnvSpace: "other"},
			wantHost:  DefaultHost,
			wantSpace: "other",
			wantToken: "secret",
		},
		{
			name:      "WIKINO_TOKENは資格情報が無くても使える",
			config:    func() *Config { return &Config{} },
			flags:     Flags{Space: "other"},
			env:       map[string]string{EnvToken: "secret"},
			wantHost:  DefaultHost,
			wantSpace: "other",
			wantToken: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Resolver{Config: tt.config(), Flags: tt.flags, Getenv: envOf(tt.env), Loc: i18n.NewLocalizer(i18n.LangJa)}
			target, err := r.Resolve()
			if err != nil {
				t.Fatalf("エラーを期待しなかったが %v だった", err)
			}

			if target.Host != tt.wantHost || target.Space != tt.wantSpace || target.EnvToken != tt.wantToken {
				t.Errorf("接続先・スペース・トークン = %q, %q, %q、期待値 = %q, %q, %q",
					target.Host, target.Space, target.EnvToken, tt.wantHost, tt.wantSpace, tt.wantToken)
			}

			gotMember := ""
			if target.Credential != nil {
				gotMember = target.Credential.MemberAtname
			}
			if gotMember != tt.wantMember {
				t.Errorf("メンバー = %q、期待値 = %q", gotMember, tt.wantMember)
			}
		})
	}
}

func TestResolveErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       func() *Config
		flags        Flags
		env          map[string]string
		wantCode     string
		wantExitCode int
		wantHint     string
	}{
		{
			name:         "資格情報が無ければ未ログインで失敗する",
			config:       func() *Config { return &Config{} },
			wantCode:     output.CodeNotLoggedIn,
			wantExitCode: output.ExitAuth,
			wantHint:     "wikino auth login <space>`",
		},
		{
			name: "同じスペースに複数のメンバーがあり既定も無ければ、候補を並べて失敗する",
			config: func() *Config {
				cfg := testConfig()
				cfg.Credentials[0].SpaceDefault = false
				return cfg
			},
			wantCode:     output.CodeUsage,
			wantExitCode: output.ExitUsage,
			wantHint:     "資格情報のあるメンバー: alice, bot",
		},
		{
			name: "複数のスペースがあり既定も無ければ、候補を並べて失敗する",
			config: func() *Config {
				cfg := testConfig()
				cfg.Default = Default{}
				return cfg
			},
			wantCode:     output.CodeUsage,
			wantExitCode: output.ExitUsage,
			wantHint:     "資格情報のあるスペース: team, solo",
		},
		{
			name:         "指定したメンバーの資格情報が無ければ、候補を並べて失敗する",
			config:       testConfig,
			flags:        Flags{Member: "nobody"},
			wantCode:     output.CodeNotLoggedIn,
			wantExitCode: output.ExitAuth,
			wantHint:     "資格情報のあるメンバー: alice, bot",
		},
		{
			name:         "指定したスペースの資格情報が無ければ失敗する",
			config:       testConfig,
			flags:        Flags{Space: "unknown"},
			wantCode:     output.CodeNotLoggedIn,
			wantExitCode: output.ExitAuth,
			wantHint:     "wikino auth login unknown`",
		},
		{
			name:         "WIKINO_TOKENと--memberの併用は使い方の誤り",
			config:       testConfig,
			flags:        Flags{Space: "team", Member: "alice"},
			env:          map[string]string{EnvToken: "secret"},
			wantCode:     output.CodeUsage,
			wantExitCode: output.ExitUsage,
		},
		{
			name:         "WIKINO_TOKENとWIKINO_MEMBERの併用は使い方の誤り",
			config:       testConfig,
			env:          map[string]string{EnvToken: "secret", EnvSpace: "team", EnvMember: "alice"},
			wantCode:     output.CodeUsage,
			wantExitCode: output.ExitUsage,
		},
		{
			name:         "WIKINO_TOKENは既定のスペースを使わず、スペースの指定が無ければ失敗する",
			config:       testConfig,
			env:          map[string]string{EnvToken: "secret"},
			wantCode:     output.CodeUsage,
			wantExitCode: output.ExitUsage,
			wantHint:     "WIKINO_SPACE",
		},
		{
			name:         "HTTPSでもループバックでもない接続先は使い方の誤り",
			config:       testConfig,
			flags:        Flags{Host: "http://example.com"},
			wantCode:     output.CodeUsage,
			wantExitCode: output.ExitUsage,
		},
		{
			name:         "パスに使えない文字を含むスペースは使い方の誤り",
			config:       testConfig,
			flags:        Flags{Space: "../team"},
			wantCode:     output.CodeUsage,
			wantExitCode: output.ExitUsage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &Resolver{Config: tt.config(), Flags: tt.flags, Getenv: envOf(tt.env), Loc: i18n.NewLocalizer(i18n.LangJa)}
			_, err := r.Resolve()

			var cliErr *output.Error
			if !errors.As(err, &cliErr) {
				t.Fatalf("CLIのエラーを期待したが %v だった", err)
			}
			if cliErr.Code != tt.wantCode || cliErr.ExitCode != tt.wantExitCode {
				t.Errorf("コード・終了コード = %q, %d、期待値 = %q, %d", cliErr.Code, cliErr.ExitCode, tt.wantCode, tt.wantExitCode)
			}
			if !strings.Contains(cliErr.Hint, tt.wantHint) {
				t.Errorf("ヒントに %q を含むことを期待したが %q だった", tt.wantHint, cliErr.Hint)
			}
		})
	}
}

func TestResolveCredentialIgnoresEnvToken(t *testing.T) {
	t.Parallel()

	r := &Resolver{
		Config: testConfig(),
		Getenv: envOf(map[string]string{EnvToken: "secret"}),
		Loc:    i18n.NewLocalizer(i18n.LangJa),
	}
	target, err := r.ResolveCredential()
	if err != nil {
		t.Fatalf("エラーを期待しなかったが %v だった", err)
	}
	if target.EnvToken != "" || target.Credential == nil || target.Credential.MemberAtname != "alice" {
		t.Errorf("保存した既定の資格情報を期待したが %+v だった", target)
	}
}
