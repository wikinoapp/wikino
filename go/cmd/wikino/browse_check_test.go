package main

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const browseEnvironmentHarness = `
set -euo pipefail

source "$WIKINO_BROWSE_TEST_SCRIPT"

command() {
  if [[ "$1" == "-v" && "${2:-}" == "${WIKINO_BROWSE_TEST_MISSING_COMMAND:-}" ]]; then
    return 1
  fi

  builtin command "$@"
}

check_environment
`

// TestBrowseCheckEnvironment covers each prerequisite independently. The command lookup seam keeps
// a deliberately missing tool local to the subprocess instead of changing the test process PATH.
//
// [Ja] TestBrowseCheckEnvironment は各前提条件を個別に確認する。コマンド検索の差し込み口に
// よって、意図的に欠けさせるツールをテストプロセスの PATH を変えず subprocess 内に閉じ込める。
func TestBrowseCheckEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		appEnv         string
		baseURL        string
		turnstile      string
		missingCommand string
		wantError      string
	}{
		{
			name:      "必要な環境が揃っている",
			appEnv:    "dev",
			baseURL:   "https://user:pass@example.test",
			turnstile: "false",
		},
		{
			name:      "APP_ENVがdevでない",
			appEnv:    "production",
			baseURL:   "https://user:pass@example.test",
			turnstile: "false",
			wantError: "APP_ENV must be dev",
		},
		{
			name:      "ベースURLがない",
			appEnv:    "dev",
			turnstile: "false",
			wantError: "KORYLUS_BROWSING_BASE_URL is not set",
		},
		{
			name:      "Turnstileが無効でない",
			appEnv:    "dev",
			baseURL:   "https://user:pass@example.test",
			turnstile: "true",
			wantError: "WIKINO_TURNSTILE_ENABLED must be false",
		},
		{
			name:           "nodeがない",
			appEnv:         "dev",
			baseURL:        "https://user:pass@example.test",
			turnstile:      "false",
			missingCommand: "node",
			wantError:      "node is not available",
		},
		{
			name:           "curlがない",
			appEnv:         "dev",
			baseURL:        "https://user:pass@example.test",
			turnstile:      "false",
			missingCommand: "curl",
			wantError:      "curl is not available",
		},
		{
			name:           "playwright-cliがない",
			appEnv:         "dev",
			baseURL:        "https://user:pass@example.test",
			turnstile:      "false",
			missingCommand: "playwright-cli",
			wantError:      "playwright-cli is not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			command := browseCheckCommand(t, browseEnvironmentHarness)
			command.Env = browseTestEnvironment(
				"APP_ENV="+tt.appEnv,
				"KORYLUS_BROWSING_BASE_URL="+tt.baseURL,
				"WIKINO_TURNSTILE_ENABLED="+tt.turnstile,
				"WIKINO_BROWSE_TEST_MISSING_COMMAND="+tt.missingCommand,
			)

			var stderr bytes.Buffer
			command.Stderr = &stderr
			err := command.Run()
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("環境検査が成功することを期待したが %v: %s", err, stderr.String())
				}
				if stderr.Len() != 0 {
					t.Errorf("標準エラー出力が空であることを期待したが %q だった", stderr.String())
				}
				return
			}

			if err == nil {
				t.Fatalf("環境検査が失敗することを期待したが成功した")
			}
			if !strings.Contains(stderr.String(), tt.wantError) {
				t.Errorf("標準エラー出力に %q を期待したが %q だった", tt.wantError, stderr.String())
			}
		})
	}
}

const browseCredentialsHarness = `
set -euo pipefail

source "$WIKINO_BROWSE_TEST_SCRIPT"

go() {
  if [[ "$#" -eq 4 && "$1" == "run" && "$2" == "./cmd/wikino" && "$3" == "devcreds" ]]; then
    printf '%s' "$4" > "$WIKINO_BROWSE_TEST_CAPTURE/role"
    if [ "$WIKINO_BROWSE_TEST_CREDS_FAIL" = "1" ]; then
      return 1
    fi
    return 0
  fi

  return 64
}

check_credentials "${1:-}"
`

