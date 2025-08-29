# Wikino 開発コマンド一覧

## 環境セットアップ

```bash
docker compose up      # Docker環境起動
mise install          # 依存関係インストール
bin/setup            # 初期セットアップ
bin/dev              # 開発環境起動
bin/rails server     # サーバー起動
```

## Ruby検証コマンド

```bash
bin/standardrb         # Rubyコードのフォーマット
bin/erb_lint --lint-all # ERBファイルのLint
bin/srb tc            # Sorbet型チェック
bin/rails sorbet:update # Sorbet型定義更新
bin/rails zeitwerk:check # オートローディングチェック
bin/rspec             # テスト実行
bin/rspec path/to/spec.rb # 特定のテスト実行
```

## JavaScript/TypeScript検証コマンド

```bash
pnpm prettier . --write # コードフォーマット
pnpm eslint . --fix    # ESLint実行・修正
pnpm tsc              # TypeScript型チェック
pnpm build:css        # CSS構築
```

## 全体検証

```bash
bin/check             # 全ての検証を実行
```

## Git操作

```bash
git status            # 変更状況確認
git diff              # 変更内容確認
git add .             # 変更をステージング
git commit -m "msg"   # コミット
```

## Rails操作

```bash
bin/rails c           # コンソール起動
bin/rails generate    # ジェネレータ実行
RAILS_ENV=test bin/rails c # テスト環境のコンソール
RAILS_ENV=test bin/rails log:clear # テストログクリア
```

## システムコマンド（Darwin）

```bash
ls                    # ファイル一覧
find . -name "*.rb"   # ファイル検索
rg "pattern"          # ripgrepで検索（grepの代替）
touch file            # ファイル作成
mkdir dir             # ディレクトリ作成
```
