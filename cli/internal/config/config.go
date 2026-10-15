// Package configは、CLIの設定ファイルと、フラグ・環境変数・設定から接続先・スペース・メンバーを
// 解決する処理を扱う。
//
// 設定ファイルには既定の組と資格情報の一覧 (接続先・スペース・メンバー) を持ち、
// トークンそのものは持たない。トークンはauthパッケージがキーチェーンかファイルに保存する。
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// DefaultHostは、接続先を指定しないときに使う接続先。
const DefaultHost = "https://wikino.app"

// fileNameは設定ファイルの名前。
const fileName = "config.toml"

// Configは設定ファイルの内容。
type Config struct {
	// Defaultは、フラグと環境変数で指定しないときに使う接続先とスペース。
	// メンバーは、そのスペースの既定のメンバー (CredentialのSpaceDefault) を使う。
	Default Default `toml:"default"`
	// Credentialsは、資格情報を保存した接続先・スペース・メンバーの組の一覧。
	Credentials []Credential `toml:"credentials"`
}

// Defaultは既定の接続先とスペース。
type Default struct {
	Host  string `toml:"host,omitempty"`
	Space string `toml:"space,omitempty"`
}

// Credentialは、資格情報を保存した接続先・スペース・メンバーの組。
// メンバーのアットネームは変えられるため、組はメンバーのIDで識別し、
// アットネームは `--member` との照合のために持つ。
type Credential struct {
	Host         string `toml:"host"`
	Space        string `toml:"space"`
	MemberID     string `toml:"member_id"`
	MemberAtname string `toml:"member_atname"`
	// Storeはトークンの保存先 (keyringかfile)。
	Store string `toml:"store"`
	// CleanupStoreは、保存先変更時に削除できなかった保存先。ログアウトや再ログインで削除を再試行する。
	CleanupStore string `toml:"cleanup_store,omitempty"`
	// SpaceDefaultは、そのスペースの既定のメンバーかどうか。
	SpaceDefault bool `toml:"space_default,omitempty"`
	// Generationは、トークンを保存するたびに変わる乱数。保存したトークンにも同じ値を入れ、
	// 遅れて完了したキーチェーンの書き込みで別のログインのトークンに置き換わったことを検出する。
	Generation string `toml:"generation,omitempty"`
}

// Dirは設定を置くディレクトリを返す。
// `$XDG_CONFIG_HOME/wikino`、未設定なら `~/.config/wikino` とする。
func Dir(getenv func(string) string) (string, error) {
	if dir := getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "wikino"), nil
	}

	home := getenv("HOME")
	if home == "" {
		var err error
		if home, err = os.UserHomeDir(); err != nil {
			return "", fmt.Errorf("ホームディレクトリを決められません: %w", err)
		}
	}

	return filepath.Join(home, ".config", "wikino"), nil
}

// Loadはdirの設定ファイルを読む。ファイルが無ければ空の設定を返す。
func Load(dir string) (*Config, error) {
	cfg := &Config{}

	data, err := os.ReadFile(filepath.Join(dir, fileName)) // #nosec G304 G703 -- 設定のディレクトリは利用者の環境変数から決める
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, &InvalidFileError{Path: filepath.Join(dir, fileName), Err: err}
	}

	return cfg, nil
}

// InvalidFileErrorは、設定ファイルを解析できないことを表す。
// 利用者が手で直して壊すことがあるため、ConvertErrorで翻訳したエラーにして出す。
type InvalidFileError struct {
	Path string
	Err  error
}

func (e *InvalidFileError) Error() string {
	return fmt.Sprintf("設定ファイル%sを読めません: %v", e.Path, e.Err)
}

func (e *InvalidFileError) Unwrap() error { return e.Err }

// ConvertErrorは、設定ファイルを解析できないエラーを、翻訳したCLIのエラーにする。
// それ以外のエラーはそのまま返す。
func ConvertError(loc *i18n.Localizer, err error) error {
	var invalid *InvalidFileError
	if !errors.As(err, &invalid) {
		return err
	}

	return &output.Error{
		Code:     output.CodeUnexpected,
		Message:  loc.T("error_config_invalid", map[string]any{"Path": invalid.Path}),
		Hint:     loc.T("error_config_invalid_hint", map[string]any{"Path": invalid.Path}),
		ExitCode: output.ExitError,
		Err:      err,
	}
}

