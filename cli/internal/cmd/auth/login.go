package auth

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/wikinoapp/wikino/cli/internal/apiclient"
	"github.com/wikinoapp/wikino/cli/internal/apiutil"
	"github.com/wikinoapp/wikino/cli/internal/auth"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/config"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// maxTokenBytesは、標準入力から読むトークンの上限。誤って大きなファイルを渡したときに読み続けない。
const maxTokenBytes = 4096

// loginFieldsは `auth login` の `--fields` で選べる項目。OAuthでログインしたときは付与されたスコープも出す。
var loginFields = append(append([]string{}, credentialFields...), "scopes")

type loginOptions struct {
	withToken bool
	device    bool
	scopes    []string
}

func newCmdLogin(f *cmdutil.Factory) *cobra.Command {
	loc := f.Localizer

	var opts loginOptions
	cmd := &cobra.Command{
		Use:     "login [<space>]",
		Short:   loc.T("auth_login_short"),
		Long:    loc.T("auth_login_long"),
		Example: loc.T("auth_login_example"),
		Annotations: map[string]string{
			cmdutil.AnnotationOutput: loc.T("auth_login_output"),
			cmdutil.AnnotationErrors: loc.T("auth_login_errors"),
		},
		Args: cmdutil.MaximumNArgs(loc, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(f, cmd, args, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.withToken, "with-token", false, loc.T("auth_login_with_token_flag"))
	cmd.Flags().StringSliceVar(&opts.scopes, "scopes", nil, loc.T("auth_login_scopes_flag"))
	cmd.Flags().BoolVar(&opts.device, "device", false, loc.T("auth_login_device_flag"))

	return cmd
}

func runLogin(f *cmdutil.Factory, cmd *cobra.Command, args []string, opts loginOptions) error {
	loc := f.Localizer

	// メンバーはトークンから決まるため、指定を受け付けると使われないメンバーを指定できてしまう。
	if config.ReadFlags(cmd.Flags()).Member != "" {
		return cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_member"))
	}
	// 個人アクセストークンのスコープは発行したときに決まっているため、要求するスコープを指定できない。
	if opts.withToken && cmd.Flags().Changed("scopes") {
		return cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_scopes_with_token"))
	}
	if opts.withToken && opts.device {
		return cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_device_with_token"))
	}

	space, err := spaceArg(f, cmd, args)
	if err != nil {
		return err
	}
	if space == "" {
		return cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_space_required"))
	}

	printer, err := output.NewPrinter(f.IOStreams, loc, cmd.Flags(), output.Spec{Fields: loginFields})
	if err != nil {
		return err
	}

	dir, cfg, err := loadConfig(f)
	if err != nil {
		return err
	}
	resolver := f.Resolver(cmd, cfg)
	host, err := resolver.ResolveHost()
	if err != nil {
		return err
	}
	if err := resolver.ValidateSpace("<space>", space); err != nil {
		return err
	}

	var secret auth.Secret
	// failは、トークンを得た後の失敗を返す。ブラウザでのログインなら、保存しないトークンを失効させる。
	fail := func(err error) error { return err }
	if opts.withToken {
		token, err := readToken(f, cmd)
		if err != nil {
			return err
		}
		secret = auth.Secret{AccessToken: token}
	} else {
		var discard func()
		if secret, discard, err = loginWithOAuth(f, cmd, host, space, opts); err != nil {
			return err
		}
		fail = func(err error) error {
			discard()
			return err
		}
	}

	// 保存する前に、トークンがそのスペースで使えることを確かめ、持ち主のメンバーを得る。
	client, err := f.NewAPIClient(host, apiclient.StaticToken(secret.AccessToken))
	if err != nil {
		return fail(err)
	}
	resp, err := client.GetCurrentSpaceMemberWithResponse(cmd.Context(), space)
	if err != nil {
		if opts.withToken {
			return loginAPIError(f, err)
		}
		return fail(apiutil.ConvertError(loc, err))
	}
	member := resp.JSON200
	if member == nil {
		return fail(unexpectedResponseError(f, resp.StatusCode()))
	}

	cred := config.Credential{
		Host:         host,
		Space:        space,
		MemberID:     member.Id.String(),
		MemberAtname: member.Atname,
	}
	var result credentialResult
	if err := withLockedConfig(cmd.Context(), f, dir, func(latest *config.Config) error {
		// 同じ資格情報のリフレッシュが、保存し直したトークンを古いトークンで上書きしないようにする。
		return auth.WithCredentialLock(cmd.Context(), dir, &cred, func() error {
			saved, err := saveLogin(f, dir, latest, cred, secret)
			if err != nil {
				return err
			}
			result = newCredentialResult(latest, saved, member)
			return nil
		})
	}); err != nil {
		return fail(err)
	}
	result.Scopes = secret.Scopes

	return printCredential(printer, loc, result)
}

// loginWithOAuthは、ブラウザでのOAuthの認可でトークンを得る。
// 既定はループバックで認可の応答を受け、SSHの接続中・ディスプレイが無いとき・`--device` のとき、
// ブラウザの起動やポートの待ち受けに失敗したときは、デバイス認可に切り替える。
// 要求するスコープの既定は、認可サーバーがトークンに付与できるすべてのスコープとし、
// 付与する範囲は利用者が同意画面で選ぶ。
// 返すdiscardは、得たトークンを保存せずに終えるとき、リフレッシュトークンを失効させる。
// 失敗したログインのたびに、使えない連携がサーバーに残らないようにするため。
func loginWithOAuth(f *cmdutil.Factory, cmd *cobra.Command, host, space string, opts loginOptions) (auth.Secret, func(), error) {
	loc := f.Localizer
	ctx := cmd.Context()

	client := f.OAuthClient(host)
	metadata, err := client.FetchMetadata(ctx)
	if err != nil {
		return auth.Secret{}, nil, auth.ConvertOAuthError(loc, space, err)
	}

	scopes, err := requestedScopes(f, cmd, metadata, opts.scopes)
	if err != nil {
		return auth.Secret{}, nil, err
	}

	now := time.Now
	if f.Now != nil {
		now = f.Now
	}

	useDevice := opts.device
	if !useDevice && auth.PrefersDeviceAuthorization(runtime.GOOS, f.Getenv) {
		_, _ = fmt.Fprintln(f.IOStreams.ErrOut, loc.T("auth_login_device_fallback"))
		useDevice = true
	}
	var token *auth.TokenResponse
	if !useDevice {
		token, err = client.LoginWithLoopback(ctx, metadata, auth.LoopbackOptions{
			Space:       space,
			Scopes:      scopes,
			Open:        func(authorizationURL string) error { return openAuthorizationURL(f, authorizationURL) },
			SuccessPage: loc.T("auth_login_page_success"),
			FailurePage: loc.T("auth_login_page_failure"),
			Lang:        loc.Lang(),
		})
		// 認可を要求する前に失敗しているため、デバイス認可でやり直しても二重に許可を求めない。
		if errors.Is(err, auth.ErrLoopbackUnavailable) {
			_, _ = fmt.Fprintln(f.IOStreams.ErrOut, loc.T("auth_login_device_fallback"))
			useDevice = true
		}
	}
	if useDevice {
		token, err = client.LoginWithDevice(ctx, metadata, auth.DeviceOptions{
			Space:  space,
			Scopes: scopes,
			Show:   func(device *auth.DeviceAuthorization) { showDeviceAuthorization(f, device) },
			Now:    now,
			Sleep:  f.Sleep,
		})
	}
	if err != nil {
		return auth.Secret{}, nil, auth.ConvertOAuthError(loc, space, err)
	}
	// 失効できなくても、利用者に伝えるべきなのは元のログインの失敗のため、失効の失敗は無視する。
	discard := func() { _ = client.Revoke(ctx, metadata, token.RefreshToken) }

	return auth.NewSecret(token, now()), discard, nil
}

// requestedScopesは、`--scopes` の値を認可サーバーが扱うスコープと照合する。
// 指定が無ければ、扱うすべてのスコープを要求する。
func requestedScopes(f *cmdutil.Factory, cmd *cobra.Command, metadata *auth.Metadata, requested []string) ([]string, error) {
	loc := f.Localizer
	if len(metadata.ScopesSupported) == 0 {
		return nil, auth.ConvertOAuthError(loc, "", auth.ErrInvalidMetadata)
	}
	if !cmd.Flags().Changed("scopes") {
		return metadata.ScopesSupported, nil
	}

	var scopes []string
	for _, value := range requested {
		// `--scopes "page:read draft_page:read"` のように空白で区切って渡されても受け付ける。
		for _, scope := range strings.Fields(value) {
			if !slices.Contains(metadata.ScopesSupported, scope) {
				return nil, cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_unknown_scope", map[string]any{
					"Scope": scope, "Scopes": strings.Join(metadata.ScopesSupported, ", "),
				}))
			}
			if !slices.Contains(scopes, scope) {
				scopes = append(scopes, scope)
			}
		}
	}
	if len(scopes) == 0 {
		return nil, cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_scopes_empty"))
	}

	return scopes, nil
}

