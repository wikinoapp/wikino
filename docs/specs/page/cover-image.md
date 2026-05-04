# カバー画像 仕様書

## 概要

カバー画像は、ページ本文の 1 行目に貼られた画像を「ページを代表する 1 枚」として自動的に記録する機能である。ユーザーは特別な操作をすることなく、本文の 1 行目に画像を貼るだけでカバー画像が設定され、そのページの og:image（SNS シェア時のプレビュー画像）やページカード（Wiki リンクのリッチプレビューや一覧表示）の表示に利用される。

ページ本体（`pages`）だけでなく、下書き（`draft_pages`）と編集提案（`suggestion_pages`）にも同じ仕組みでカバー画像が記録される。下書き保存・公開・編集提案の反映いずれのタイミングでも、本文 1 行目を再スキャンしてカバー画像 ID を更新する。

**目的**:

- ページに「顔」を持たせ、SNS シェアやリンクカードでの視認性を高める
- ユーザーがカバー画像のための追加 UI を覚える必要をなくす（本文の 1 行目に画像を置けば自動で決まる）

**背景**:

- Markdown のページ本文では「重要な画像は冒頭に置く」という慣習が強く、1 行目の画像をそのままカバーとして使う規約が直感的かつ運用負荷が低い
- 1 行目に明示的な画像が無いページにはカバー画像を設定しないことで、内容と関係のない画像が SNS プレビューに使われる事故を避けられる
- og:image 配信は Go 版で公開判定を統合した専用エンドポイント (`/attachments/:id/og_image`) で行い、SNS クローラーのキャッシュ寿命を超えても URL が無効化されないように設計されている（詳細は本仕様書の「og:image 配信」を参照）

## 仕様

### カバー画像の自動抽出

- システムは下書き保存・ページ公開・編集提案の作成/反映時に、本文 1 行目から画像 ID を抽出してカバー画像として記録する
- 抽出対象は本文 1 行目（先頭から最初の `\n` まで、前後の空白は無視）のみ。2 行目以降の画像はカバー画像として扱われない
- 1 行目に画像が無い場合、または抽出した画像 ID が現在のスペース内の有効な添付ファイルでない場合はカバー画像 ID を `NULL` にする
- 抽出は以下の 2 形式に対応する。Markdown 画像形式が優先され、両方マッチする場合は Markdown 形式の ID が採用される
  - **Markdown 画像形式**: `![alt](/attachments/{id})` または `![alt](/attachments/{id} "title")`
  - **HTML img 要素**: `<img src="/attachments/{id}" ...>`（大文字小文字不問）

- 1 行目が通常の Markdown リンク（`[text](/attachments/{id})` のように `!` の付かないもの）であってもカバー画像とは判定しない

### スペーススコープ

- カバー画像として登録できるのは、ページが属するスペース内に存在する添付ファイルのみである
- 抽出した ID が他スペースの添付ファイル ID であった場合や、削除済みである場合はカバー画像を設定しない（`NULL` のまま記録される）

### og:image 配信

- システムは各ページの `<head>` に、カバー画像が存在するページについてのみ `og:image` メタタグを出力する
- og:image の URL は `/attachments/{attachment_id}/og_image` という「再検証付き永続 URL」として組み立てる。SNS クローラーがキャッシュした URL の寿命が切れても URL 文字列自体は無効化されないため、リッチプレビューが破綻しない
- `/attachments/{attachment_id}/og_image` へのリクエストは、その添付ファイルが「公開トピック内のページから参照されている」場合のみ画像を返す。非公開トピックのページからしか参照されていない場合や、どのページからも参照されていない場合は 404 を返す
- 「公開でない添付」と「存在しない添付」はレスポンス上区別せず一律 404 を返す（添付の存在自体を秘匿するため）
- カバー画像が GIF の場合は og:image を出力しない（GIF は SNS プレビューでアニメーションが再生されないなど扱いが特殊なため、デフォルトの OGP 画像にフォールバックする）

### カード画像

- ページカード（Wiki リンクのリッチプレビュー、ページ一覧画面など）にカバー画像が存在する場合、その画像をカード内のサムネイルとして表示する
- カード用画像の URL は短時間（1 時間）で失効する署名付き URL として生成する。HTML 自体のキャッシュ寿命と同等の短さで十分なため、og:image のような再検証付き永続 URL は使用しない
- カバー画像が GIF の場合は、サムネイル変換ではなくオリジナル画像の署名付き URL を返す（GIF をリサイズすると静止画化されるため）

### 下書き・編集提案でのカバー画像

- 下書きと編集提案にも、それぞれ独立したカバー画像 ID を保持する
- 下書きが自動保存・手動保存（下書き保存）されるたびに、その時点の本文 1 行目から抽出してカバー画像 ID を更新する
- 編集提案を作成すると、元になった下書きのカバー画像 ID がそのまま編集提案にコピーされる
- 編集提案を反映するとき、編集提案のカバー画像 ID がページ本体に反映される

