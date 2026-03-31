# コードレビュー: usecase-2-1

## レビュー情報

| 項目                       | 内容                                                              |
| -------------------------- | ----------------------------------------------------------------- |
| レビュー日                 | 2026-03-27                                                        |
| 対象ブランチ               | usecase-2-1                                                       |
| ベースブランチ             | usecase-1-3                                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（タスク2-1） |
| 変更ファイル数             | 13 ファイル（レビュー・計画書除く）                               |
| 変更行数（実装）           | +165 / -71 行                                                     |
| 変更行数（テスト）         | +281 / -28 行                                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ、UseCase
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約、ログ出力
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/usecase/create_suggestion_comment.go`
- [ ] `go/internal/validator/suggestion_comment.go`

### テストファイル

- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/usecase/create_suggestion_comment_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/validator/suggestion_comment.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **不要な `session` パッケージの import**: `SuggestionCommentCreateValidator` は `model.ValidationError` に移行済みだが、同ファイル内の `SuggestionCommentUpdateValidator` がまだ `session.FormErrors` を使用しているため `session` import が残っている。これ自体はタスク 3-3 で解消される想定だが、移行途中であることを認識した上で問題ないか確認したい。

  **対応方針**:
  - [x] タスク 3-3 で解消されるため、現状のまま進める
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書タスク 2-1 の要件と実装の対応:

| 要件                                                                             | 実装状況                                      |
| -------------------------------------------------------------------------------- | --------------------------------------------- |
| UseCase に Validator・Policy の呼び出しを統合                                    | ✅                                            |
| Validator の Result 型を廃止し `(data, error)` 返しに変更                        | ✅                                            |
| Handler から Validator・Policy の直接呼び出しを削除し `errors.As` パターンに変更 | ✅                                            |
| テンプレートの `session.FormErrors` 参照を `model.ValidationError` に変更        | ✅ (該当なし: フラッシュメッセージ経由のため) |
| `main.go` の DI 構成を更新                                                       | ✅                                            |
| 全テストを更新                                                                   | ✅                                            |

UseCase の処理順序が作業計画書「検討事項 4」の確定方針に一致:

```
1. データ取得（トランザクション外）
2. 認可チェック
3. バリデーション
4. ビジネスロジック（計算、変換等）
5. トランザクション（永続化のみ）
```

エラー型の使い分けが作業計画書「検討事項 1」の確定方針に一致:

| エラー型                 | 生成元    | Handler の対応                    | 実装 |
| ------------------------ | --------- | --------------------------------- | ---- |
| `*model.ValidationError` | Validator | フラッシュメッセージ+リダイレクト | ✅   |
| `*model.AppError`        | UseCase   | NotFound/Forbidden/500            | ✅   |
| 素の `error`             | どこでも  | ログ+500                          | ✅   |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 2-1（パイロット移行）の要件をすべて満たしている。UseCase がオーケストレーターとしてデータ取得→認可→バリデーション→ビジネスロジック→永続化の流れを統括し、Handler は HTTP の入出力変換に徹する構造に正しくリファクタリングされている。

良い点:

- UseCase の `Execute` メソッドが `fetchData` → `authorize` → `validate` → ビジネスロジック → `createComment` と明確に分離されており、可読性が高い
- Handler の `handleError` メソッドが `ValidationError` → `AppError` → 素の `error` の3パターンを正しく処理している
- UseCase テストが正常系（2ケース）・異常系（3ケース: NotFound, Forbidden, ValidationError）を網羅している
- Handler の依存性が `flashMgr` + `createSuggestionCommentUsecase` の2つに削減され、薄い Adapter としての設計が明確
- `CanCreateSuggestionComment` メソッドが全 Policy 実装（owner, admin, member, guest）に一貫して追加されている
