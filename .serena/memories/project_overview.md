# Wikino プロジェクト概要

## プロジェクトの目的

WikinoはWikiアプリケーションで、ユーザーが「スペース」と呼ばれる場所にページを作成し、ページ間をリンクで繋げることができるシステムです。

## 技術スタック

### バックエンド

- Ruby 3.4.4
- Ruby on Rails 8.0.0
- PostgreSQL
- Sorbet（型検査）
- Active Job（Solid Queue）

### フロントエンド

- TypeScript
- Hotwire (Turbo + Stimulus)
- Tailwind CSS 4
- CodeMirror 6（ページエディタ）

### ツール・ライブラリ

- パッケージマネージャー: Bundler, pnpm
- テスティング: RSpec, FactoryBot
- Linter: Standard (Ruby), ERB Lint, ESLint, Prettier
- ViewComponent, html-pipeline, meta-tags

## プロジェクト構造

- app/controllers/: HTTPリクエスト処理（1アクション1コントローラー）
- app/records/: DBテーブル操作（ActiveRecord）
- app/models/: ドメインロジック（PORO）
- app/repositories/: RecordとModel間の変換
- app/services/: データ永続化を伴うビジネスロジック
- app/policies/: 認可ルール（権限管理）
- app/components/: ViewComponent（再利用可能なUI要素）
- app/views/: ビュー（ViewComponent使用）
