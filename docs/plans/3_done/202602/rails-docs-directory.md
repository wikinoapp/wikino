# Rails 版ドキュメントの分割・整理 作業計画書

## 仕様書

- なし（新規作成）

## 概要

現在 `rails/CLAUDE.md` に集約されている Rails 版のガイドラインを、Go 版と同様に `rails/docs/` ディレクトリに分割して配置する。
これにより、作業計画書テンプレート（`docs/plans/template.md`）から個別のガイドラインを参照できるようになり、開発時の参照性が向上する。

Go 版では以下のドキュメントが `go/docs/` に存在する：

- `architecture-guide.md` - アーキテクチャガイド
- `handler-guide.md` - HTTP ハンドラーガイドライン
- `i18n-guide.md` - 国際化ガイド
- `security-guide.md` - セキュリティガイドライン
- `templ-guide.md` - templ テンプレートガイド
- `validation-guide.md` - バリデーションガイド

Rails 版でも同様に、`rails/CLAUDE.md` の内容を適切な粒度で `rails/docs/` に切り出す。

## タスクリスト

### フェーズ 1: ドキュメント分割

- [x] **1-1**: [Rails] `rails/docs/` ディレクトリの作成とドキュメント分割
  - `rails/CLAUDE.md` から以下の内容を個別ファイルに切り出す（候補）：
    - `rails/docs/architecture-guide.md` - クラス設計と依存関係、サービスクラスのルール
    - `rails/docs/testing-guide.md` - RSpec のコーディング規約、テスト戦略
    - `rails/docs/security-guide.md` - セキュリティガイドライン
  - `rails/CLAUDE.md` から切り出した箇所に各ドキュメントへのリンクを記載
  - `docs/plans/template.md` の「Rails 版の実装の場合」セクションにリンクを追加
  - **想定ファイル数**: 約 5 ファイル（実装 5 + テスト 0）
  - **想定行数**: 約 300 行（実装 300 行 + テスト 0 行）

### 実装しない機能（スコープ外）

以下は今回の実装では**実装しません**：

- **新規ガイドラインの執筆**: 既存の `rails/CLAUDE.md` の内容を移動するのみ。新しいガイドラインの追加は別タスクとする
