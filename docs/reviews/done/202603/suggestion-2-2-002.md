# コードレビュー: suggestion-2-2

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-16                       |
| 対象ブランチ               | suggestion-2-2                   |
| ベースブランチ             | develop                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 15 ファイル                      |
| 変更行数（実装）           | +486 / -25 行                    |
| 変更行数（テスト）         | +371 / -11 行                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [ ] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/index.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion/index.templ`
- [x] `go/internal/templates/pages/suggestion/index_templ.go` (自動生成)
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/get_suggestion_list.go`
- [x] `go/internal/viewmodel/suggestion.go` (差分なし・前PRで追加済み)

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion/main_test.go`
- [x] `go/internal/usecase/get_suggestion_list_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-2-2-001.md` (前回レビュー)

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/handler.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - Handler構造体の定義
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 依存関係

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#Handler構造体の定義]**: 既存の `topic/handler.go` では `flashMgr *session.FlashManager` を Handler のフィールドに含めているが、`suggestion/handler.go` には含まれていない。現在の Index ハンドラーではフラッシュメッセージを使用していないが、今後のフェーズ（編集提案の作成・反映・クローズなど）でフラッシュメッセージが必要になる可能性が高い。

  **修正案**:

  現時点ではフラッシュメッセージを使用しないため、YAGNI原則に従って不要。今後のフェーズで必要になった時点で追加する方針で問題ない。ただし、topic handler との一貫性を重視するかどうかの判断を確認したい。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] YAGNI原則に従い、現時点では追加しない（今後必要になったら追加）
  - [ ] 一貫性のため、今の段階で `flashMgr` を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 2-2（編集提案一覧のハンドラーとテンプレート）の実装として、作業計画書の仕様に沿って正しく実装されている。

**良かった点**:

- 既存の `topic/show.go` のパターン（UseCase呼び出し → ViewModel変換 → レイアウトデータ構築 → レンダリング）を忠実に踏襲しており、コードベースの一貫性が保たれている
- UseCaseにスペース・トピックの取得・権限チェックのロジックを移動し、ハンドラーをシンプルに保っている。トピック詳細画面のUseCaseと同様のパターン
- 非公開トピックの権限チェック（スペースオーナーまたはトピックメンバーのみ閲覧可能）が正しく実装されている
- オープン/クローズのタブ切り替え、件数表示、空状態の表示がGitHub風のUIパターンで自然に実装されている
- テストがハンドラーとUseCaseの両方をカバーしており、正常系（公開トピック閲覧・クローズタブ・スペースオーナー閲覧）と異常系（存在しないスペース・不正なトピック番号・非公開トピック未ログイン）を適切に網羅している
- i18nの翻訳キーが命名規則（`suggestion_index_*`）に従い、description も適切に記述されている
- アーキテクチャの依存関係ルール（Handler → UseCase → Repository）が正しく守られている

**軽微な確認事項**:

- `handler.go` の `flashMgr` の有無について、上記で確認を記載した。現時点では問題ないが方針を確認したい
