# 推奨コマンドリスト

## 開発環境セットアップ
```bash
docker compose up      # Docker環境の起動
mise install          # mise経由での依存関係インストール
bin/setup            # 初期セットアップ
bin/dev              # 開発環境の起動
bin/rails server     # Railsサーバーの起動
```

## 開発中によく使うコマンド

### 包括的なチェック
```bash
bin/check            # 全ての検証を実行（推奨）
```

### 個別のチェック・修正コマンド
```bash
# Ruby関連
bin/standardrb       # Ruby Linter (Standard)
bin/erb_lint --lint-all  # ERB Linter
bin/srb tc          # Sorbet型チェック
bin/rails sorbet:update  # Sorbet型定義の更新
bin/rails zeitwerk:check # オートローディングのチェック

# JavaScript/TypeScript関連
pnpm prettier . --write  # Prettierでコード整形
pnpm eslint . --fix     # ESLintでコード修正
pnpm tsc               # TypeScript型チェック

# テスト
bin/rspec              # テスト実行
```

## Git操作（Darwin/macOS）
```bash
git status            # 変更状況の確認
git diff             # 差分の確認
git add .            # 全ファイルをステージング
git commit -m "message"  # コミット
git push             # プッシュ
```

## ファイル操作（Darwin/macOS）
```bash
ls -la               # ファイル一覧（隠しファイル含む）
find . -name "*.rb"  # ファイル検索
grep -r "text" .     # テキスト検索
```

## タスク完了時の推奨フロー
1. `bin/check` を実行して全ての検証をパス
2. 必要に応じて個別のコマンドで修正
3. テストが通ることを確認