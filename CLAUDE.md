# Wikino 開発ガイドライン

## プロジェクト概要

WikinoはWikiアプリケーションです。
ユーザーは「スペース」と呼ばれる場所にページを作成し、ページ間をリンクで繋げることができます。

## モノレポ構造

このリポジトリは、Go 版と Rails 版の 2 つのサブプロジェクトをモノレポとして管理しています：

```
/workspace/
├── go/                  # Go版の実装（段階的に機能を移行中）
├── rails/               # Rails版の実装（既存の本番システム）
├── caddy/               # リバースプロキシ設定
├── docs/                # Wikino固有のドキュメント（仕様書、作業計画書など）
├── .claude/             # AIガイドライン・スキル（apm install で自動配置）
├── .github/             # 共通のCI/CD設定
├── apm.yml              # APM（Agent Package Manager）の依存関係定義
├── apm.lock.yaml        # APM のロックファイル
├── apm_modules/         # APM の作業ディレクトリ
├── Dockerfile.dev       # 統合開発コンテナのDockerfile
├── docker-compose.yml   # Docker Compose設定
├── mise.toml            # 開発ツールバージョン管理（Go, Ruby）
└── CLAUDE.md            # このファイル（プロジェクト全体のガイド）
```

## Rails から Go への移行について

現在、既存の Rails 実装の Wikino を Go で段階的に再実装するプロジェクトが進行中です。

### 移行の基本方針

- **既存 DB をそのまま使用**: Rails 側で管理されている PostgreSQL データベースを共有
- **段階的移行**: Rails と Go が同一の DB とセッションストアを共有し、段階的に機能を移行
- **データマイグレーション不要**: DB スキーマは既存のものを使用し、データ移行は行わない
- **共通インフラの継続利用**: PostgreSQL などの共通インフラは Go 版移行後も継続して使用

### 移行時の設計方針

- **ページネーション**: Go 版ではオフセットベースページネーションを採用する。Rails 版ではカーソルベースページネーション（`cursor_paginate` gem）を使用しているが、Go 版では実装のシンプルさとページ番号による直感的なナビゲーションを優先し、オフセットベースとする

### Rails 側のソースコード

Rails 版のソースコードは `/workspace/rails/` 配下に格納されています：

```
/workspace/rails/
├── app/controllers/     # コントローラー
├── app/records/         # ActiveRecordモデル
├── app/use_cases/       # ユースケース（ビジネスロジック）
├── app/views/           # ビューテンプレート
├── config/routes.rb     # ルーティング定義
└── db/structure.sql     # DBスキーマ
```

Go 版を実装する際は、Rails 版のコードを参考にすることで既存の仕様を理解できます。

## 共通インフラ

### データベース（PostgreSQL）

- **バージョン**: PostgreSQL 18.1
- **共有方針**: Rails 版と Go 版で同一のデータベースを共有
- **開発環境**: Docker Compose で管理（ポート: 4204）
- **データベース名**:
  - 開発: `wikino_development`
  - テスト: `wikino_test`

### セッションストア（PostgreSQL）

- **ストレージ**: PostgreSQL の `sessions` テーブルを使用
- **Rails 版**: ActiveRecord SessionStore を使用
  - 各リクエストで `updated_at` カラムを自動更新
  - セッションの有効期限: 30 日
- **Go 版**: 同じ `sessions` テーブルを共有
  - 認証ミドルウェアで `updated_at` カラムを更新
  - Rails 版と完全に互換性のあるセッション管理を実現
- **セッションクリーンアップ**: 毎日 19:00 に `rake session:sweep` タスクが実行され、30 日以上前のセッションを自動削除
- **共有方針**: Rails 版と Go 版で同一のセッションストアを共有（段階的移行を実現）

## 開発環境のセットアップ

### 前提条件

- Docker 及び Docker Compose がインストール済み

### セットアップ手順

1. **リポジトリのクローン**

```sh
git clone git@github.com:wikinoapp/wikino.git
cd wikino
```

2. **Docker Compose の起動**

```sh
docker compose up
```

### 開発サーバーの起動

プロジェクトルートで以下のコマンドを実行すると、Go 版・Rails 版の全サービスを一括で起動できます：

```sh
make dev
```

