# コードレビュー: handler-usecase-refactor-2-3

## レビュー情報

| 項目                       | 内容                                                                                                        |
| -------------------------- | ----------------------------------------------------------------------------------------------------------- |
| レビュー日                 | 2026-03-12                                                                                                  |
| 対象ブランチ               | handler-usecase-refactor-2-3                                                                                |
| ベースブランチ             | handler-usecase-refactor-2-2                                                                                |
| 作業計画書（指定があれば） | [docs/plans/1_doing/handler-usecase-refactor.md](/workspace/docs/plans/1_doing/handler-usecase-refactor.md) |
| 変更ファイル数             | 14 ファイル                                                                                                 |
| 変更行数（実装）           | +27 / -8 行（ハンドラー・main.go・golangci.yml の差分）                                                     |
| 変更行数（テスト）         | +26 / -8 行（ハンドラーテストの差分）                                                                       |

※ `validator/page.go`, `validator/page_move.go` 及び対応テストファイルはリネーム+リファクタリングのため、実質的な変更行数は上記より大きく見えるが、ロジックの変更は命名規則の変更のみ。

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/validator/page.go`（新規: `handler/page/validator.go` から移動）
- [x] `go/internal/validator/page_move.go`（新規: `handler/page_move/validator.go` から移動）
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/handler/page_move/handler.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/validator/page_test.go`（新規: `handler/page/validator_test.go` から移動）
- [x] `go/internal/validator/page_move_test.go`（新規: `handler/page_move/validator_test.go` から移動）
- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/handler/page_move/create_test.go`
- [x] `go/internal/handler/page_move/new_test.go`

### 設定・その他

- [x] `go/.golangci.yml`
- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

### `go/.golangci.yml`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 依存関係ルール
- [@go/docs/development-guide.md](/workspace/go/docs/development-guide.md) - golangci-lint

**問題点・改善提案**:

- **[作業計画書との整合性]**: Handler → Repository 禁止の depguard ルールが追加されているが、現在の handler パッケージ（`page/handler.go`, `page_move/handler.go` を含む計 19 ファイル）はまだ `internal/repository` を import している。この状態で `make lint` を実行するとすべての handler ファイルで depguard 違反が検出され、CI が失敗する。

  作業計画書のタスク 2-3 には「`make lint` で違反箇所がすべて検出されることを確認」と記載されているため、検出自体は意図通り。しかし、このルールを今追加すると **CI が通らなくなり、この PR 及びフェーズ 3〜6 の PR がマージできなくなる**可能性がある。

  **修正案**:

  depguard ルールの追加はフェーズ 6（すべての handler から repository import が除去された後）に延期するか、あるいは一時的にコメントアウトして handler のリファクタリングが完了してから有効化する。

  ```yaml
  # フェーズ6完了後に有効化する
  # - pkg: github.com/wikinoapp/wikino/go/internal/repository
  #   desc: "HandlerはRepositoryに直接依存できません。UseCaseを経由してください。"
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] depguard ルールの追加をフェーズ 6 完了後に延期する
  - [ ] ルールをコメントアウトした状態でこの PR に含める（意図の記録として）
  - [ ] CI 失敗を許容してこのまま進める（後続 PR で順次修正）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

Validator の `internal/validator/` パッケージへの移動は作業計画書通りに正確に実施されている。命名規則（`PageUpdateValidator`, `PageMoveCreateValidator`）は既存パターンと一貫性があり、Handler の依存性注入パターン・`main.go` での構築も適切。テストも正常系・異常系を網羅しており品質は高い。

唯一の問題点は、**depguard の Handler → Repository 禁止ルールの追加タイミング**。現時点で多数の handler ファイルがまだ repository を import しているため、このルールを有効化すると CI が失敗する。ルール追加の延期またはコメントアウトを推奨する。