// openAuthorizationURLは、認可要求のURLをブラウザで開き、標準エラーに示す。
// ブラウザが表示されないときやエージェントが利用者にURLを渡すときのため、ブラウザを開けたときもURLを示す。
// URLにはstateとコードチャレンジが入るが、コード検証子は入らないため、示しても交換には使えない。
// ブラウザを開けなければ、デバイス認可に切り替えるため、URLを示さずにエラーを返す。
func openAuthorizationURL(f *cmdutil.Factory, authorizationURL string) error {
	if f.OpenBrowser == nil {
		return errors.New("ブラウザを開く手段がありません")
	}
	if err := f.OpenBrowser(authorizationURL); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(f.IOStreams.ErrOut, f.Localizer.T("auth_login_open_browser", map[string]any{"URL": authorizationURL}))

	return nil
}

// showDeviceAuthorizationは、デバイス認可の確認用のURLとユーザーコードを標準エラーに示す。
// 標準出力はログインの結果のために空けておく。デバイスコードは秘密の値のため示さない。
func showDeviceAuthorization(f *cmdutil.Factory, device *auth.DeviceAuthorization) {
	loc := f.Localizer
	errOut := f.IOStreams.ErrOut

	_, _ = fmt.Fprintln(errOut, loc.T("auth_login_device_code", map[string]any{
		"URL":      device.VerificationURI,
		"UserCode": device.UserCode,
		"Minutes":  (device.ExpiresIn + 59) / 60,
	}))
	if device.VerificationURIComplete != "" {
		_, _ = fmt.Fprintln(errOut, loc.T("auth_login_device_code_complete", map[string]any{"URL": device.VerificationURIComplete}))
	}
}

