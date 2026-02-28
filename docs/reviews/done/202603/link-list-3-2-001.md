# コードレビュー: link-list-3-2

## レビュー情報

| 項目                       | 内容                                      |
| -------------------------- | ----------------------------------------- |
| レビュー日                 | 2026-02-28                                |
| 対象ブランチ               | link-list-3-2                             |
| ベースブランチ             | page-edit                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/link-list-alignment.md |
| 変更ファイル数             | 13 ファイル                               |
| 変更行数（実装）           | +237 / -15 行（手書きコード、templ + Go） |
| 変更行数（テスト）         | +464 / -0 行                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/page_backlink_list/handler.go`
- [x] `go/internal/handler/page_backlink_list/show.go`
- [x] `go/internal/handler/page_link_list/show.go`
- [x] `go/internal/templates/components/backlink_list.templ`
- [x] `go/internal/templates/components/link_list.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/viewmodel/backlink_list.go`

### テストファイル

- [x] `go/internal/handler/page_backlink_list/main_test.go`
- [x] `go/internal/handler/page_backlink_list/show_test.go`

### 自動生成ファイル

- [x] `go/internal/templates/components/backlink_list_templ.go`（templ生成）
- [x] `go/internal/templates/components/link_list_templ.go`（templ生成）

### 設定・その他

- [x] `docs/plans/1_doing/link-list-alignment.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書タスク3-2の要件と実装を照合しました。

### 実装済みの要件

| 要件                                            | 状態 |
| ----------------------------------------------- | ---- |
| `page_backlink_list/handler.go` の新規作成      | ✅   |
| `page_backlink_list/show.go` の新規作成         | ✅   |
| `templates/path.go` に `GoPageBacklinkListPath` | ✅   |
| `cmd/server/main.go` にルーティング登録         | ✅   |
| `middleware/reverse_proxy.go` にパス追加        | ✅   |
| テストの追加                                    | ✅   |

### 設計との差異

作業計画書では `internal/templates/pages/page/edit.templ` を更新（バックリンクセクションにDatastarリスナーを追加）する予定だったが、diff には含まれていない。しかし、これは問題ない:

- 初回のバックリンクデータは既存の `page_link_list/show.go` SSEレスポンスに含まれている（タスク3-1で実装済み）
- 新しい `page_backlink_list/show.go` エンドポイントは「もっと見る」ボタン（`backlink_list.templ` 内の `data-on:click`）からのみ呼び出される
- `edit.templ` に別途Datastarリスナーを追加する必要がない合理的な簡略化

### リバースプロキシの確認

`/go/` プレフィックスが `reverse_proxy.go` のホワイトリストに含まれており、新しい `/go/s/{space_identifier}/pages/{page_number}/links/{linked_page_number}/backlink_list` パスは既存の設定でカバーされている。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク3-2（バックリンク一覧のSSEエンドポイント追加）が作業計画書に沿って適切に実装されている。

**良かった点**:

- **アーキテクチャ準拠**: 3層アーキテクチャの依存ルール（Handler → Repository → Query）に完全準拠。Handlerは`query.Queries`に直接依存せず、すべてRepositoryを経由している
- **セキュリティ**: 認証（ユーザーチェック）、認可（スペースメンバーチェック、トピックポリシーチェック）が適切に実装されている。リポジトリ層のクエリでは`space_id`によるスコープが確保されている
- **ハンドラーガイドライン準拠**: 標準ファイル名（`handler.go`, `show.go`）を使用し、リソースディレクトリ化（`page_backlink_list/`）が正しい
- **テストカバレッジ**: 未ログイン(401)、存在しないスペース(404)、非メンバー(404)、不正なページ番号(404)、正常系（バックリンクなし/あり/ページネーション）と、異常系・正常系が網羅されている
- **templガイドライン準拠**: テンプレートは`viewmodel`を引数に取り、`ctx`は暗黙的に使用。翻訳は`templates.T(ctx, "...")`を使用
- **Datastar構文**: `data-on:click`でコロン区切りの正しい構文を使用している
- **既存コードとの一貫性**: `page_link_list` ハンドラーと同じパターンで実装されており、コードベース全体の一貫性が保たれている
