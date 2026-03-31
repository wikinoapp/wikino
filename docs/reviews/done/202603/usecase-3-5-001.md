# コードレビュー: usecase-3-5

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-5                                          |
| ベースブランチ             | usecase-3-4                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 13 ファイル                                          |
| 変更行数（実装）           | +166 / -151 行                                       |
| 変更行数（テスト）         | +225 / -116 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/page_move/handler.go`
- [x] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/internal/usecase/get_page_move_data.go`
- [x] `go/internal/usecase/move_page.go`
- [x] `go/internal/validator/page_move.go`

### テストファイル

- [x] `go/internal/handler/page_move/new_test.go`
- [x] `go/internal/handler/page_move/create_test.go`
- [x] `go/internal/usecase/get_page_move_data_test.go`
- [x] `go/internal/usecase/move_page_test.go`
- [x] `go/internal/validator/page_move_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。全ファイルがガイドラインに従っています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 3-5「move_page UseCase の移行」が設計通りに実装されている。主要な変更点と評価は以下の通り。

**設計との整合性**: ✅ 完全に一致

- UseCase がオーケストレーターとして、データ取得 → 認可 → バリデーション → 永続化の流れを統括
- Handler は HTTP の入出力変換に徹する（`create.go` の `handleCreateError` パターン）
- Validator は `(*model.Topic, error)` の2値返しに変更（Result 構造体の廃止）
- `session.FormErrors` への依存を `model.ValidationError` に置き換え

**既存パターンとの一貫性**: ✅ 良好

- `handleCreateError` のエラー分岐（ValidationError → AppError → 素の error）は `suggestion/create.go`、`page/update.go` と同一パターン
- `fetchPageAccessData` + `authorizePageUpdate` の共有ヘルパー利用は `publish_page.go` と一致
- `pageAccessRepos()` メソッドによるリポジトリ群の集約も同一パターン

**アーキテクチャ**: ✅ 適切

- Handler から `validator`・`policy` パッケージへの import が削除され、depguard ルール変更の方針に沿っている
- UseCase → Validator → Policy の依存方向が作業計画書の設計に一致
- トランザクションは `movePage` メソッドに分離され、Execute 内では永続化前のロジックのみ実行

**テストカバレッジ**: ✅ 十分

- Handler テスト: 成功系、バリデーションエラー系が実装
- UseCase テスト: 正常系、スペース不存在（AppError）、権限なし（AppError）、バリデーションエラー（ValidationError）を網羅
- Validator テスト: 空入力、不正値、同一トピック、タイトル重複、正常系を網羅
- テストが `SetupTx` → `GetTestDB` に適切に変更されている（UseCase が自前でトランザクション管理するため）

**セキュリティ**: ✅ 問題なし

- 認可チェックは UseCase 内で必ず実行される
- スペース ID によるクエリスコープが維持されている