// saveLoginは、ロック内で既存トークンを退避し、トークンと設定を更新する。
// 設定の保存に失敗した場合は既存の保存先を復元し、削除できない旧保存先は設定に残す。
func saveLogin(f *cmdutil.Factory, dir string, cfg *config.Config, cred config.Credential, secret auth.Secret) (*config.Credential, error) {
	store := f.TokenStore(dir)
	var previous *config.Credential
	var backup auth.Secret
	for _, c := range cfg.FindCredentials(cred.Host, cred.Space) {
		if c.MemberID == cred.MemberID {
			found := *c
			previous = &found
			break
		}
	}
	if previous != nil {
		var err error
		backup, err = store.Load(previous)
		if err != nil {
			var cliErr *output.Error
			// 保存済みのトークンが無い場合は、新しいトークンでログインし直せるようにする。
			if !errors.As(err, &cliErr) || cliErr.Code != output.CodeNotLoggedIn {
				return nil, authStorageError(f, "auth_error_backup", err)
			}
		}
	}
	cred.CleanupStore = ""
	var err error
	if cred.Store, err = store.Save(&cred, secret); err != nil {
		if cred.CleanupStore == auth.StoreKeyring {
			// 保存に失敗しても、時間切れのキーチェーン操作は完了する可能性がある。
			cred.Store = auth.StoreKeyring
			if rollbackErr := rollbackLogin(store, previous, backup, cred); rollbackErr != nil {
				return nil, authStorageError(f, "auth_error_rollback", errors.Join(err, rollbackErr))
			}
		}
		return nil, err
	}
	if previous != nil && cred.CleanupStore == "" {
		cred.CleanupStore = previous.CleanupStore
	}
	if previous != nil && previous.Store != cred.Store {
		cred.CleanupStore = previous.Store
	}
	// 現在の保存先を上書きできた場合、その保存先に残った旧トークンは無い。
	if cred.CleanupStore == cred.Store {
		cred.CleanupStore = ""
	}
	cfg.AddCredential(cred)
	if err := cfg.Save(dir); err != nil {
		rollbackErr := rollbackLogin(store, previous, backup, cred)
		if rollbackErr != nil {
			return nil, authStorageError(f, "auth_error_rollback", errors.Join(err, rollbackErr))
		}
		return nil, err
	}
	for _, saved := range cfg.FindCredentials(cred.Host, cred.Space) {
		if saved.MemberID != cred.MemberID {
			continue
		}
		if saved.CleanupStore != "" {
			// 新しい資格情報は保存済みのため、旧保存先の削除の失敗ではログインを失敗にしない。
			// 削除先は設定に残っており、次の再ログインかログアウトで再試行する。
			if err := store.Cleanup(saved); err != nil {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, f.Localizer.T("auth_warning_cleanup_failed"))
				return saved, nil
			}
			if err := cfg.Save(dir); err != nil {
				return nil, err
			}
		}
		return saved, nil
	}
	return nil, errors.New("保存した資格情報が見つかりません")
}