// TestBrowseCheckCredentials pins the default and translated roles and reports a roster miss.
//
// [Ja] TestBrowseCheckCredentials は既定 role と変換後の role を固定し、名簿に無い場合を報告する。
func TestBrowseCheckCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested string
		wantRole  string
		fails     bool
	}{
		{name: "指定なしはowner", wantRole: "owner"},
		{name: "従来の2はcollaborator", requested: "2", wantRole: "collaborator"},
		{name: "任意のrole", requested: "guest", wantRole: "guest"},
		{name: "名簿にroleがない", requested: "missing", wantRole: "missing", fails: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			captureDir := t.TempDir()
			command := browseCheckCommand(t, browseCredentialsHarness, tt.requested)
			command.Env = browseTestEnvironment(
				"WIKINO_BROWSE_TEST_CAPTURE="+captureDir,
				"WIKINO_BROWSE_TEST_CREDS_FAIL="+boolFlag(tt.fails),
			)

			var stderr bytes.Buffer
			command.Stderr = &stderr
			err := command.Run()
			if tt.fails && err == nil {
				t.Fatal("資格情報が無いときは失敗することを期待したが成功した")
			}
			if !tt.fails && err != nil {
				t.Fatalf("資格情報検査が成功することを期待したが %v: %s", err, stderr.String())
			}
			if tt.fails && !strings.Contains(stderr.String(), "no credentials for role 'missing'") {
				t.Errorf("名簿に無いroleの診断を期待したが %q だった", stderr.String())
			}

			role, readErr := os.ReadFile(filepath.Join(captureDir, "role"))
			if readErr != nil {
				t.Fatalf("検索したroleを読み込めなかった: %v", readErr)
			}
			if got := string(role); got != tt.wantRole {
				t.Errorf("資格情報をrole %q で検索することを期待したが %q だった", tt.wantRole, got)
			}
		})
	}
}

const browseReachableHarness = `
set -euo pipefail

source "$WIKINO_BROWSE_TEST_SCRIPT"
mkdir -p "$TMP_DIR"
printf 'stale credentials' > "$CURL_CONFIG_FILE"
chmod 600 "$CURL_CONFIG_FILE"
printf '%s' "$CURL_CONFIG_FILE" > "$WIKINO_BROWSE_TEST_CAPTURE/curl-config-path"
check_reachable
`

type browseRequestObservation struct {
	path       string
	username   string
	password   string
	basicAuth  bool
	configMode os.FileMode
	configErr  error
}