// Saveはcfgをdirの設定ファイルに書く。
// 書きかけのファイルを他のプロセスが読まないよう、一時ファイルに書いてから置き換える。
// 読み込んだ設定をコマンドから更新するときは、WithLocked内で呼ぶ。
func (c *Config) Save(dir string) error {
	var b strings.Builder
	if err := toml.NewEncoder(&b).Encode(c); err != nil {
		return err
	}

	return WriteFileAtomic(dir, fileName, []byte(b.String()))
}

// WriteFileAtomicは、dirのnameにdataを書く。ディレクトリは0700、ファイルは0600で作り、
// 一時ファイルに書いてから置き換えて、書きかけのファイルを他のプロセスに読ませない。
func WriteFileAtomic(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil { // #nosec G703 -- 利用者が指定した設定ディレクトリに保存する
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // #nosec G703 -- この処理で作った一時ファイルだけを削除する

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), filepath.Join(dir, name)) // #nosec G703 -- 同じ設定ディレクトリ内の一時ファイルで置き換える
}

// FindCredentialsは、hostのspaceの資格情報を返す。スペースの識別子の大文字と小文字は区別しない
// (サーバーも区別しない)。
func (c *Config) FindCredentials(host, space string) []*Credential {
	var found []*Credential
	for i := range c.Credentials {
		cred := &c.Credentials[i]
		if cred.Host == host && strings.EqualFold(cred.Space, space) {
			found = append(found, cred)
		}
	}

	return found
}

// Spacesは、hostで資格情報のあるスペースを、最初に現れた表記で重複なく返す。
func (c *Config) Spaces(host string) []string {
	var spaces []string
	for _, cred := range c.Credentials {
		if cred.Host != host || containsFold(spaces, cred.Space) {
			continue
		}
		spaces = append(spaces, cred.Space)
	}

	return spaces
}

// IsDefaultは、credが既定の組 (既定の接続先とスペースの、既定のメンバー) かを返す。
func (c *Config) IsDefault(cred *Credential) bool {
	return cred.SpaceDefault && c.Default.Host == cred.Host && strings.EqualFold(c.Default.Space, cred.Space)
}

// AddCredentialは資格情報を足す。同じ組 (接続先・スペース・メンバーのID) があれば置き換え、
// 置き換えた元を返す。
//
// 既定の組が無ければ足した組を既定にし、スペースに既定のメンバーが無ければ足したメンバーを既定にする。
// 既定の組が無いときは、スペースに既定のメンバーがいても足したメンバーへ移す。
// 既定の組のメンバーはスペースの既定のメンバーで表すため、移さないと別のメンバーが既定の組になる。
func (c *Config) AddCredential(cred Credential) (replaced *Credential) {
	sameSpace := c.FindCredentials(cred.Host, cred.Space)

	cred.SpaceDefault = true
	for _, other := range sameSpace {
		if other.MemberID == cred.MemberID {
			old := *other
			replaced = &old
			cred.SpaceDefault = other.SpaceDefault
			*other = cred
			break
		}
		if other.SpaceDefault {
			cred.SpaceDefault = false
		}
	}
	if replaced == nil {
		c.Credentials = append(c.Credentials, cred)
	}

	if c.Default.Host == "" {
		c.SetDefault(&cred)
	}

	return replaced
}

// RemoveCredentialは資格情報を取り除く。
// 既定の組だった場合は既定を外す。既定が無いときの解決は、資格情報が1つだけならそれを選ぶため、
// 残った資格情報から推測で既定を選び直さない。
func (c *Config) RemoveCredential(cred *Credential) {
	if c.IsDefault(cred) {
		c.Default = Default{}
	}

	removed := *cred
	kept := c.Credentials[:0]
	for _, other := range c.Credentials {
		if other.Host == removed.Host && strings.EqualFold(other.Space, removed.Space) && other.MemberID == removed.MemberID {
			continue
		}
		kept = append(kept, other)
	}
	c.Credentials = kept
}

// SetDefaultは、credを既定の組にし、そのスペースの既定のメンバーにする。
func (c *Config) SetDefault(cred *Credential) {
	for _, other := range c.FindCredentials(cred.Host, cred.Space) {
		other.SpaceDefault = other.MemberID == cred.MemberID
	}
	c.Default = Default{Host: cred.Host, Space: cred.Space}
}

func containsFold(values []string, s string) bool {
	for _, v := range values {
		if strings.EqualFold(v, s) {
			return true
		}
	}

	return false
}
