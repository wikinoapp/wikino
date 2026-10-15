# Wikino CLI

`wikino` は、[Wikino](https://wikino.app/) の公式のコマンドラインツールです。
Wikinoの公開APIを呼んで、スペースのページや下書きを読み書きします。
人間が端末で使うほか、コーディングエージェントから使うことを主な用途にしています。

## インストール

### GitHub Releasesから入れる

[Releases](https://github.com/wikinoapp/wikino/releases) から、`Wikino CLI` のリリースのうち使っているOSとアーキテクチャのアーカイブをダウンロードします。

| OS      | アーカイブ                                                      |
| ------- | --------------------------------------------------------------- |
| macOS   | `wikino_vX.Y.Z_darwin_arm64.tar.gz` / `..._darwin_amd64.tar.gz` |
| Linux   | `wikino_vX.Y.Z_linux_amd64.tar.gz` / `..._linux_arm64.tar.gz`   |
| Windows | `wikino_vX.Y.Z_windows_amd64.zip` / `..._windows_arm64.zip`     |

同じリリースの `wikino_vX.Y.Z_checksums.txt` で、ダウンロードしたアーカイブを確かめます。

```sh
# Linux
sha256sum --ignore-missing -c wikino_vX.Y.Z_checksums.txt

# macOS
shasum -a 256 --ignore-missing -c wikino_vX.Y.Z_checksums.txt
```

アーカイブを展開し、中の `wikino` (Windowsでは `wikino.exe`) をPATHの通ったディレクトリに置きます。

```sh
tar -xzf wikino_vX.Y.Z_linux_amd64.tar.gz
install wikino_vX.Y.Z_linux_amd64/wikino ~/.local/bin/
```

バイナリには署名をしていません。
macOSでブラウザからダウンロードしたときに実行を止められた場合は、`xattr -d com.apple.quarantine ~/.local/bin/wikino` で隔離の属性を外してください。

### `go install` で入れる

Goの開発環境があれば、`go install` でも入れられます。

```sh
go install github.com/wikinoapp/wikino/cli/cmd/wikino@latest
```

入れたら、バージョンが表示されることを確かめます。

```sh
wikino --version
```

## ログイン

使うスペースの識別子 (URLの `/s/{識別子}` の部分) を指定してログインします。
ブラウザでWikinoの認可画面が開くので、許可するとログインが終わります。

```sh
wikino auth login my-space
```

SSHで接続しているときなど、ブラウザを開けない環境ではデバイス認可に切り替わります。
表示されたURLを手元の端末のブラウザで開き、コードを入力して許可してください。
スペース設定で発行した個人アクセストークンでもログインできます。
方法ごとの詳しい手順とスコープの指定は `wikino auth login --help` を参照してください。

ログインした資格情報は `wikino auth status` で確かめられます。

## 使い方

ページの番号はスペースの中で一意で、ページのURLの `/pages/{番号}` の部分です。

```sh
# ページを検索する
wikino page search '議事録'

# ページ12を読む
wikino page get 12

# ページ12をWikinoファイル形式のMarkdownで書き出し、編集してから書き戻す
wikino page get 12 --md > page.md
wikino page update 12 --md-file page.md

# ページ12の下書きを作成する
wikino draft create 12 --body-file minutes.md
```

標準出力がTTYなら人間向けの表示を、パイプやファイルに出すときはJSONを出します。
コマンドの一覧は `wikino --help` で、各コマンドの引数・出力の項目・エラーは `wikino <コマンド> --help` で確かめられます。

ヘルプとエラーは日本語と英語で表示します。
表示言語は `WIKINO_LANG` → `LC_ALL` → `LC_MESSAGES` → `LANG` の順に決め、日本語以外は英語になります。

## コーディングエージェントで使う

このCLIには、コーディングエージェント向けのスキル (SKILL.mdと場面ごとの手順書) を同梱しています。
次のコマンドで、Claude Codeのユーザー単位のスキルのディレクトリ (`~/.claude/skills/wikino/`) に書き出します。

```sh
wikino skill install
```

ほかのエージェントで使うときは、`--dir` でそのエージェントがスキルを読むディレクトリを指定します。
CLIを更新したときは、もう一度実行してスキルも更新してください。

エージェントに渡すトークンは、任せる作業に要るスコープだけに絞ることを勧めます。
たとえば読み取りと下書きのスコープだけでログインすると、エージェントはページを直接書き換えられず、下書きの公開は人が行う運用にできます。

```sh
wikino auth login my-space --scopes page:read,draft_page:read,draft_page:write
```

## リリース

`cli/vX.Y.Z` の形のタグをpushすると、[CLI Release](../.github/workflows/cli-release.yml) のワークフローがバイナリとチェックサムをGitHub Releasesに出します。
バージョンにハイフンを含むタグ (`cli/v1.0.0-rc.1` など) はプレリリースになります。

```sh
git tag cli/v1.0.0
git push origin cli/v1.0.0
```

リポジトリには接頭辞の無い古いアプリのタグ (`v0.7.0` など) が残っているため、CLIのタグには必ず `cli/` を付けてください。
同じタグは `go install ...@vX.Y.Z` でも使われます。