// TestBrowseCheckReachable runs the real curl against a local HTTP server. This verifies curl's own
// config parser receives quoted credentials intact, rather than teaching a stub how that parser is
// expected to behave.
//
// [Ja] TestBrowseCheckReachable は実際の curl をローカル HTTP サーバへ接続する。curl の設定
// parser を模倣した stub に期待動作を教えるのではなく、引用符を含む資格情報が parser 自身を
// そのまま通ることを確認する。
func TestBrowseCheckReachable(t *testing.T) {
	specialUser := "user name\"$\\path"
	specialPassword := "p@ss word \"$\\quoted:tail"

	tests := []struct {
		name              string
		status            int
		csrfCookie        bool
		username          string
		password          string
		connectionFailure bool
		wantError         string
		wantRequest       bool
	}{
		{
			name:        "200とCSRF_Cookie",
			status:      http.StatusOK,
			csrfCookie:  true,
			username:    specialUser,
			password:    specialPassword,
			wantRequest: true,
		},
		{
			name:        "Basic認証が401",
			status:      http.StatusUnauthorized,
			username:    "user",
			password:    "wrong",
			wantError:   "dev URL rejected the Basic-auth credentials",
			wantRequest: true,
		},
		{
			name:        "CSRF_Cookieがない",
			status:      http.StatusOK,
			username:    "user",
			password:    "pass",
			wantError:   "without the Go version's CSRF cookie",
			wantRequest: true,
		},
		{
			name:              "接続できない",
			username:          "user",
			password:          "pass",
			connectionFailure: true,
			wantError:         "dev URL did not respond",
		},
		{
			name:       "資格情報に制御文字がある",
			status:     http.StatusOK,
			csrfCookie: true,
			username:   "user",
			password:   "line\nbreak",
			wantError:  "credentials must not include control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureDir := t.TempDir()
			observations := make(chan browseRequestObservation, 1)

			var baseURL string
			var server *httptest.Server
			if tt.connectionFailure {
				baseURL = unreachableBrowseURL(t, tt.username, tt.password)
			} else {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					configPathBody, readErr := os.ReadFile(filepath.Join(captureDir, "curl-config-path"))
					var mode os.FileMode
					var statErr error
					if readErr == nil {
						var info os.FileInfo
						info, statErr = os.Stat(string(configPathBody))
						if statErr == nil {
							mode = info.Mode().Perm()
						}
					} else {
						statErr = readErr
					}

					username, password, ok := r.BasicAuth()
					observations <- browseRequestObservation{
						path:       r.URL.Path,
						username:   username,
						password:   password,
						basicAuth:  ok,
						configMode: mode,
						configErr:  statErr,
					}
					if tt.csrfCookie {
						http.SetCookie(w, &http.Cookie{Name: "wikino_csrf_token", Value: "test-token"})
					}
					w.WriteHeader(tt.status)
				}))
				defer server.Close()
				baseURL = browseURLWithCredentials(t, server.URL, tt.username, tt.password)
			}

			command := browseCheckCommand(t, browseReachableHarness)
			command.Env = browseTestEnvironment(
				"KORYLUS_BROWSING_BASE_URL="+baseURL,
				"WIKINO_BROWSE_TEST_CAPTURE="+captureDir,
				"WIKINO_BROWSE_TMP_DIR="+filepath.Join(captureDir, "tmp"),
			)
			var stderr bytes.Buffer
			command.Stderr = &stderr
			err := command.Run()
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("到達確認が成功することを期待したが %v: %s", err, stderr.String())
				}
			} else {
				if err == nil {
					t.Fatalf("到達確認が失敗することを期待したが成功した")
				}
				if !strings.Contains(stderr.String(), tt.wantError) {
					t.Errorf("標準エラー出力に %q を期待したが %q だった", tt.wantError, stderr.String())
				}
			}

			if tt.wantRequest {
				select {
				case observation := <-observations:
					if observation.path != "/sign_in" {
						t.Errorf("/sign_inへのリクエストを期待したが %q だった", observation.path)
					}
					if !observation.basicAuth {
						t.Error("Basic認証ヘッダーがない")
					}
					if observation.username != tt.username || observation.password != tt.password {
						t.Errorf(
							"Basic認証資格情報が変化した: got (%q, %q), want (%q, %q)",
							observation.username,
							observation.password,
							tt.username,
							tt.password,
						)
					}
					if observation.configErr != nil {
						t.Errorf("curl設定ファイルをリクエスト中に確認できなかった: %v", observation.configErr)
					}
					if observation.configMode != 0o600 {
						t.Errorf("curl設定ファイルの権限が600であることを期待したが %o だった", observation.configMode)
					}
				default:
					t.Fatal("dev URLへのリクエストが観測されなかった")
				}
			} else {
				select {
				case observation := <-observations:
					t.Errorf("dev URLへリクエストしないことを期待したが %+v を観測した", observation)
				default:
				}
			}

			configPathBody, readErr := os.ReadFile(filepath.Join(captureDir, "curl-config-path"))
			if readErr != nil {
				t.Fatalf("curl設定ファイルのパスを読み込めなかった: %v", readErr)
			}
			if _, statErr := os.Stat(string(configPathBody)); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("成功・失敗後にcurl設定ファイルが削除されることを期待したが err=%v", statErr)
			}
		})
	}
}

const browseCheckMainHarness = `
set -euo pipefail

source "$WIKINO_BROWSE_TEST_SCRIPT"

check_environment() {
  printf 'environment\n'
}
check_credentials() {
  printf 'credentials:%s\n' "${1:-}"
  if [ "$WIKINO_BROWSE_TEST_CREDS_FAIL" = "1" ]; then
    return 1
  fi
}
check_reachable() {
  printf 'reachable\n'
}

main "$@"
`

// TestBrowseMainDispatchesCheck verifies dispatch, role reporting, and failure status without
// repeating the checks that each function's focused test already covers.
//
// [Ja] TestBrowseMainDispatchesCheck は、各関数の個別テストを繰り返さず、振り分け・role の
// 報告・失敗時の終了ステータスを確認する。
func TestBrowseMainDispatchesCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		role      string
		fails     bool
		wantLines []string
		notWant   string
	}{
		{
			name:      "既定roleで成功",
			wantLines: []string{"environment", "credentials:", "reachable", "browser environment ready for owner"},
		},
		{
			name:      "roleを変換して成功",
			role:      "2",
			wantLines: []string{"environment", "credentials:2", "reachable", "browser environment ready for collaborator"},
		},
		{
			name:      "資格情報検査の失敗を返す",
			role:      "missing",
			fails:     true,
			wantLines: []string{"environment", "credentials:missing"},
			notWant:   "reachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := []string{"check"}
			if tt.role != "" {
				args = append(args, tt.role)
			}
			command := browseCheckCommand(t, browseCheckMainHarness, args...)
			command.Env = browseTestEnvironment("WIKINO_BROWSE_TEST_CREDS_FAIL=" + boolFlag(tt.fails))
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			if tt.fails && err == nil {
				t.Fatal("check内の検査が失敗したときは非ゼロ終了を期待したが成功した")
			}
			if !tt.fails && err != nil {
				t.Fatalf("checkが成功することを期待したが %v: %s", err, stderr.String())
			}
			for _, want := range tt.wantLines {
				if !strings.Contains(stdout.String(), want+"\n") {
					t.Errorf("標準出力に行 %q を期待したが %q だった", want, stdout.String())
				}
			}
			if tt.notWant != "" && strings.Contains(stdout.String(), tt.notWant) {
				t.Errorf("標準出力に %q を期待しなかったが %q だった", tt.notWant, stdout.String())
			}
		})
	}
}

