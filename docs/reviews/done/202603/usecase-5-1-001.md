# コードレビュー: usecase-5-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-31                                           |
| 対象ブランチ               | usecase-5-1                                          |
| ベースブランチ             | usecase-4-4                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 5 ファイル                                           |
| 変更行数（実装）           | +808 / -635 行（ドキュメントのみ、テストなし）       |
| 変更行数（テスト）         | +0 / -0 行                                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@CLAUDE.md#コメントのガイドライン](/workspace/CLAUDE.md) - コメントのガイドライン
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/usecase-guide.md](/workspace/go/docs/usecase-guide.md) - ユースケースガイド

## 変更ファイル一覧

### ドキュメント

- [x] `CLAUDE.md`
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [ ] `go/docs/usecase-guide.md`
- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/docs/usecase-guide.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md#コメントのガイドライン](/workspace/CLAUDE.md) - コメントのガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- 作業計画書との整合性

**問題点・改善提案**:

- **[作業計画書との整合性]**: 「ハンドラーでの使用」セクション（386-434行目）の実装例が旧パターン（Handler がオーケストレーターとして Validator を直接呼び出し、UseCase を直接生成する）のまま記述されている

  ```go
  // 現在の記述（386-434行目）: Handler が Validator を直接呼び出し、UseCase を new して実行する旧パターン
  func (h *Handler) ProcessPasswordReset(w http.ResponseWriter, r *http.Request) {
      // リクエストバリデーション
      req := &PasswordResetRequest{...}
      if formErrors := req.Validate(ctx); formErrors != nil { ... }

      // ユーザーを検索
      user, err := h.queries.GetUserByEmail(ctx, req.Email)

      // ユースケースを実行
      uc := usecase.NewCreatePasswordResetTokenUsecase(h.db, h.queries)
      result, err := uc.Execute(ctx, user.ID)

      // メール送信
      err = h.sendPasswordResetEmail(ctx, user.Email, result.Token)
  }
  ```

  このパターンでは Handler が `queries.GetUserByEmail` を直接呼び出し、`PasswordResetRequest.Validate` も Handler 内で実行しており、新しいアーキテクチャ（UseCase がオーケストレーター）と矛盾する。同ファイル内の「Handler の実装パターン」セクション（190-227行目）では新しい `errors.As` パターンが正しく記述されているため、整合性が取れていない。

  **修正案**:

  「ハンドラーでの使用」セクションの実装例を新パターンに更新する。具体的には、Handler は UseCase を呼び出すだけにし、バリデーション・データ取得・メール送信はすべて UseCase 内で行うパターンに変更する。または、このセクション自体が「Handler の実装パターン」セクションと重複するため、削除して「Handler の実装パターン」セクションへの参照に置き換える。

  **対応方針**:
  - [ ] 「ハンドラーでの使用」セクションの実装例を新パターンに更新する
  - [x] 「ハンドラーでの使用」セクションを削除し「Handler の実装パターン」セクションを参照する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書のタスク **5-1** の要件:

> - 3層アーキテクチャの図を更新（Worker を Presentation 層に、Dispatcher を Domain/Infrastructure 層に追加）
> - UseCase のオーケストレーション責務、処理順序を反映
> - 書き込み UseCase のルール（旧ルール1の廃止）を反映
> - 依存関係ルールの更新（depguard ルールとの整合性）

確認結果:

| 要件                                       | 対応状況                                            |
| ------------------------------------------ | --------------------------------------------------- |
| 3層アーキテクチャの図を更新                | ✅ 対応済み（architecture-guide.md, CLAUDE.md）     |
| UseCase のオーケストレーション責務を反映   | ✅ 対応済み（usecase-guide.md に分離・新設）        |
| UseCase 内の処理順序を反映                 | ✅ 対応済み（usecase-guide.md に記述）              |
| 書き込み UseCase のルール更新              | ✅ 対応済み（旧ルール1を廃止、2つのルールに変更）   |
| 依存関係ルールの更新                       | ✅ 対応済み（Handler→Policy/Validator禁止等を反映） |
| Worker の Presentation 層移動を反映        | ✅ 対応済み                                         |
| Dispatcher の Domain/Infrastructure 層追加 | ✅ 対応済み                                         |
| 採用しなかった方針の更新                   | ✅ 対応済み（D, E を追加）                          |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

architecture-guide.md から UseCase 関連の内容を `usecase-guide.md` に分離した構成は適切で、作業計画書の要件をほぼすべて満たしている。3層アーキテクチャの図・依存関係ルール・UseCase の処理順序・採用しなかった方針の追加も正確に反映されている。

1点、`usecase-guide.md` の「ハンドラーでの使用」セクションの実装例が旧パターンのまま残っている点のみ確認が必要。同ファイル内の「Handler の実装パターン」セクションでは正しい新パターンが記述されているため、読者の混乱を避けるために対応が望ましい。
