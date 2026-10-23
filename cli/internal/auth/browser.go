package auth

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// OpenBrowserは、システムの既定のブラウザでurlを開く。
// ブラウザを起動するコマンドの終了を待たない (ブラウザが閉じるまで戻らないものがあるため)。
func OpenBrowser(url string) error {
	// コマンドの引数として解釈されないよう、HTTPのURLだけを渡す。
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return errors.New("ブラウザで開けるのはHTTPのURLだけです")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) // #nosec G204 -- 認可サーバーのメタデータから組み立てたURLを開く
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) // #nosec G204 -- 同上
	default:
		cmd = exec.Command("xdg-open", url) // #nosec G204 -- 同上
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()

	return nil
}

// sshEnvKeysは、SSHで接続したシェルに入る環境変数。
var sshEnvKeys = []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"}

// PrefersDeviceAuthorizationは、ループバックでは認可の応答を受け取れない見込みの環境かを返す。
// SSHで接続しているときは、ブラウザが手元の端末で開くため、リダイレクトがこのコンピューターの
// 127.0.0.1に届かない。macOSとWindows以外でディスプレイが無いときは、ブラウザを開けない。
// goosは実行中のOS (runtime.GOOS)、getenvは環境変数を引く関数。
func PrefersDeviceAuthorization(goos string, getenv func(string) string) bool {
	for _, key := range sshEnvKeys {
		if getenv(key) != "" {
			return true
		}
	}
	switch goos {
	case "darwin", "windows":
		return false
	}

	return getenv("DISPLAY") == "" && getenv("WAYLAND_DISPLAY") == ""
}
