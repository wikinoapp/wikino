---
paths:
  - "**"
---

# korylus-guidelines 編集ガイドライン

本ドキュメントは `korylus-guidelines` (`.apm/instructions/` 配下) のガイドラインを編集するときに守りたい原則をまとめたものです。Korylus 全プロダクト (Annict, Mewst, Wikino) で共通利用されるため、一貫性と保守性を維持するためにここで定めた方針に従ってください。

APM の基本的な使い方やフォーマット運用は [apm.instructions.md](apm.instructions.md) を参照してください。

## プロダクト固有 vs 共通の振り分け

判断基準: **「Annict・Mewst・Wikino のいずれにも当てはまるか?」**。1 つでも当てはまらないなら共通には書かない。

| 内容                                                                 | 配置先                                       |
| -------------------------------------------------------------------- | -------------------------------------------- |
| 全プロダクトに適用される原則・パターン                               | `korylus-guidelines` の該当ファイル          |
| 全プロダクトに適用されるが、具体的なファイルパス・実装詳細を伴うもの | 概念だけ共通に書き、固有のパス・実装名は除去 |
| 一部プロダクトのみに適用される規約 (例: スペース ID クエリスコープ)  | 該当プロダクトの `CLAUDE.md`                 |
| プロダクト固有の設定値 (例: `WIKINO_` 環境変数プレフィックス、DB 名) | 該当プロダクトの `CLAUDE.md`                 |

例外: 現状は全プロダクト共通だがいずれ個別に状況が変わる事情 (例: Rails-to-Go 移行) は共通に残しつつ概念レベルに汎用化する。具体的な実装ファイル名 (例: `internal/middleware/reverse_proxy.go`) や具体的なパス (例: `/workspace/rails/`) は書かない。

## プロダクト名直書きを避ける

- 共通ガイドラインの本文 (見出し・説明文・本文) に **「Wikino」「Annict」「Mewst」を直書きしない**
- 「Go 版 Wikino」のような呼称は「Go 版プロジェクト」へ書き換える
- 特定プロダクトの `CLAUDE.md` への直接リンクや、プロダクト固有のディレクトリパス (`/workspace/rails/` など) も書かない

## コード例の表記

コード例では各プロダクトの実モデル名 (Wikino の `Topic` / `Space`, Annict の `Work` / `Episode` など) を使ってよい。中立化された架空のドメイン名にするより、実コードベースで参照可能な実モデル名を使うほうが運用上自然 (各プロダクトの対応中に随時 korylus-guidelines をアップデートする運用に合致する)。

ただし、**どのプロダクトの例かを必ず明示する**。

### コードブロックの場合: 先頭にコメントで明示

````markdown
```go
// Wikino の例
type Topic struct {
    ID    TopicID
    Space *Space
}
```
````

SQL なら `-- Wikino の例`、その他言語なら該当言語のコメント記法を使う。

### Markdown のテーブル・ファイルツリーの場合: キャプションで明示

```markdown
**Wikino の例**:

| Model  | Repository       |
| ------ | ---------------- |
| `Page` | `PageRepository` |
```

### 単発の inline 言及 (`code` 1 個など)

文中に `Page`, `SpaceID` のような 1 単語の言及を 1 度入れる程度であれば、注釈は不要。あくまで「**コードブロック単位**」「**テーブル/ファイルツリー単位**」で注釈を付ける。

## import パスの書き方

Go コード例の import パスは **常に `example.com/app/...` を使う** (Go 公式の予約済み example ドメイン)。

```go
// ✅ Good
import "example.com/app/internal/middleware"

// ❌ Bad
import "github.com/wikinoapp/wikino/internal/middleware"
import "github.com/kiraka/annict/internal/middleware"
```

理由: 特定プロダクトの実 import パスを残すと、他プロダクトで `apm install` した際に「自分のプロダクトには無い package path」が見えて違和感が出る。`example.com/app/...` は中立で、Go 公式が予約しているドメインなので衝突もしない。

## 既存の Annict / Mewst 例で注釈が無いものの扱い

過去の経緯で Annict や Mewst のドメイン例 (例: `Work`, `Episode`, `Cast`, `Staff`) が共通ガイドラインに混入している箇所がある。これらは順次「該当プロダクトに対応している作業」のついでに `// Annict の例` などの注釈を付与していけばよい。本ガイドライン制定時点で一括対応はしない (PR サイズが大きくなりすぎるため)。

新規にコード例を追加する場合は、本ドキュメントの規則に沿って **必ず注釈を付ける**。

## チェックリスト

`korylus-guidelines` の編集をコミットする前に以下を確認:

- [ ] 本文 (見出し・説明文) に「Wikino」「Annict」「Mewst」が直書きされていない
- [ ] プロダクト固有の規約・設定値が共通に紛れ込んでいない
- [ ] 新規コードブロックには `// {Product} の例` の注釈が付いている
- [ ] 新規テーブル/ファイルツリーには `**{Product} の例**:` キャプションが付いている
- [ ] Go import パスが `example.com/app/...` になっている
- [ ] `pnpm fmt:check` が通る
