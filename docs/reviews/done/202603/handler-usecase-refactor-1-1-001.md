# コードレビュー: handler-usecase-refactor-1-1

## レビュー情報

| 項目                       | 内容                                                               |
| -------------------------- | ------------------------------------------------------------------ |
| レビュー日                 | 2026-03-12                                                         |
| 対象ブランチ               | handler-usecase-refactor-1-1                                       |
| ベースブランチ             | go-topic-fix-1                                                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md                     |
| 変更ファイル数             | 5 ファイル                                                         |
| 変更行数（実装）           | +238 / -218 行（すべてドキュメント変更のため実装コードの変更なし） |
| 変更行数（テスト）         | +0 / -0 行                                                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@CLAUDE.md#コメントのガイドライン](/workspace/CLAUDE.md) - コメントのガイドライン
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### ドキュメント

- [x] `go/CLAUDE.md`
- [ ] `go/docs/architecture-guide.md`
- [ ] `go/docs/handler-guide.md`
- [ ] `go/docs/validation-guide.md`
- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

### `go/docs/validation-guide.md`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md)
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md)

**問題点・改善提案**:

- **[コード例の不整合]**: 501 行目のテストコード例で、変数名が `validator` のままになっている。492 行目で変数名を `v` に変更しているが、「有効な認証情報」テストケースの Validate 呼び出しが更新されていない。同じファイル内の 518 行目「無効なパスワード」テストケースでは正しく `v.Validate` に更新されている。

  ```go
  // 501行目: 問題のあるコード
  result := validator.Validate(ctx, input)
  ```

  **修正案**:

  ```go
  // 修正後のコード
  result := v.Validate(ctx, input)
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `v.Validate` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[コメントの誤記]**: 370 行目のコード例で、コメントが `// external/validator パッケージ` になっているが、正しくは `// internal/validator パッケージ`。

  ```go
  // 370行目: 問題のあるコード
  validator       *validator.SignInCreateValidator  // external/validator パッケージ
  ```

  **修正案**:

  ```go
  // 修正後のコード
  validator       *validator.SignInCreateValidator  // internal/validator パッケージ
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `internal` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/docs/architecture-guide.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 自身の整合性
- 作業計画書: docs/plans/1_doing/handler-usecase-refactor.md

**問題点・改善提案**:

- **[Validator の層分類]**: Validator パッケージが「Validator（独立パッケージ）」として Application 層と Domain/Infrastructure 層の間に記述されているが、Validator は Repository に依存し Handler から呼び出されるため、論理的な層の位置づけが曖昧。`go/CLAUDE.md` の主要パッケージテーブルでは層が `-`（なし）になっている。architecture-guide.md 側でもセクションタイトルに `（独立パッケージ）` と付いているものの、3 層アーキテクチャの図には含まれていないため、読者にとって Validator がどの層に位置するのか判断しにくい可能性がある。

  **修正案**:

  「Validator（独立パッケージ）」セクションの冒頭に、Validator が 3 層のどこにも属さない横断的なパッケージであることを 1 行補足する（例: 「Validator は 3 層アーキテクチャのいずれの層にも属さない独立パッケージとして、Handler（Presentation 層）から呼び出され、Repository（Domain/Infrastructure 層）に依存する」）。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 修正案の通り補足を追加する
  - [ ] 現状のまま（`go/CLAUDE.md` の層 `-` と `（独立パッケージ）` の記述で十分）
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  Presentation層に属するのが良いかなと思ったのですが、どう思いますか？懸念点などあれば教えてください。
  懸念点などなければPresentation層に属する方向でドキュメントの更新をお願いします。
  ```

### `go/docs/handler-guide.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md)
- 作業計画書: docs/plans/1_doing/handler-usecase-refactor.md

**問題点・改善提案**:

- **[標準ファイル名の数え方]**: 「標準ファイル名（8 種類のみ）」としているが、リスト内のファイルは `handler.go`, `index.go`, `show.go`, `new.go`, `create.go`, `edit.go`, `update.go`, `delete.go` の 8 個。「まとめ」セクション（530 行目付近）でも「8 種類のファイル名のみを使用」と一致している。ただし、以前の「9 種類」は `validator.go` を含んでいたのに対し、現在は `validator.go` を削除して 8 種類としている。テストファイル（`handler_test.go` 等）はカウントに含めない方針が暗黙的だが、見出しの「のみ」が強い制約に見える。これは既存の方針との一貫性の問題なので、既存パターンを確認した上で判断してほしい。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 現状のまま（「8 種類のみ」で問題ない。テストファイルは暗黙的に許可されている）
  - [x] テストファイルが暗黙許可であることを明示する注釈を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書タスク **1-1** の要件との整合性を確認:

| 要件                                                                             | 状態 | 備考                                                     |
| -------------------------------------------------------------------------------- | ---- | -------------------------------------------------------- |
| `architecture-guide.md`: Handler → Repository の依存を禁止するルールを追加       | ✅   | 275 行目、320 行目で明確に記述                           |
| `architecture-guide.md`: UseCase の役割を「書き込み + 読み取り」に拡張           | ✅   | 538-600 行目で読み取り/書き込み UseCase を詳細記述       |
| `architecture-guide.md`: Validator パッケージの分離について記述を追加            | ✅   | 228-237 行目で独立パッケージとして記述                   |
| `architecture-guide.md`: 読み取り UseCase の設計パターンとコード例を追加         | ✅   | 576-600 行目にコード例あり                               |
| `CLAUDE.md`: 重要な設計原則セクションを更新                                      | ✅   | 3 つの新原則を追加                                       |
| `validation-guide.md`: Validator の配置先を `internal/validator/` に変更         | ✅   | 全体的にパッケージ・命名規則を更新                       |
| `handler-guide.md`: ハンドラーディレクトリから `validator.go` を削除する旨を更新 | ✅   | ディレクトリ構造、ファイル名規則、バリデーター配置を更新 |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

作業計画書タスク 1-1 の 7 つの要件がすべて反映されており、ドキュメント間の整合性も概ね取れている。4 つのドキュメントが一貫した方針（Handler → Repository 禁止、UseCase の読み取り/書き込み二分化、Validator の独立パッケージ化）で更新されている点は良い。

指摘事項は以下の通り:

- **必須修正 2 件**: `validation-guide.md` のコード例における変数名 `validator` → `v` の更新漏れ（501 行目）、および `external` → `internal` の誤記（370 行目）
- **確認事項 2 件**: architecture-guide.md での Validator の層位置づけの明確化、handler-guide.md での標準ファイル名カウントとテストファイルの関係
