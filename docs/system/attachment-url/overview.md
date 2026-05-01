# 添付ファイル URL 配信 仕様書

## 概要

ページ本文先頭の画像を `<meta property="og:image">` として配信する仕組みである。Rails が HTML をレンダリングする際に「再検証付き永続 URL」(`/attachments/:id/og_image`) を出力し、Go の専用ハンドラーがアクセス時に公開判定を再評価したうえで imgproxy 経由のリサイズ済み画像へ 302 リダイレクトする。

URL 文字列自体は失効しないため、ブラウザ・CDN・SNS クローラが HTML を長期キャッシュしても og:image が壊れない。一方、トピックの公開・非公開が変更された場合は、Go ハンドラーが 1 SQL で「生きている公開トピックのページから参照されている添付か」を再判定するため、非公開化された添付は即座に 404 でブロックされる。

画像変換は **imgproxy** (外部サービス) が担当する。Rails Active Storage の variant 機構には依存せず、元画像 (S3 / R2 上の blob) のみを保持して on-the-fly でリサイズ・JPEG 変換する設計とする。これにより Rails 撤去後も画像配信パイプラインが Go 側で自己完結し、将来の本文中画像やサムネイル配信も同じ仕組みに統合できる。

**目的**:

- og:image の URL を HTML キャッシュ寿命に左右されず無期限に有効化し、SNS のリッチプレビューが時間経過で壊れる不具合を解消する
- トピック visibility の変更に対し、URL ではなくアプリ層で即時追従できるアクセス制御を提供する
- 画像変換パイプラインを Active Storage variant から imgproxy へ移行し、Rails 撤去に向けた依存関係を切り離す

**背景**:

- 旧実装では Rails の `PageRecord#og_image_url` が S3 signed URL (有効期限 1 時間) を埋め込んでいた。HTML レスポンスを CDN や SNS クローラが長期キャッシュするため、1 時間後にキャッシュ済み HTML 経由で og:image にアクセスすると S3 が `InvalidArgument` を返し、リッチプレビューが壊れていた
- 期限を伸ばすだけでは「公開→非公開」変更時の漏洩リスクが大きく、根本解決にならない。アクセスごとにアプリ層で公開判定する設計に切り替えた
- 本文中画像のダウンロード URL (`/attachments/:id`) はページ表示時に都度 URL 解決されるためキャッシュ問題が顕在化しておらず、本仕様の対象外 (Rails が従来通り signed URL で配信する)

## 仕様

### URL の発行

- `PageRecord#og_image_url` は、ページに featured image があり、かつ GIF でない場合に `"#{Wikino.config.app_url}/attachments/#{attachment.id}/og_image"` を返す
  - featured image が存在しない場合: `nil`
  - GIF の場合: `nil` (デフォルト OGP 画像にフォールバックさせるため)
- visibility 判定は URL 発行時には行わない。常に永続 URL を返し、判定はアクセス時に Go ハンドラーが行う
- `Wikino.config.app_url` は環境変数 `WIKINO_URL` から解決される

### 配信エンドポイント (GET /attachments/:id/og_image)

- 認証不要のエンドポイント。URL を知っている任意のクライアント (SNS クローラ・ゲスト含む) から閲覧可能
- 「生きている公開トピックのページから参照されている添付ファイル」の場合のみ、imgproxy 上のリサイズ画像へ 302 リダイレクトする
- 以下のいずれにも当てはまる場合は、配信判定を満たすとみなす:
  - `attachments.id` が一致するレコードが存在する
  - そのレコードを参照する `page_attachment_references` 経由で、`pages.discarded_at IS NULL` かつ `topics.discarded_at IS NULL` の生きている参照が 1 件以上存在する
  - 上記の生きている参照について、トピックの `visibility = 0` (公開) を満たす参照のみで構成されている (1 件でも非公開トピックを含む場合は配信しない)
- 配信判定を満たさない場合は、レスポンス上では存在しない添付と区別せず 404 を返す (添付の存在を秘匿する)

### レスポンス