## 設計

### データベース設計

カバー画像 ID は以下の 3 テーブルに `featured_image_attachment_id` カラムとして保存されている（カラム名は歴史的経緯で `featured` を使用しているが、機能名としては「カバー画像」で統一する）。

| テーブル           | カラム                         | 用途                       |
| ------------------ | ------------------------------ | -------------------------- |
| `pages`            | `featured_image_attachment_id` | 公開済みページのカバー画像 |
| `draft_pages`      | `featured_image_attachment_id` | 下書きのカバー画像         |
| `suggestion_pages` | `featured_image_attachment_id` | 編集提案のカバー画像       |

すべて `attachments(id)` への外部キー制約 (`ON DELETE` 指定なし) を持ち、`uuid` 型で nullable。`pages.featured_image_attachment_id` には B-tree インデックスが張られている。

### 抽出ロジック

本文 1 行目から添付ファイル ID を抽出するロジックは `internal/markup/attachment_extract.go` の `ExtractFeaturedImageID` に集約されている。本文 1 行目（前後の空白を除いたもの）に対して以下の順で正規表現を適用する。

1. Markdown 画像形式: `!\[[^\]]*\]\(/attachments/([^\s/)]+)(\s[^)]*)?\)`
2. HTML img 要素: `(?i)<img[^>]+src=["']/attachments/([^/"']+)["'][^>]*>`

抽出後の存在確認は `internal/usecase/attachment_ref.go` の `extractFeaturedImageAttachmentID` で行う。`AttachmentRepository.ExistsByIDAndSpace` を使い、現在のスペース内に該当 ID の添付ファイルが存在する場合のみ `*model.AttachmentID` を返し、それ以外は `nil` を返す。

### 抽出タイミング

| イベント                 | 抽出元の本文           | 反映先                                                                     |
| ------------------------ | ---------------------- | -------------------------------------------------------------------------- |
| 下書き自動保存・手動保存 | 下書きの本文           | `draft_pages.featured_image_attachment_id`                                 |
| ページ公開（PATCH）      | フォーム送信された本文 | `pages.featured_image_attachment_id`                                       |
| 編集提案作成             | 元になった下書きの本文 | `suggestion_pages.featured_image_attachment_id` (`draft_pages` からコピー) |
| 編集提案反映             | 編集提案の本文         | `pages.featured_image_attachment_id` (`suggestion_pages` から反映)         |

下書きでの抽出は `usecase.calculateDraftPageSaveData` に集約されており、Markdown レンダリング・Wiki リンクスキャン・カバー画像抽出を 1 箇所で実行する。ページ公開時は `PublishPageUsecase.Execute` 内で改めて本文 1 行目から抽出を行う（フォームの本文は下書きと別経路で送られてくるため）。

### og:image 配信エンドポイント

```
GET /attachments/{attachment_id}/og_image
```

- ハンドラー: `internal/handler/attachment_og_image.Handler.Show`
- UseCase: `GetAttachmentOgImageUsecase`
- Repository: `AttachmentRepository.FindPubliclyReferencedBlobByID` が「公開トピックのページから参照されている添付ファイル」のみ blob 情報を返す。公開判定が SQL に統合されており、呼び出し側で検証を忘れる構造的事故が起きない設計になっている
- 画像本体の配信は `image.OgImageBuilder` が組み立てる imgproxy 署名付き URL へリダイレクトすることで行う
- リバースプロキシミドルウェアは `^/attachments/[^/]+/og_image$` を Go 版で常に処理するパスとしてホワイトリスト登録している（`/attachments/:id` 自体は Rails 版のダウンロード URL なのでパスを限定している）

### カード画像 URL の生成

カード画像の URL 生成は現状 Rails 版の `PageRecord#card_image_url` が担当する（`featured_image_is_gif?` で GIF を判定し、GIF ならオリジナル URL、それ以外なら `attachment.thumbnail_url(size: AttachmentThumbnailSize::Card, expires_in: 1.hour)` を返す）。Go 版でページ表示画面を実装する際に同等のロジックを Go 側に移植する。

### ファイル構成

