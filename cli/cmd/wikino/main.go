// wikinoコマンドはWikinoの公式のコマンドラインツール。
// Wikinoの公開API (/api/v1) を呼んで、ページや下書きを読み書きする。
//
// 同じ名前のサーバーのコマンド (go/cmd/wikino) とは別のモジュールで、
// `go install` でサーバーの依存を引き込まないよう分けている。
package main

import (
	"os"
	"runtime/debug"

	"github.com/wikinoapp/wikino/cli/internal/auth"
	"github.com/wikinoapp/wikino/cli/internal/cmd/root"
	"github.com/wikinoapp/wikino/cli/internal/cmdutil"
	"github.com/wikinoapp/wikino/cli/internal/i18n"
	"github.com/wikinoapp/wikino/cli/internal/iostreams"
	"github.com/wikinoapp/wikino/cli/internal/output"
)

// versionはリリースのビルドで `-ldflags "-X main.version=..."` により埋め込む。
var version string

func main() {
	info, ok := debug.ReadBuildInfo()

	os.Exit(run(os.Args[1:], resolveVersion(version, info, ok), iostreams.System(os.Environ()), os.Getenv))
}

// runはargsでコマンドを実行し、プロセスの終了コードを返す。
// 引数・入出力・環境変数を受け取るのは、テストプロセスを終了させずに実行を確かめるため。
func run(args []string, version string, ios *iostreams.IOStreams, getenv func(string) string) int {
	f := &cmdutil.Factory{
		IOStreams: ios,
		Localizer: i18n.NewLocalizer(i18n.DetectLang(getenv)),
		Version:   version,
		Getenv:    getenv,
		Keyring:   auth.SystemKeyring(getenv),
		// ブラウザの起動はOSに依存するため、テストでは差し替える。
		OpenBrowser: auth.OpenBrowser,
	}

	jsonRequested, err := root.Execute(args, f)
	if err == nil {
		return output.ExitOK
	}

	// エラーは標準エラーを読む側に合わせ、標準エラーがTTYでないか `--json` のときにJSONで出す。
	// 書き込みエラーは意図的に捨てる。ここは診断情報の出力先そのものであり、
	// その書き込みに失敗したことを報告する先が残っていないため。
	_ = output.WriteError(ios.ErrOut, err, jsonRequested || !ios.IsStderrTTY())

	return output.ExitCodeOf(err)
}

// resolveVersionは `wikino --version` で出すバージョンを決める。
// `go install ...@vX.Y.Z` ではldflagsを渡せないため、モジュールのバージョンで補う。
// どちらも無い手元のビルド (`(devel)`) では "dev" とする。
func resolveVersion(ldflagsVersion string, info *debug.BuildInfo, ok bool) string {
	if ldflagsVersion != "" {
		return ldflagsVersion
	}

	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return "dev"
}