const browseCloseHarness = `
set -euo pipefail

source "$WIKINO_BROWSE_TEST_SCRIPT"
pw() { :; }
mkdir -p "$TMP_DIR"
printf 'credentials' > "$CURL_CONFIG_FILE"
printf '%s' "$CURL_CONFIG_FILE" > "$WIKINO_BROWSE_TEST_CAPTURE/curl-config-path"
cmd_close
`

func TestBrowseCloseRemovesCurlConfig(t *testing.T) {
	t.Parallel()

	captureDir := t.TempDir()
	command := browseCheckCommand(t, browseCloseHarness)
	command.Env = browseTestEnvironment(
		"WIKINO_BROWSE_TEST_CAPTURE="+captureDir,
		"WIKINO_BROWSE_TMP_DIR="+filepath.Join(captureDir, "tmp"),
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("closeが成功することを期待したが %v: %s", err, output)
	}

	configPathBody, err := os.ReadFile(filepath.Join(captureDir, "curl-config-path"))
	if err != nil {
		t.Fatalf("curl設定ファイルのパスを読み込めなかった: %v", err)
	}
	if _, err := os.Stat(string(configPathBody)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("close後にcurl設定ファイルが削除されることを期待したが err=%v", err)
	}
}

func TestBrowseScriptIsExecutable(t *testing.T) {
	t.Parallel()

	info, err := os.Stat(browseCheckScriptPath(t))
	if err != nil {
		t.Fatalf("browse.shをstatできなかった: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("browse.shが直接実行可能であることを期待したが mode=%o だった", info.Mode().Perm())
	}
}

func browseCheckCommand(t *testing.T, harness string, args ...string) *exec.Cmd {
	t.Helper()

	commandArgs := []string{"-c", harness, "browse-check-test"}
	commandArgs = append(commandArgs, args...)
	command := exec.CommandContext(t.Context(), "bash", commandArgs...)
	command.Env = browseTestEnvironment()

	return command
}

func browseCheckScriptPath(t *testing.T) string {
	t.Helper()

	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "browse.sh"))
	if err != nil {
		t.Fatalf("browse.shの絶対パスを取得できなかった: %v", err)
	}

	return scriptPath
}

func browseTestEnvironment(overrides ...string) []string {
	overridden := make(map[string]struct{}, len(overrides)+1)
	overridden["WIKINO_BROWSE_TEST_SCRIPT"] = struct{}{}
	for _, override := range overrides {
		key, _, _ := strings.Cut(override, "=")
		overridden[key] = struct{}{}
	}

	environment := make([]string, 0, len(os.Environ())+len(overrides)+1)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, found := overridden[key]; !found {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "WIKINO_BROWSE_TEST_SCRIPT="+browseScriptPathForEnvironment())
	environment = append(environment, overrides...)

	return environment
}

func browseScriptPathForEnvironment() string {
	scriptPath, err := filepath.Abs(filepath.Join("..", "..", "scripts", "browse.sh"))
	if err != nil {
		panic(err)
	}

	return scriptPath
}

func browseURLWithCredentials(t *testing.T, rawURL string, username string, password string) string {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("テスト用URLを解析できなかった: %v", err)
	}
	parsed.User = url.UserPassword(username, password)

	return parsed.String()
}

func unreachableBrowseURL(t *testing.T, username string, password string) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("到達不能URL用のportを確保できなかった: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("到達不能URL用のlistenerを閉じられなかった: %v", err)
	}

	return browseURLWithCredentials(t, "http://"+address, username, password)
}