| ケース                      | ステータス | レスポンスヘッダ                                                               | ボディ                             |
| --------------------------- | ---------- | ------------------------------------------------------------------------------ | ---------------------------------- |
| 配信判定を満たす            | 302 Found  | `Cache-Control: public, max-age=60, s-maxage=300` + `Location: <imgproxy URL>` | (リダイレクト先で画像ボディを返す) |
| 不正な UUID / 存在しない ID | 404        | `Cache-Control: private, no-store`                                             | 404 HTML ページ                    |
| 配信判定を満たさない添付    | 404        | `Cache-Control: private, no-store`                                             | 404 HTML ページ                    |
| imgproxy 設定が未構成       | 500        | `Cache-Control: private, no-store`                                             | `Internal Server Error`            |

- 302 のキャッシュは CDN・ブラウザに許可するが、`max-age=60`、`s-maxage=300` で leak window を短く抑える
- 404 / 500 は CDN にキャッシュさせない (`Cache-Control: private, no-store`)。「公開→非公開→公開」と切り替わったときに古い 404 が CDN から返り続ける逆方向 leak を防ぐため

### 配信画像のサイズ・フォーマット

- Go ハンドラーは imgproxy が要求する署名付き URL を組み立てて 302 でリダイレクトする
- リサイズ・フォーマットは og:image 用に固定:
  - サイズ: `resize:fit:1200:630` (OGP の推奨サイズ)
  - フォーマット: `format:jpg` 固定 (`auto` ではなく) — Slack / X / Discord 等 SNS クローラの一部が WebP / AVIF を扱えず、サムネイル生成が壊れるケースがあるため
  - 署名 TTL: 発行時刻 + 1 時間 (`expires:<unix>`)。HTML キャッシュ寿命 (`s-maxage=300`) を十分に上回り、CDN・ブラウザのリダイレクトキャッシュにも余裕を持たせる
- 元画像のソース URL は `s3://{R2_BUCKET_NAME}/{active_storage_blobs.key}`。imgproxy 側は `IMGPROXY_USE_S3=true` + `IMGPROXY_S3_ENDPOINT` で Cloudflare R2 から読み取る

### キャッシュ層と visibility 変更の整合性

ハンドラーがアクセスごとに visibility を再評価するため、アプリ層では visibility 変更が即座に反映される。一方、上位のキャッシュ層は以下のように leak window を持つ。

| キャッシュ層                  | visibility 変更後の挙動                                                        | leak window              |
| ----------------------------- | ------------------------------------------------------------------------------ | ------------------------ |
| **CDN** (Cloudflare 等)       | `s-maxage` が経過するまで古い 302 レスポンスを配信し続ける                     | `s-maxage` の値 (300 秒) |
| **ブラウザキャッシュ**        | `max-age` が経過するまで古いリダイレクト先に飛ぶ                               | `max-age` の値 (60 秒)   |
| **imgproxy 側のキャッシュ**   | imgproxy が画像をキャッシュしている場合、そのキャッシュは expire まで生きる    | imgproxy 設定次第        |
| **SNS クローラ** (Slack/X 等) | 各サービスが独自にサムネイルをキャッシュ。`Cache-Control` を必ずしも尊重しない | 数時間〜数日 (制御不能)  |

SNS クローラのキャッシュは構造的に制御不能であり、これはどの URL 戦略 (signed URL でも永続 URL でも) でも変わらない構造的制約である。CDN・ブラウザ側の leak window は `s-maxage` / `max-age` で短く保つ運用としている。

### セキュリティ

- 公開判定は Repository の SQL クエリ (`FindPubliclyReferencedAttachmentBlobByID`) に統合されており、呼び出し側で検証を忘れる構造的事故が発生しない
- 不正な UUID 形式の `attachment_id` は Repository の正規表現で弾かれ、DB クエリに到達しない
- 配信判定を満たさない場合と存在しない ID の場合でレスポンスを区別しないため、添付 ID の存在有無を外部から推測できない
- 公開判定 SQL は space スコープを取らない。URL 文字列を知っているクライアント (ゲスト含む) であれば誰でも閲覧可能であることを前提にしている

### パフォーマンス

- 公開判定は 1 SQL で完結する (`page_attachment_references` 経由のサブクエリ 2 つ + 添付本体 SELECT)
- 元画像の取得・リサイズ・フォーマット変換は imgproxy がオンザフライで行うため、Go アプリプロセスは画像処理を行わない
- リダイレクト先の imgproxy URL は CDN にキャッシュされ、2 回目以降は imgproxy へのリクエストも省略される (CDN ヒット時)

## 設計

### 配信フロー

