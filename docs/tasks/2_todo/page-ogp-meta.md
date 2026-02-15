# ページのOGPメタタグ設定 タスクリスト

<!--
このテンプレートの使い方:
1. このファイルを `docs/tasks/` ディレクトリにコピー
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**タスクリストの性質**:
- タスクリストは「何をどう変えるか」という変更内容を記述するドキュメントです
- 新しい機能の場合は、概要・要件・設計もこのドキュメントに記述します
- 現在のシステムの状態は `docs/specs/` の仕様書に記述されています
- タスク完了後は、仕様書を新しい状態に更新してください（設計判断や採用しなかった方針も含める）

**仕様書との関係**:
- 新しい機能の場合: タスク完了後に `docs/specs/` に仕様書を作成する
- 既存機能の変更の場合: 「仕様書」セクションに対応する仕様書へのリンクを記載し、タスク完了後に仕様書を更新する
-->

## 仕様書

- [OGPメタタグ 仕様書](../specs/page/ogp-meta.md)（タスク完了後に作成予定）

## 概要

ページ表示画面にOGP（Open Graph Protocol）メタタグを設定する。ページタイトル、説明文、OGP画像（アイキャッチ画像）をメタタグとして出力し、SNSでのリンク共有時にリッチプレビューを表示可能にする。

Rails版では`Pages::ShowView#before_render`で`set_meta_tags`を使って設定している。

前提タスク: [ページ編集画面のGo移行](../1_doing/page-edit-go-migration.md)

### 関連タスク

- [@docs/tasks/1_doing/page-edit-go-migration.md](../1_doing/page-edit-go-migration.md) - ページ編集画面のGo移行（前提タスク）
- [@docs/tasks/2_todo/page-show-go-migration.md](page-show-go-migration.md) - ページ表示画面のGo移行（前提タスク）
- [@docs/tasks/2_todo/attachment-tracking-featured-image.md](attachment-tracking-featured-image.md) - 添付ファイル参照追跡とアイキャッチ画像（OGP画像の取得元）

## 要件

- ページ表示画面にOGPメタタグ（og:title, og:description, og:image等）が設定される
- アイキャッチ画像が存在する場合はog:imageとして設定される
- SNSでページURLを共有した際にリッチプレビューが表示される

## 実装ガイドラインの参照

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の8種類のみ**）
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

### Rails版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - 全体的なコーディング規約

## 設計

<!--
ガイドライン:
- 技術的な実装の設計を記述
-->

## 採用しなかった方針

なし

## タスクリスト

<!--
ガイドライン:
- フェーズごとに段階的な実装計画を記述
- 1タスク = 1 Pull Request の粒度で作成してください
-->

### フェーズ N: 仕様書への反映

- [ ] **N-1**: 仕様書の作成・更新
  - `docs/specs/page/ogp-meta.md` に仕様書を作成する
  - タスクリストの概要・要件・設計・採用しなかった方針を仕様書に反映する