```
go/
├── internal/
│   ├── markup/
│   │   ├── attachment_extract.go             # ExtractFeaturedImageID（1行目から画像ID抽出）
│   │   └── attachment_extract_test.go
│   ├── usecase/
│   │   ├── attachment_ref.go                 # extractFeaturedImageAttachmentID（存在確認込み）
│   │   ├── draft_page_content.go             # 下書き保存時の抽出フロー
│   │   ├── publish_page.go                   # ページ公開時の抽出フロー
│   │   ├── create_suggestion.go              # 編集提案作成時のコピー
│   │   ├── apply_suggestion.go               # 編集提案反映時の反映
│   │   ├── get_attachment_og_image.go        # og:image配信用UseCase
│   │   └── get_attachment_og_image_test.go
│   ├── repository/
│   │   └── attachment.go                     # ExistsByIDAndSpace, FindPubliclyReferencedBlobByID
│   ├── handler/
│   │   └── attachment_og_image/
│   │       ├── handler.go
│   │       └── show.go                       # GET /attachments/:id/og_image
│   ├── image/
│   │   ├── og_image.go                       # OgImageBuilder（imgproxy署名URL生成）
│   │   └── og_image_test.go
│   └── middleware/
│       └── reverse_proxy.go                  # /attachments/:id/og_image を Go 版で処理
└── db/
    ├── migrations/
    │   ├── 20260319072803_add_linked_page_ids_and_featured_image_to_suggestion_pages.sql
    │   ├── 20260319074138_add_featured_image_attachment_id_to_draft_pages.sql
    │   └── 20260319082723_add_featured_image_fk_to_suggestion_pages_and_draft_pages.sql
    └── queries/
        ├── pages.sql                         # featured_image_attachment_id を含む UPDATE
        ├── draft_pages.sql                   # featured_image_attachment_id を含む UPSERT
        └── suggestion_pages.sql              # featured_image_attachment_id を含む INSERT/UPDATE
```

## 採用しなかった方針

### 「アイキャッチ画像」「featured image」という名称

WordPress 由来の「Featured Image / アイキャッチ画像」という呼称をそのまま採用する案を検討した。実際、DB のカラム名 (`featured_image_attachment_id`) や Go 内部のメソッド名 (`ExtractFeaturedImageID` 等) には現状この呼称が残っている。

**不採用の理由**:

- 「アイキャッチ」は編集者がブログ記事に手で設定する画像、というニュアンスが強い。Wikino の本機能は「本文 1 行目に画像があれば自動でそれを採用する」自動的な挙動であり、編集者が "特集する" という意図とは合わない
- og:image とページカードの双方で使われる「ページの代表画像」という機能を中立に表すには `cover_image`（カバー画像）の方が直感的で、Notion・Scrapbox など同種のツールでも採用されている呼称である

DB カラム名と Go 内部の識別子の `featured_image_*` から `cover_image_*` への置き換えは、運用への影響範囲が大きいため別タスクで段階的に行う。

### ユーザーが明示的にカバー画像を選択する UI

カバー画像専用のアップロード UI（編集画面の専用フォーム）を提供し、ユーザーが本文とは独立にカバー画像を選択できるようにする案を検討した。

**不採用の理由**:

- 専用 UI を追加すると、編集者が覚えるべき操作が増える
- 「本文 1 行目に画像を置く」運用は Markdown 編集の自然な流れに沿っており、追加 UI なしでも自然にカバー画像が付けられる
- 将来、自動抽出に加えて明示的な選択も可能にしたくなった時点で UI を追加する余地は残せる

### 本文中の任意の位置にある画像をカバー画像として扱う

本文 1 行目に限定せず、本文中で最初に登場する画像をカバー画像とする案を検討した。

**不採用の理由**:

- 本文中盤・末尾の挿絵が意図せずカバーになる事故が起きやすい
- 「カバー画像にしたければ 1 行目に置く / したくなければ置かない」という規約のほうが、編集者にとって挙動が予測しやすい

### og:image を imgproxy 署名 URL 直リンクで返す

og:image にも `card_image_url` と同じく imgproxy 署名付き URL を直接埋め込む案を検討した。

**不採用の理由**:

- 署名付き URL には有効期限が必要で、SNS クローラーが取得した HTML をキャッシュしている期間中に URL が失効すると、SNS 上のリッチプレビューが壊れる
- 一方で「期限なしの署名 URL」は流出リスクが大きい
- `/attachments/:id/og_image` という再検証付き永続 URL を返し、アクセスごとに公開判定と署名 URL 生成を行う方式であれば、URL 文字列を不変にしつつ非公開化された画像も即座にブロックできる

### og:image でも GIF をそのまま配信する

GIF を含む全形式のカバー画像をそのまま og:image として配信する案を検討した。

**不採用の理由**:

- 多くの SNS プレビューでは GIF アニメーションが再生されず最初のフレームが切り出される、もしくはサイズや形式の互換性で表示が崩れるなど挙動が一定でない
- GIF はカード画像（自サイト内の表示）ではアニメーション付きで楽しめるよう原寸を返す一方、外部 SNS への露出となる og:image では出力しない、という非対称な扱いにすることで、最も安定した SNS プレビュー体験（デフォルト OGP 画像へのフォールバック）を提供する

## 参考資料

- [og:image 配信実装](/workspace/go/internal/handler/attachment_og_image) — Go 版の og:image 配信エンドポイント
- [カバー画像抽出ロジック](/workspace/go/internal/markup/attachment_extract.go) — 本文 1 行目からの画像 ID 抽出
- [PageRecord#card_image_url](/workspace/rails/app/records/page_record.rb) — Rails 版のカード画像 URL 生成
- [ページ編集 仕様書](edit.md) — 下書き保存・公開フローの全体像