```
[og:image URL 発行]
  Rails: PageRecord#og_image_url
    → "https://example.dev/attachments/{id}/og_image" を返す (永続 URL)
    → ページ HTML の <meta property="og:image" content="..."> に埋め込まれる

[og:image アクセス]
  GET /attachments/:id/og_image (Go: handler/attachment_og_image)
    1. リバースプロキシが Go 側に振り分け (常時 Go 処理)
    2. UseCase が Repository.FindPubliclyReferencedBlobByID を呼ぶ
       - 内部で page_attachment_references / pages / topics を JOIN し公開判定を 1 SQL で実施
       - 判定を満たさなければ nil → UseCase が AppError(NotFound) を返し Handler が 404
    3. OgImageBuilder が imgproxy 用署名付き URL を組み立てる
       例: https://imgproxy.example.dev/{sig}/resize:fit:1200:630/expires:<unix>/format:jpg/plain/s3://{bucket}/{blob_key}
    4. 302 redirect で imgproxy URL を返す
       Cache-Control: public, max-age=60, s-maxage=300

[imgproxy 側]
  S3 (Cloudflare R2) から元画像 blob を取得 → リサイズ・JPEG 変換 → 画像レスポンス
  CDN がリダイレクト先 imgproxy URL もキャッシュ

[再アクセス時]
  CDN がリダイレクト (302) と imgproxy のレスポンスを max-age に従ってキャッシュ
  ブラウザは Cache-Control に従ってキャッシュ
```

### 公開判定 SQL

公開判定は 1 SQL で表現し、Repository 内に閉じている (Go 側にロジックを持ち出さない)。

```sql
-- go/db/queries/attachments.sql: FindPubliclyReferencedAttachmentBlobByID

SELECT a.id, a.space_id, asb.key AS blob_key, asb.content_type AS blob_content_type
FROM attachments a
INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
WHERE a.id = $1
  AND EXISTS (
    SELECT 1 FROM page_attachment_references par
    INNER JOIN pages p ON par.page_id = p.id
    INNER JOIN topics t ON p.topic_id = t.id
    WHERE par.attachment_id = a.id
      AND p.discarded_at IS NULL
      AND t.discarded_at IS NULL
  )
  AND NOT EXISTS (
    SELECT 1 FROM page_attachment_references par
    INNER JOIN pages p ON par.page_id = p.id
    INNER JOIN topics t ON p.topic_id = t.id
    WHERE par.attachment_id = a.id
      AND p.discarded_at IS NULL
      AND t.discarded_at IS NULL
      AND t.visibility <> 0
  );
```

Rails 版 `AttachmentRecord#all_referencing_pages_public?` (= `referencing_topics.any? && referencing_topics.all?(&:visibility_public?)`) と等価な判定を 1 SQL に統合している。判定スコープからは論理削除済みのページ・トピック (`discarded_at IS NOT NULL`) を除外する。

### コード構成

#### Go 側

| パッケージ / ファイル                          | 責務                                                                                     |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `internal/handler/attachment_og_image/show.go` | 公開 og:image エンドポイント (302 リダイレクト + Cache-Control 設定)                     |
| `internal/usecase/get_attachment_og_image.go`  | 公開判定済み blob 情報の取得 (NotFound 系エラーは AppError に変換)                       |
| `internal/repository/attachment.go`            | `FindPubliclyReferencedBlobByID` (公開判定 SQL のラッパー)                               |
| `internal/image/og_image.go`                   | `OgImageBuilder` (og:image 用のリサイズ・フォーマット・TTL ポリシーを集約)               |
| `internal/image/imgproxy.go`                   | `Helper` (HMAC-SHA256 署名付き imgproxy URL の組み立て、og:image 以外にも利用可)         |
| `internal/middleware/reverse_proxy.go`         | `/attachments/{id}/og_image` (GET 限定) を常に Go 処理する `goHandledRegexPatterns` 登録 |

`OgImageBuilder` は og:image 専用のリサイズ・フォーマット・TTL ポリシーを内部定数として保持する (`1200x630` / `jpg` / 1 時間)。本文中画像など他用途で別ポリシーが必要になった場合は別の Builder を用意する想定。

#### Rails 側

`PageRecord#og_image_url` は visibility 判定を行わず、常に Go の永続 URL を返す。

```ruby
# app/records/page_record.rb
sig { returns(T.nilable(String)) }
def og_image_url
  attachment = featured_image_attachment_record
  return nil unless attachment
  return nil if featured_image_is_gif?

  "#{Wikino.config.app_url}/attachments/#{attachment.id}/og_image"
end
```