このコマンドは [hivemind](https://github.com/DarthSim/hivemind) を使用して `Procfile.dev` に定義された以下のプロセスを並行起動します：

| プロセス       | 内容                                        |
| -------------- | ------------------------------------------- |
| `go-server`    | Go 版サーバー（air によるホットリロード）   |
| `go-assets`    | Go 版フロントエンドアセットの監視・再ビルド |
| `rails-server` | Rails 版サーバー                            |
| `rails-css`    | Rails 版 CSS の監視・再ビルド               |
| `rails-js`     | Rails 版 JavaScript の監視・再ビルド        |

環境変数や Go / Rails 固有のセットアップ手順は、`.claude/rules/go-development.md` と `.claude/rules/rails-common.md` を参照してください。

## ドキュメント

ドキュメントは `docs/` 配下で管理しており、ユーザーが直接体験する機能の仕様は `docs/specs/` に、ユーザーが直接体験しないシステム内部の仕組みは `docs/system/` に配置しています。配置先の判断基準やディレクトリ構成は [@docs/README.md](/workspace/docs/README.md) を参照してください。

- [@docs/README.md](/workspace/docs/README.md) - ドキュメント全体のガイド
- [@docs/specs/](/workspace/docs/specs/) - サービス仕様書 (ユーザーが直接体験する機能)
- [@docs/system/](/workspace/docs/system/) - システム仕様書 (ユーザーが直接体験しないシステム内部の仕組み)

## 参照するガイドライン

Claude Code は `.claude/rules/` 配下のガイドラインを自動で読み込むため、通常は特に意識せず書いて OK。ガイドラインの実体は `korylus-guidelines` から `apm install` で配置されています。

- **Korylus 共通**: `.claude/rules/common.md` / `.claude/rules/apm.md` / `.claude/rules/guidelines-authoring.md`
- **Go 版**: `.claude/rules/go-*.md` (coding, architecture, handler, usecase, testing, validation, security, templ, i18n, development)
- **Rails 版**: `.claude/rules/rails-*.md` (common, architecture, testing, security)

APM 管理下のファイルは `apm install` で上書きされます。編集したい場合は `korylus-guidelines` 側を修正してください。

## Wikino 固有のガイドライン

`apm install` で配置される共通ガイドライン (`.claude/rules/`) に加えて、Wikino プロジェクト固有の規約を本セクションに記述する。Korylus の他プロダクトには適用されない、Wikino 独自のドメイン規約・セキュリティ規約を扱う。

### スペース ID によるクエリスコープ

主に Go 版の SQL クエリ (sqlc / 生 SQL) に適用する規約。Rails 版では ActiveRecord の関連 (`current_space.pages` など) を経由することで同等のスコープを実現しているため、本規約の対象外。

#### 基本方針

Wikino ではスペースがデータの分離境界です。スペース内のリソースに対するクエリには、**必ず `space_id` を WHERE 条件に含めてください**。

ハンドラーやユースケースでの認可チェックに加え、クエリレベルでもスペースをまたいだデータアクセスを防止する防御的プログラミングです。

#### 対象テーブル

`space_id` カラムを持つテーブル:

- `pages`, `draft_pages`, `topics`, `topic_members`, `space_members`
- `page_editors`, `page_revisions`, `attachments`

`space_id` カラムを持たないが、スペース内リソースを参照するテーブル:

- `page_attachment_references`（`pages` 経由でスペースに紐づく）

#### 実装方法

```sql
-- ✅ Good: テーブル自体に space_id がある場合は直接条件に含める
UPDATE pages SET title = $2 WHERE id = $1 AND space_id = $3;
DELETE FROM draft_pages WHERE id = $1 AND space_id = $2;

-- ✅ Good: テーブルに space_id がない場合は JOIN で検証する
SELECT par.* FROM page_attachment_references par
INNER JOIN pages p ON par.page_id = p.id
WHERE par.page_id = $1 AND p.space_id = $2;

-- ❌ Bad: space_id なしで ID のみで操作している
UPDATE pages SET title = $2 WHERE id = $1;
DELETE FROM draft_pages WHERE id = $1;
SELECT * FROM page_attachment_references WHERE page_id = $1;
```

#### なぜ必要か

- **防御の多層化**: 認可チェックとクエリスコープの 2 重防御により、どちらかに不備があっても安全
- **影響範囲の限定**: 万一不正な ID が渡されても、操作がスペース内に閉じる
- **意図の明確化**: クエリを読むだけでスペーススコープであることが分かる

### 環境変数プレフィックス

Wikino で定義する環境変数には、外部ライブラリなどが指定してくるものを除き、**必ず `WIKINO_` プレフィックスを付ける**。Go 版・Rails 版の両方で同一の規約を適用する。

- 例: `WIKINO_PORT`, `WIKINO_DOMAIN`, `WIKINO_RAILS_APP_URL`
- 外部ライブラリが要求する環境変数はそのまま使用 (例: `DATABASE_URL`, `REDIS_URL`)
- **例外**: `APP_ENV` などの慣習的な環境変数は `WIKINO_` プレフィックスなしで使用する

## 開発ワークフロー

### フィーチャーフラグによる開発

Korylus 共通の方針は [.claude/rules/common.md](/workspace/.claude/rules/common.md) の「フィーチャーフラグによる開発」セクションを参照してください。

Wikino における具体的な仕組み (DB スキーマ、リバースプロキシの判定ロジックなど) は仕様書を参照:

- [@docs/system/feature-flag/overview.md](/workspace/docs/system/feature-flag/overview.md) — フィーチャーフラグ 仕様書

## CI/CD

このモノレポの CI/CD 設定は`.github/workflows/`ディレクトリに配置されています：

- `go-ci.yml`: Go 版の CI（lint、test、build）
- `fmt-ci.yml`: フォーマットチェック（Oxfmt）
- `rails-ci.yml`: Rails 版の CI（zeitwerk、sorbet、standard、erb_lint、eslint、rspec）

各 CI は対応するファイルが変更されたときに実行されます。

## トラブルシューティング

### データベース接続エラー

- PostgreSQL コンテナが起動しているか確認: `docker compose ps`
- ポートが正しいか確認: 開発環境は 4204
