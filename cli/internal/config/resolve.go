package config

import (
	"strings"

	"github.com/spf13/pflag"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// 接続先・スペース・メンバーを選ぶフラグの名前。すべてのコマンドで同じ意味に使う。
const (
	FlagHost   = "host"
	FlagSpace  = "space"
	FlagMember = "member"
)

// CLIが使う環境変数。サーバー・Rails版では同じ名前を使わない (AGENTS.mdを参照)。
const (
	EnvToken  = "WIKINO_TOKEN"
	EnvHost   = "WIKINO_HOST"
	EnvSpace  = "WIKINO_SPACE"
	EnvMember = "WIKINO_MEMBER"
)

// AddFlagsは、接続先・スペース・メンバーを選ぶフラグをfsに足す。
func AddFlags(fs *pflag.FlagSet, loc *i18n.Localizer) {
	fs.String(FlagHost, "", loc.T("help_host_flag"))
	fs.String(FlagSpace, "", loc.T("help_space_flag"))
	fs.String(FlagMember, "", loc.T("help_member_flag"))
}

// Flagsは、接続先・スペース・メンバーのフラグの値。指定されなかったものは空。
type Flags struct {
	Host   string
	Space  string
	Member string
}

// ReadFlagsは、fsから接続先・スペース・メンバーのフラグの値を読む。fsにフラグが無ければ空にする。
func ReadFlags(fs *pflag.FlagSet) Flags {
	get := func(name string) string {
		if f := fs.Lookup(name); f != nil {
			return f.Value.String()
		}
		return ""
	}

	return Flags{Host: get(FlagHost), Space: get(FlagSpace), Member: get(FlagMember)}
}

// Targetは、コマンドがAPIを呼ぶ接続先・スペースと、使う資格情報。
type Target struct {
	Host  string
	Space string
	// EnvTokenは `WIKINO_TOKEN` の値。値があるときは保存した資格情報を使わず、Credentialはnil。
	EnvToken string
	// Credentialは使う資格情報。
	Credential *Credential
}

// Resolverは、フラグ・環境変数・設定から接続先・スペース・メンバーを決める。
type Resolver struct {
	Config *Config
	Flags  Flags
	Getenv func(string) string
	Loc    *i18n.Localizer
}

// Resolveは、コマンドがAPIを呼ぶ接続先・スペースと資格情報を決める。
//
//   - 接続先: `--host` → WIKINO_HOST → 既定の接続先 → https://wikino.app
//   - スペース: `--space` → WIKINO_SPACE → 既定のスペース → 資格情報のあるスペースが1つならそれ
//   - メンバー: `--member` → WIKINO_MEMBER → そのスペースの既定のメンバー → そのスペースの資格情報が1つならそれ
//
// WIKINO_TOKENに値があれば、保存した資格情報より優先して使う。トークンからメンバーが決まるため
// `--member` との併用は使い方の誤りにし、トークンのスペースは既定から推測せず指定を求める。
// 決まらないときは候補を並べたエラーを返す。
func (r *Resolver) Resolve() (*Target, error) {
	host, err := r.ResolveHost()
	if err != nil {
		return nil, err
	}

	if token := r.Getenv(EnvToken); token != "" {
		return r.resolveEnvToken(host, token)
	}

	return r.resolveCredential(host)
}

// ResolveCredentialは、WIKINO_TOKENを使わずに、保存した資格情報からResolveと同じ順で決める。
// 保存した資格情報を操作するコマンド (`auth logout` など) で使う。
func (r *Resolver) ResolveCredential() (*Target, error) {
	host, err := r.ResolveHost()
	if err != nil {
		return nil, err
	}

	return r.resolveCredential(host)
}

func (r *Resolver) resolveCredential(host string) (*Target, error) {
	space, err := r.resolveSpace(host)
	if err != nil {
		return nil, err
	}

	cred, err := r.ResolveMember(host, space, r.member())
	if err != nil {
		return nil, err
	}

	return &Target{Host: host, Space: space, Credential: cred}, nil
}

// ResolveHostは接続先を決め、スキームとホストだけの形にして返す。
func (r *Resolver) ResolveHost() (string, error) {
	name, raw := "--"+FlagHost, r.Flags.Host
	if raw == "" {
		name, raw = EnvHost, r.Getenv(EnvHost)
	}
	if raw == "" {
		if r.Config.Default.Host != "" {
			return r.Config.Default.Host, nil
		}
		return DefaultHost, nil
	}

	u, err := apiclient.ValidateHost(name, raw)
	if err != nil {
		return "", apiutil.ConvertError(r.Loc, err)
	}

	return u.String(), nil
}

// ValidateSpaceは、パスに埋め込むスペースの識別子を検証する。nameはエラーに載せる値の名前。
func (r *Resolver) ValidateSpace(name, space string) error {
	if err := apiclient.ValidateSpaceIdentifier(name, space); err != nil {
		return apiutil.ConvertError(r.Loc, err)
	}

	return nil
}

func (r *Resolver) resolveEnvToken(host, token string) (*Target, error) {
	if r.member() != "" {
		return nil, r.usageError(r.Loc.T("usage_error_env_token_with_member"), r.Loc.T("usage_error_env_token_with_member_hint"))
	}

	name, space := "--"+FlagSpace, r.Flags.Space
	if space == "" {
		name, space = EnvSpace, r.Getenv(EnvSpace)
	}
	if space == "" {
		return nil, r.usageError(r.Loc.T("usage_error_env_token_without_space"), r.Loc.T("usage_error_env_token_without_space_hint"))
	}
	if err := r.ValidateSpace(name, space); err != nil {
		return nil, err
	}

	return &Target{Host: host, Space: space, EnvToken: token}, nil
}

func (r *Resolver) resolveSpace(host string) (string, error) {
	name, space := "--"+FlagSpace, r.Flags.Space
	if space == "" {
		name, space = EnvSpace, r.Getenv(EnvSpace)
	}
	if space != "" {
		return space, r.ValidateSpace(name, space)
	}

	if r.Config.Default.Host == host && r.Config.Default.Space != "" {
		return r.Config.Default.Space, nil
	}

	spaces := r.Config.Spaces(host)
	switch len(spaces) {
	case 0:
		return "", r.NotLoggedInError(host, "")
	case 1:
		return spaces[0], nil
	default:
		return "", r.usageError(
			r.Loc.T("usage_error_space_ambiguous"),
			r.Loc.T("usage_error_space_ambiguous_hint", map[string]any{"Spaces": strings.Join(spaces, ", ")}),
		)
	}
}

// ResolveMemberは、hostのspaceで使う資格情報を決める。memberが空でなければアットネーム
// (大文字と小文字は区別しない) で照合し、空ならスペースの既定のメンバー、資格情報が1つならそれを選ぶ。
func (r *Resolver) ResolveMember(host, space, member string) (*Credential, error) {
	creds := r.Config.FindCredentials(host, space)
	if len(creds) == 0 {
		return nil, r.NotLoggedInError(host, space)
	}

	if member != "" {
		for _, cred := range creds {
			if strings.EqualFold(cred.MemberAtname, member) {
				return cred, nil
			}
		}

		return nil, &output.Error{
			Code:     output.CodeNotLoggedIn,
			Message:  r.Loc.T("error_member_not_logged_in", map[string]any{"Space": space, "Member": member}),
			Hint:     r.Loc.T("error_member_not_logged_in_hint", map[string]any{"Space": space, "Members": atnames(creds)}),
			ExitCode: output.ExitAuth,
		}
	}

	for _, cred := range creds {
		if cred.SpaceDefault {
			return cred, nil
		}
	}
	if len(creds) == 1 {
		return creds[0], nil
	}

	return nil, r.usageError(
		r.Loc.T("usage_error_member_ambiguous", map[string]any{"Space": space}),
		r.Loc.T("usage_error_member_ambiguous_hint", map[string]any{"Space": space, "Members": atnames(creds)}),
	)
}

func (r *Resolver) member() string {
	if r.Flags.Member != "" {
		return r.Flags.Member
	}

	return r.Getenv(EnvMember)
}

// NotLoggedInErrorは、資格情報が無いことを表すエラーを返す。spaceが空なら、接続先のどのスペースにも
// ログインしていないことを表す。ヒントのスペースの識別子は検証済みの値だけを使う。
func (r *Resolver) NotLoggedInError(host, space string) *output.Error {
	err := &output.Error{
		Code:     output.CodeNotLoggedIn,
		Message:  r.Loc.T("error_not_logged_in", map[string]any{"Host": host}),
		Hint:     r.Loc.T("error_not_logged_in_hint"),
		ExitCode: output.ExitAuth,
	}
	if space != "" {
		err.Message = r.Loc.T("error_space_not_logged_in", map[string]any{"Host": host, "Space": space})
		err.Hint = r.Loc.T("error_space_not_logged_in_hint", map[string]any{"Space": space})
	}

	return err
}

func (r *Resolver) usageError(message, hint string) *output.Error {
	return &output.Error{Code: output.CodeUsage, Message: message, Hint: hint, ExitCode: output.ExitUsage}
}

func atnames(creds []*Credential) string {
	names := make([]string, len(creds))
	for i, cred := range creds {
		names[i] = cred.MemberAtname
	}

	return strings.Join(names, ", ")
}
