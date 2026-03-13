# Go 版エラーページの実装 作業計画書

## 仕様書

- タスク完了後に作成予定: `docs/specs/error-page/overview.md`

## 概要

Go 版で `http.NotFound(w, r)` が呼ばれた際に表示される 404 ページが、Go 標準の素のテキスト（`"404 page not found"`）になっている。Rails 版には `public/404.html` としてスタイル付きの 404 ページが存在するため、Go 版にも同等のエラーページを実装する。

ユーザーが存在しないページや権限のないリソースにアクセスした際に、アプリケーションのデザインに沿った 404 ページを表示することで、ユーザー体験を改善する。

## 要件

### 機能要件

- 404 レスポンス時にスタイル付きのエラーページが表示される
- エラーページには「トップページに戻る」リンクが含まれる
- 日本語・英語の両方に対応する（i18n）
- Rails 版の 404 ページ（`rails/public/404.html`）と同等のデザイン・トーンを維持する

### 非機能要件

- **パフォーマンス**: エラーページの表示はセッション取得やDB アクセスを必要としない（メンテナンスページと同様のスタンドアロン方式）
- **保守性**: 各ハンドラーの変更を最小限に抑える

## 実装ガイドラインの参照

### Go 版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 設計

### 方針: chi の NotFound ハンドラーで一括対応

chi ルーターには `r.NotFound()` メソッドがあり、ルーティングにマッチしなかった場合のハンドラーを設定できる。これに加えて、各ハンドラー内の `http.NotFound(w, r)` 呼び出しを共通のヘルパー関数に置き換える。

### エラーページのテンプレート

メンテナンスページ（`maintenance.templ`）と同様に、レイアウトに依存しないスタンドアロンの templ コンポーネントとして実装する。

理由:

- 404 発生時にはセッションや DB が利用できない可能性がある
- ログインユーザー情報の取得が不要で、シンプルに保てる
- メンテナンスページの実装パターンと統一感がある

### ファイル構成

```
go/internal/templates/pages/errors/
└── not_found.templ          # 404エラーページテンプレート

go/internal/handler/
└── errors.go                # NotFound ヘルパー関数（全ハンドラーで共用）
```

### テンプレート設計

Rails 版の 404 ページ（`rails/public/404.html`）を参考に、以下の要素を含める：

- 絵文字（🫥）による視覚的なインジケーター
- 「お探しのページは見つかりませんでした」メッセージ（i18n 対応）
- 「トップページに戻る」リンク
- CSS はインライン埋め込み（外部依存なし）

### i18n 翻訳キー

```toml
# ja.toml
[error_not_found_message]
description = "404エラーページのメッセージ"
other = "お探しのページは見つかりませんでした"

[error_not_found_back_to_top]
description = "404エラーページのトップページリンク"
other = "トップページに戻る"

# en.toml
[error_not_found_message]
description = "404 error page message"
other = "The page you are looking for could not be found"

[error_not_found_back_to_top]
description = "404 error page link to top page"
other = "Back to top page"
```

### ヘルパー関数

```go
// go/internal/handler/errors.go
package handler

// NotFound はスタイル付きの404ページをレンダリングする
func NotFound(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    errors.NotFound().Render(r.Context(), w)
}
```

### ハンドラーの変更

各ハンドラー内の `http.NotFound(w, r)` を `handler.NotFound(w, r)` に置き換える。

### chi ルーターの設定

`cmd/server/main.go` で chi ルーターに NotFound ハンドラーを登録する。これにより、ルーティングにマッチしなかった場合も同じエラーページが表示される。

```go
r.NotFound(handler.NotFound)
```

## 採用しなかった方針

### A. デフォルトレイアウト内でエラーページを表示する

サイドバーやナビゲーションを含むデフォルトレイアウトを使って404ページを表示する方針。

**不採用の理由**:

- 404 発生時にセッション情報やサイドバーデータの取得が必要になり、追加の DB アクセスが発生する
- セッションや DB が利用できない場合にエラーページ自体が表示できなくなるリスクがある
- メンテナンスページのスタンドアロン方式と統一感がなくなる

### B. `http.NotFound` をそのまま使い、ミドルウェアでレスポンスを書き換える

`ResponseWriter` をラップするミドルウェアを作成し、ステータスコード 404 のレスポンスをインターセプトしてエラーページに差し替える方針。

**不採用の理由**:

- `ResponseWriter` のラップは実装が複雑になる（`WriteHeader` と `Write` の順序制御、`Flush` や `Hijack` インターフェースの対応など）
- 各ハンドラーのヘルパー関数置き換えの方がシンプルで明示的

## タスクリスト

### フェーズ 1: 404 エラーページの実装

- [ ] **1-1**: [Go] 404 エラーページテンプレートと共通ヘルパーの実装
  - `go/internal/templates/pages/errors/not_found.templ` を作成
  - `go/internal/handler/errors.go` にヘルパー関数を作成
  - i18n 翻訳キーを `ja.toml`、`en.toml` に追加
  - `cmd/server/main.go` で chi ルーターに `r.NotFound()` を設定
  - **想定ファイル数**: 約 5 ファイル（実装 4 + テスト 1）
  - **想定行数**: 約 130 行（実装 100 行 + テスト 30 行）

### フェーズ 2: 既存ハンドラーの `http.NotFound` 置き換え

- [ ] **2-1**: [Go] 全ハンドラーの `http.NotFound(w, r)` を `handler.NotFound(w, r)` に置き換え
  - 対象ハンドラーを一括置き換え
  - **想定ファイル数**: 約 15 ファイル（実装 15 + テスト 0）
  - **想定行数**: 約 50 行（実装のみ、各ファイル 1-3 行の変更）

### フェーズ 3: 仕様書への反映

- [ ] **3-1**: 仕様書の作成・更新
  - `docs/specs/error-page/overview.md` に仕様書を作成する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **500 エラーページ**: 今回は 404 のみを対象とする。500 ページは別途計画する
- **403 Forbidden ページ**: 現在は 404 として扱っているため、専用ページは不要
