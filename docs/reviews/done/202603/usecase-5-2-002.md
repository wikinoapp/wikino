# コードレビュー: usecase-5-2

## レビュー情報

| 項目                       | 内容                                                   |
| -------------------------- | ------------------------------------------------------ |
| レビュー日                 | 2026-03-31                                             |
| 対象ブランチ               | usecase-5-2                                            |
| ベースブランチ             | usecase-5-1                                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md   |
| 変更ファイル数             | 4 ファイル                                             |
| 変更行数（実装）           | +507 / -366 行（ドキュメント変更のみ、コード変更なし） |
| 変更行数（テスト）         | +0 / -0 行                                             |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/usecase-guide.md](/workspace/go/docs/usecase-guide.md) - ユースケースガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### ドキュメント

- [x] `go/docs/handler-guide.md`
- [ ] `go/docs/validation-guide.md`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`（タスク 5-2 のチェックボックス更新のみ）
- [x] `docs/reviews/usecase-5-2-001.md`（前回のレビュードキュメント追加）

## ファイルごとのレビュー結果

### `go/docs/validation-guide.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 依存関係ルール
- [@go/docs/usecase-guide.md](/workspace/go/docs/usecase-guide.md) - UseCase の設計パターン
- 作業計画書: 検討事項 5（Validator パッケージの位置づけ）

**問題点・改善提案**:

- **ファイル構成の例から `password_reset` ハンドラーのディレクトリが削除されている**: 変更前は `sign_in` と `password_reset` の 2 つのハンドラーが例示されていたが、変更後は `sign_in` のみになっている。`password_reset` は `usecase` に置き換わっており、ファイル構成の全体像としては問題ないが、Handler 側の例が 1 つだけになっているのは意図的か確認したい。

  **変更前**:

  ```
  internal/handler/sign_in/
  ├── handler.go
  ├── new.go
  └── create.go

  internal/handler/password_reset/
  ├── handler.go
  ├── new.go
  └── create.go
  ```

  **変更後**:

  ```
  internal/usecase/
  ├── create_sign_in.go
  └── create_sign_in_test.go

  internal/handler/sign_in/
  ├── handler.go
  ├── new.go
  └── create.go
  ```

  **対応方針**:
  - [x] 意図的な変更である（UseCase を加えたので Handler の例は 1 つで十分）
  - [ ] Handler の例として `password_reset` も残す
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書のタスク 5-2 の要件:

> - Handler の責務変更（オーケストレーター → 薄い Adapter）を反映
> - Validator の呼び出し元変更（Handler → UseCase）と Result 型廃止を反映
> - エラー型の使い分け（ValidationError, AppError, 素の error）を追記

**チェック結果**:

- [x] **handler-guide.md**: Handler の責務を「薄い Adapter」に変更済み。基本方針の冒頭に明記
- [x] **handler-guide.md**: Handler から Policy・Validator への直接依存禁止を明記
- [x] **handler-guide.md**: エラーハンドリングセクションを新設し、`errors.As` パターンを記載
- [x] **handler-guide.md**: DI 構築の順序（Validator → UseCase → Handler）を記載
- [x] **handler-guide.md**: コード例を新パターン（UseCase 経由）に更新
- [x] **handler-guide.md**: まとめセクションを新パターンに更新
- [x] **validation-guide.md**: Validator の呼び出し元を Handler → UseCase に変更
- [x] **validation-guide.md**: Result 型を廃止し `(data, error)` の 2 値返しに変更
- [x] **validation-guide.md**: エラー型の使い分け（ValidationError, AppError, 素の error）を追記
- [x] **validation-guide.md**: 「ハンドラーでの使用」セクションを「UseCase での使用」に変更
- [x] **validation-guide.md**: DI 構築のコード例を新パターンに更新
- [x] **validation-guide.md**: テストコードを新パターン（`error` 返し、`model.AsValidationError`）に更新
- [x] **validation-guide.md**: ベストプラクティスのコード例を新パターンに更新

**architecture-guide.md との整合性**:

- [x] handler-guide.md の「Handler は薄い Adapter」が architecture-guide.md の記述と一致
- [x] validation-guide.md の「UseCase から呼び出される」が architecture-guide.md の記述と一致
- [x] エラー型（ValidationError, AppError）の定義と使い分けが architecture-guide.md と一致

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 5-2 の要件通り、handler-guide.md と validation-guide.md が新しいアーキテクチャパターン（UseCase をオーケストレーターとする設計）に更新されている。

**良かった点**:

- handler-guide.md に「エラーハンドリング」セクションを新設し、3 種類のエラー型（ValidationError, AppError, 素の error）の使い分けを明確に記載している
- コード例が具体的で、実際のプロジェクトのコード（suggestion_comment, sign_in 等）に基づいている
- validation-guide.md の Result 型廃止と `(data, error)` 返しへの移行が一貫して反映されている
- DI 構築の順序（Validator → UseCase → Handler）が handler-guide.md と validation-guide.md の両方で一貫している
- architecture-guide.md（5-1 で更新済み）との整合性が取れている

**確認事項**:

- validation-guide.md のファイル構成例から `password_reset` ハンドラーが削除されている点について、意図的な変更かの確認が 1 件