### imgproxy のセットアップ

#### 開発環境

`docker-compose.yml` に imgproxy サービスを追加し、Cloudflare R2 (Active Storage と同じバケット) を参照させる。

```yaml
imgproxy:
  image: ghcr.io/imgproxy/imgproxy:v3.31.2
  ports:
    - "4206:8080"
  environment:
    - AWS_ACCESS_KEY_ID=${WIKINO_IMGPROXY_AWS_ACCESS_KEY_ID:-}
    - AWS_SECRET_ACCESS_KEY=${WIKINO_IMGPROXY_AWS_SECRET_ACCESS_KEY:-}
    - IMGPROXY_ALLOWED_SOURCES=${WIKINO_IMGPROXY_ALLOWED_SOURCES:-}
    - IMGPROXY_KEY=${WIKINO_IMGPROXY_KEY:-}
    - IMGPROXY_S3_ENDPOINT=${WIKINO_IMGPROXY_S3_ENDPOINT:-}
    - IMGPROXY_S3_REGION=${WIKINO_IMGPROXY_S3_REGION:-}
    - IMGPROXY_SALT=${WIKINO_IMGPROXY_SALT:-}
    - IMGPROXY_USE_S3=${WIKINO_IMGPROXY_USE_S3:-}
```

#### 環境変数

Go アプリが参照する環境変数 (Wikino プレフィックス):

| 環境変数                | 用途                                                                   |
| ----------------------- | ---------------------------------------------------------------------- |
| `WIKINO_IMGPROXY_URL`   | imgproxy のベース URL (例: `https://imgproxy.example.dev`)             |
| `WIKINO_IMGPROXY_KEY`   | 署名用の秘密鍵 (16 進数文字列)                                         |
| `WIKINO_IMGPROXY_SALT`  | 署名用の salt (16 進数文字列)                                          |
| `WIKINO_R2_BUCKET_NAME` | 元画像 URL `s3://{bucket}/{key}` の bucket 部分。imgproxy ソース構築用 |

`WIKINO_IMGPROXY_URL` または `WIKINO_R2_BUCKET_NAME` が未設定の場合、Go アプリは `OgImageBuilder` を構築せずに WARN ログを出して起動を継続する。設定不備のままエンドポイントへリクエストが届いた場合は 500 を返し、設定ミスを早期に可視化する。

imgproxy コンテナは AWS SDK の慣習に従い `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` をプレフィックスなしで受け取る。ホスト側の `.env` には `WIKINO_IMGPROXY_AWS_ACCESS_KEY_ID` / `WIKINO_IMGPROXY_AWS_SECRET_ACCESS_KEY` を `WIKINO_` プレフィックス付きで定義し、docker-compose.yml で imgproxy コンテナ内の `AWS_*` にマッピングする (CLAUDE.md の例外規定: 外部ライブラリが要求する変数名はそのまま使用)。

### ルーティング

`/attachments/:id/og_image` は GET 限定で `goHandledRegexPatterns` に登録され、リバースプロキシミドルウェアが常に Go 側に振り分ける (フィーチャーフラグでの段階公開は完了済み)。`/attachments/:id` (本文中画像のダウンロード URL) は引き続き Rails が処理する。

```go
// internal/middleware/reverse_proxy.go (抜粋)
{pattern: regexp.MustCompile(`^/attachments/[^/]+/og_image$`), methods: []string{http.MethodGet}}
```

## 採用しなかった方針

### 非公開トピックの og:image を従来通り signed URL で配信する

「非公開トピックの添付ファイルは従来通り signed URL (1 時間期限) を返し、公開トピックのみ永続 URL を返す」という分岐方式を検討した。

**不採用の理由**: Go 側エンドポイントが公開判定を 1 SQL で再評価し、満たさなければ 404 を返すため、非公開トピックの永続 URL を埋め込んでもクローラには 404 が返るだけで漏洩しない。永続 URL に統一することで、Rails 側でページ一覧表示時の N+1 (`page_attachment_references` / `pages` / `topics` の追加クエリ) も解消できる。実装もシンプル。

### 添付ファイルにトピック帰属 (`topic_id` カラム) を持たせる

添付ファイルに `topic_id` を持たせ、公開・非公開を直接判定する案を検討した。