// rollbackLoginは、新しい保存先だけを取り除き、元の保存先へ退避したトークンを戻す。
// Setが時間切れになってからファイルへ切り替わった場合も、遅れて完了したSetの後で復元する。
func rollbackLogin(store *auth.Store, previous *config.Credential, backup auth.Secret, saved config.Credential) error {
	if previous == nil {
		return store.Delete(&saved)
	}
	var err error
	if previous.Store != saved.Store || backup.AccessToken == "" {
		remove := saved
		remove.CleanupStore = ""
		err = store.Delete(&remove)
	}
	// 時間切れのSetが後から書く保存先は、元の設定に記録が無ければ誰も消せなくなるため、ここで消す。
	if saved.CleanupStore != "" && saved.CleanupStore != previous.Store && saved.CleanupStore != previous.CleanupStore {
		pending := saved
		pending.Store, pending.CleanupStore = saved.CleanupStore, ""
		err = errors.Join(err, store.Delete(&pending))
	}
	if backup.AccessToken != "" && !holdsToken(store, previous, backup) {
		err = errors.Join(err, store.Restore(previous, backup))
	}
	return err
}

// holdsTokenは、元の保存先に退避したトークンが残っているかを返す。
// 保存に失敗して別の保存先へ切り替えた場合、元の保存先は変わっていないことが多く、
// 使えない保存先へ書き直して復元の失敗を通知するのを避ける。
func holdsToken(store *auth.Store, previous *config.Credential, backup auth.Secret) bool {
	current, err := store.Load(previous)
	return err == nil && current.AccessToken == backup.AccessToken
}

// authStorageErrorは、保存先の復元や削除を再試行できることを、秘密の値を含めずに通知する。
func authStorageError(f *cmdutil.Factory, messageID string, err error) *output.Error {
	return &output.Error{
		Code:     output.CodeUnexpected,
		Message:  f.Localizer.T(messageID),
		Hint:     f.Localizer.T(messageID + "_hint"),
		ExitCode: output.ExitError,
		Err:      err,
	}
}

// readTokenは、標準入力からトークンを読む。秘密の値をシェルの履歴やプロセスの一覧に残さないよう、
// フラグや引数では受け取らない。
func readToken(f *cmdutil.Factory, cmd *cobra.Command) (string, error) {
	loc := f.Localizer

	data, err := io.ReadAll(io.LimitReader(f.IOStreams.In, maxTokenBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxTokenBytes {
		return "", cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_token_invalid"))
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_token_empty"))
	}
	if strings.ContainsFunc(token, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return "", cmdutil.NewUsageError(loc, cmd, loc.T("auth_login_error_token_invalid"))
	}

	return token, nil
}

// loginAPIErrorは、トークンの検証の失敗をCLIのエラーにする。
// 401と404の既定のヒントは保存した資格情報やスペースの指定を確かめるものだが、
// ログインでは渡したトークンそのものを確かめる必要があるため差し替える。
func loginAPIError(f *cmdutil.Factory, err error) error {
	converted := apiutil.ConvertError(f.Localizer, err)

	cliErr, ok := converted.(*output.Error)
	if !ok {
		return converted
	}
	switch cliErr.Status {
	case http.StatusUnauthorized:
		cliErr.Hint = f.Localizer.T("auth_login_error_unauthorized_hint")
	case http.StatusNotFound:
		cliErr.Hint = f.Localizer.T("auth_login_error_not_found_hint")
	}

	return cliErr
}

// unexpectedResponseErrorは、成功のステータスでもJSONの本文が無い応答をCLIのエラーにする。
func unexpectedResponseError(f *cmdutil.Factory, status int) *output.Error {
	return &output.Error{
		Code:     output.CodeResponse,
		Message:  f.Localizer.T("error_response"),
		Hint:     f.Localizer.T("error_response_hint"),
		Status:   status,
		ExitCode: output.ExitError,
	}
}
