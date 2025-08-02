# Wikino プロジェクト概要

WikinoはWikiアプリケーションです。

## 主な機能
- ユーザーは「スペース」と呼ばれる場所にページを作成できる
- ページ間をリンクで繋げることができる
- 国際化対応（日本語と英語）

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
- CSS: Tailwind CSS（@tailwindcss/cli）
- その他: ViewComponent, html-pipeline, meta-tags