# Wikino開発ガイドライン

このファイルは、コーディングエージェントがこのリポジトリで作業する際のガイダンスを提供します。

## 概要

WikinoはWikiアプリケーションです。
ユーザーは「スペース」と呼ばれる場所にページを作成し、ページ間をリンクで繋げることができます。

## プロジェクト構造

このリポジトリは、Go版とRails版の2つのサブプロジェクトをモノレポとして管理しています。

```
/workspace/
├── go/                  # Go版の実装 (段階的に機能を移行中)
├── rails/               # Rails版の実装 (既存の本番システム)
├── caddy/               # リバースプロキシ設定
├── docs/                # Wikino固有のドキュメント (ADR、作業計画書など)
├── .github/             # 共通のCI/CD設定
├── Dockerfile.dev       # 統合開発コンテナのDockerfile
├── docker-compose.yml   # Docker Compose設定
├── mise.toml            # 開発ツールバージョン管理 (Go, Ruby)
├── AGENTS.md            # このファイル (プロジェクト全体のガイド)
└── CLAUDE.md            # AGENTS.md へのポインタ (Claude Code用)
```

## RailsからGoへの移行について

現在、既存のRails実装のWikinoをGoで段階的に再実装するプロジェクトが進行中です。

### 移行の基本方針

- **既存DBをそのまま使用**: Rails側で管理されているPostgreSQLデータベースを共有
- **段階的移行**: RailsとGoが同一のDBとセッションストアを共有し、段階的に機能を移行
- **データマイグレーションはGo側で実行**: Go側に用意しているマイグレーション機構 (dbmate) を使用
- **共通インフラの継続利用**: PostgreSQLなどの共通インフラはGo版移行後も継続して使用
- **Rails側のソースコードは変更しない**: 機能の追加・変更が必要なときは、Rails側をいじらずまずGoに移行する
  - ただし、以下の場合はこの原則の対象外とし、Rails側で最小差分の修正を行って良い
    - 依存パッケージのセキュリティ修正に追随するための最小限の保守変更 (例: gemのメジャーアップに伴う破壊的変更への対応)
    - Go移行に伴って不要になったRails側の処理を削除するとき
    - 本番のエラー監視 (Sentry) で通知されたエラーへの対応として行う最小限の修正 (例: 未処理例外による500の抑止)

Go版を実装する際は、Rails版のコードを参考にすることで既存の仕様を理解できます。

## フィーチャーフラグによる開発

Wikinoではフィーチャーブランチではなく **フィーチャーフラグ** を使って機能の公開を制御しています。
リリース前の機能はフラグでオフのまま開発し、本番投入の準備が整ってからフラグを切り替えて公開します。

## 開発ワークフロー

### 実装時のガイドライン

**既存コードとの一貫性**:

実装を行う前に、コードベース内に類似の処理がないか確認してください。
類似処理が存在する場合は、そのパターンに従って実装することで、コードベース全体の一貫性を保ちます。

### 実装後のチェック

実装を終え作業の完了を伝える前に、必ず以下を確認してください:

- コードフォーマット
- リント
- テスト

実行するコマンドは `Makefile` で管理しています。
[Makefile](./Makefile), [go/Makefile](./go/Makefile), [rails/Makefile](./rails/Makefile) を参照してください。

## ドキュメント

システムの現在の状態 (バリデーション・権限・構造) はコードとテストを正本とします。
ドキュメントは、設計判断の「なぜ」を記録する **ADR (Architecture Decision Record)** と、進行中の変更 (作業計画書・レビュー) に集中させます。

なぜその設計になっているかを理解するにはまずADRを、現在どう動くかを理解するにはコードとテストを参照してください。

- [docs/README.md](./docs/README.md) - ドキュメントガイド。ADRは `docs/private/adr/` に置いています

## 言語・文章ルール

このリポジトリの開発言語は日本語です。
コードコメント・ドキュメント・コミットメッセージ・プルリクエスト・Issueは、すべて日本語のみで書きます。

英語を使うのは次の2つだけです。

- **技術的慣習として英語であるもの**: 識別子 (型名・関数名・変数名)、APIのフィールド名、ログのフィールドキー、エラーコード、環境変数名
- **リポジトリの入口に置く案内板**: `README.md` / `CONTRIBUTING.md` / `SECURITY.md` の3ファイルに英語版 (`xxx.en.md`) を併置する

日本語テキストでは半角丸括弧を使い、括弧の両端に半角スペースを入れます (行頭・行末の外側は不要)。
英数字と日本語の間にはスペースを入れません。

## コーディング規約

- `space_id` カラムを持つテーブルに対するクエリには、必ず `space_id` をWHERE条件に含めること

```sql
-- ✅ Good: テーブル自体にspace_idがある場合は直接条件に含める
UPDATE pages SET title = $2 WHERE id = $1 AND space_id = $3;
DELETE FROM draft_pages WHERE id = $1 AND space_id = $2;

-- ✅ Good: テーブルにspace_idがない場合はJOINで検証する
SELECT par.* FROM page_attachment_references par
INNER JOIN pages p ON par.page_id = p.id
WHERE par.page_id = $1 AND p.space_id = $2;

-- ❌ Bad: space_idなしでIDのみで操作している
UPDATE pages SET title = $2 WHERE id = $1;
DELETE FROM draft_pages WHERE id = $1;
SELECT * FROM page_attachment_references WHERE page_id = $1;
```

- Wikinoで定義する環境変数には、外部ライブラリなどが指定してくるものを除き、必ず `WIKINO_` プレフィックスを付けること