**不採用の理由**: 現状の `all_referencing_pages_public?` ロジックは、ページ移動 (公開トピック→非公開トピック) に対して自動で正しく追従する。`page_attachment_references` 経由で常に最新の参照ページの visibility を見るため、添付ファイルに `topic_id` を持たせて手動で同期する設計より堅牢。同じ添付が複数ページから参照されるケースも 1 つの `topic_id` では表現できない。

### og:image URL の有効期限を 30 日などに延長する

HTML キャッシュ寿命を上回る期限を設定する案を検討した。

**不採用の理由**: 期限切れの先延ばしであり根本解決ではない。トピック visibility が「公開→非公開」に変更された際、最大 30 日間古い URL が有効になる残存リスクが大きい。

### S3 オブジェクトの ACL を `public-read` に切り替える

公開トピックの添付ファイルを S3 上で公開設定にし、直接の S3 URL を埋め込む案を検討した。

**不採用の理由**: トピック visibility 変更時に S3 オブジェクトの ACL を一括更新するバッチが必要で、運用コストが高い。アップロード時にも公開・非公開の判定が必要になり、影響範囲が広がる。imgproxy 経由のアプローチで十分なパフォーマンスが出る見込み。

### Rails で og:image エンドポイントを実装する

og:image エンドポイントを Rails 側に実装する案を検討した。

**不採用の理由**: リバースプロキシで Go にパスを振り分けられるため、HTML を Rails が描画している間でも Go 実装で機能する。全体方針として Rails → Go の段階的移行を進めており、新規エンドポイントを Rails に追加するのは方向性に逆行する。imgproxy 経由の設計を採用したため、Active Storage variant 機構への依存も切り離せる。

### Rails で Active Storage variant を eager 生成し、`attachments.og_variant_blob_key` カラムに保存する

Rails のアップロード処理で og variant を事前生成し、Go から参照する案を検討した。

**不採用の理由**: Rails 撤去を最終目標とする方針に対し、Active Storage variant 機構への依存を新たに増やすのは逆行する。imgproxy を採用すれば元画像のみを保持していればよく、variant の事前生成・バックフィル・カラム追加が一切不要になる。variant のサイズ・フォーマット変更時にも、URL パラメータの変更のみで済む。

### Go アプリ自身で libvips 等を使って画像変換を実装する

画像処理ライブラリの Go バインディングで variant 生成を Go アプリ自身に持ち込む案を検討した。

**不採用の理由**: アプリプロセスで画像処理を行うとリクエストレイテンシが伸びる。imgproxy のような専用サービスのほうが適切で、CDN との相性も良く、画像処理のスケールアウトを別軸で運用できる。Korylus 他プロジェクト (Annict) でも imgproxy を採用しており、運用ノウハウが共有できる。

## 将来検討する機能

### visibility 変更イベントによる CDN purge

「公開→非公開」のトピック visibility 変更や、ページ移動・削除・添付削除のタイミングで、対象 og:image URL を CDN キャッシュから明示的に purge する仕組み。`s-maxage=300` の leak window で許容できる範囲とみなして初期ロールアウトでは実装を見送っているが、運用観察で問題化した場合に追加実装するか、あるいは `s-maxage` を縮める形で対応するかを判断する。

### 本文中画像・サムネイルの imgproxy 化

現状、imgproxy 経由の配信は og:image のみ。本文中画像 (`/attachments/:id`) のダウンロードや、ページ一覧のサムネイル表示は引き続き Rails が Active Storage の signed URL / variant で配信している。将来別タスクで段階的に imgproxy へ統合する想定で、本パッケージ (`internal/image`) の `Helper` は og:image 以外の用途にも転用可能なように設計してある。

## 参考資料

- [imgproxy 公式ドキュメント](https://docs.imgproxy.net/)
- [@docs/system/feature-flag/overview.md](/workspace/docs/system/feature-flag/overview.md) — リバースプロキシの URL ルーティング判定の仕組み
- [@docs/private/plans/3_done/202605/og-image-expired-url-fix.md](/workspace/docs/private/plans/3_done/202605/og-image-expired-url-fix.md) — 本仕様の元になった作業計画書 (採用しなかった方針の詳細経緯を含む)
- [@docs/private/plans/2_todo/page-ogp-meta.md](/workspace/docs/private/plans/2_todo/page-ogp-meta.md) — OGP メタタグ全体の Go 移行 (本仕様の og:image URL を埋め込む側の実装)
