# コードレビュー: go-page-edit-fix

## レビュー情報

| 項目                       | 内容                                       |
| -------------------------- | ------------------------------------------ |
| レビュー日                 | 2026-03-07                                 |
| 対象ブランチ               | go-page-edit-fix                           |
| ベースブランチ             | go-page-edit                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-rollout.md |
| 変更ファイル数             | 8 ファイル                                 |
| 変更行数（実装）           | +103 / -14 行                              |
| 変更行数（テスト）         | +247 / -17 行                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/markup/attachment_extract.go`
- [x] `go/internal/markup/attachment_filter.go`
- [x] `go/internal/markup/markup.go`
- [x] `go/internal/repository/attachment.go`

### テストファイル

- [x] `go/internal/markup/attachment_extract_test.go`
- [x] `go/internal/markup/attachment_filter_test.go`
- [x] `go/internal/markup/markup_test.go`
- [x] `go/internal/markup/pipeline_integration_test.go`

### 設定・その他

（なし）

## ファイルごとのレビュー結果

全ファイルについて問題は検出されませんでした。以下に各ファイルの確認結果を記載します。

### レビュー対象のコミット（5件）

| コミット | 内容                                                      |
| -------- | --------------------------------------------------------- |
| ba6834f5 | Markdownタイトル付き添付ファイルURLのID抽出バグを修正     |
| ec4e8e13 | 画像キャプション表示機能を実装                            |
| e8ae4d86 | CRLF改行コードによるimg要素キャプション変換の不具合を修正 |
| 6187db90 | 添付ファイルIDの不正な形式によるDBエラーを修正            |
| eb352ef8 | GFMテーブルのalign属性出力を修正                          |

### 各ファイルの確認内容

**`go/internal/markup/attachment_extract.go`**:

- 正規表現がMarkdownのtitle付き形式（`![alt](/attachments/id "title")`）を正しく処理するよう更新されている
- `addID`関数でURLデコードとバックスラッシュ検証が追加されており、不正入力に対する防御が適切
- `net/url`パッケージのインポートが追加されている

**`go/internal/markup/attachment_filter.go`**:

- `extractAttachmentID`のパラメータ名が`url`から`rawURL`に変更され、`net/url`パッケージとの名前衝突を回避している（良い修正）
- `attachmentPathRegex`にバックスラッシュの除外パターン（`[^/\\]`）が追加されている
- URLデコード後のマッチングにより、bluemondayのパーセントエンコード（`%5C`）問題を正しく処理している
- `getImageCaptionSiblings`関数は空白テキストノードを正しくスキップし、`<br>` + `<em>`/`<strong>`のパターンを検出する
- `wrapInParagraphWithSiblings`関数は画像リンクとキャプションをまとめて`<p>`でラップする実装が明快
- セキュリティ: HTML DOM操作はテキストノードを使用しており、XSSリスクなし

**`go/internal/markup/markup.go`**:

- CRLF正規化（`\r\n` → `\n`）がMarkdown変換前に正しい位置で実行されている
- `standaloneImgRegex`が末尾の空白を許容するよう更新されている（`[ \t]*`）
- `zwnjBrRegex`が`<br />`と`<br/>`の両方の形式に対応するよう更新されている
- GFM拡張が個別指定に変更され、`extension.TableCellAlignAttribute`により`align`属性がCSSスタイルではなくHTML属性として出力される
- サニタイズポリシーに`td`/`th`の`align`属性が許可されており、GFMテーブルの配置指定が正しく動作する

**`go/internal/repository/attachment.go`**:

- UUID形式バリデーション（`uuidRegex`）がDB問い合わせ前に追加されており、不正なIDによるDBエラーを防止している
- `FindByIDsAndSpace`で非UUID形式のIDをフィルタリングし、有効なIDがない場合は早期リターンしている
- `ExistsByIDAndSpace`と`FindByIDAndSpace`にも同様のバリデーションが追加されている
- 正規表現は小文字UUIDのみマッチするが、PostgreSQLが小文字で生成するため問題なし

## 設計との整合性チェック

作業計画書にはフェーズ0のバグ修正として「下書きリビジョン削除」タスク（0-1）のみが記載されているが、本ブランチの修正はページ編集のGo版展開中に発見された追加のバグ修正である。作業計画書のスコープ外だが、Go版の全ユーザー展開の品質を確保するために必要な修正であり、方針に矛盾はない。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

5つのバグ修正がそれぞれ適切に実装されている。各修正に対応するテストが十分に追加されており、正常系・異常系をカバーしている。コーディング規約・アーキテクチャガイドラインに準拠しており、セキュリティ面でも問題はない。特に以下の点が良い：

- UUID形式バリデーションによるDBエラーの防止（防御的プログラミング）
- パーセントエンコードされたバックスラッシュの処理（bluemondayとの統合問題への対応）
- GFMテーブルの`align`属性出力方式の変更（`extension.TableCellAlignAttribute`の使用）
- CRLF正規化の適切な位置での実行
- キャプション検出ロジックの空白テキストノード考慮
