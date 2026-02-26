# ガイドライン違反修正 仕様書

<!--
このファイルは /workspace/docs/specs/template.md からコピーして作成しました。
-->

## 概要

Go版Wikinoプロジェクトにおいて、CLAUDE.mdおよびgo/CLAUDE.mdに記載されているガイドラインに違反している箇所を修正します。

**目的**:

- コーディングガイドラインに完全準拠し、コードの一貫性と品質を維持する
- 将来の開発者が迷わずに開発できるようにする

**背景**:

- プロジェクト全体のコードレビューを実施し、ガイドライン違反を洗い出した

## 要件

### 機能要件

- templテンプレートの引数パターンを構造体ベースに統一する

### 非機能要件

- **保守性**: ドキュメントと実装の一貫性を確保

## 設計

### 発見された違反

#### 1. templテンプレート引数パターン違反

**ガイドライン（go/CLAUDE.md より）**:

> テンプレート関数の引数は**構造体ベースのパターン**を使用します。
>
> - ✅ **構造体を使用**: テンプレートに渡すデータは専用の構造体にまとめる
> - ❌ **`context.Context` を明示的に渡さない**: templ は `ctx` を暗黙的に提供するため不要
> - ❌ **複数の引数を個別に渡さない**: 引数が増えるたびにシグネチャ変更が必要になる

**違反箇所**:

| ファイル                                                                       | 現在の引数パターン                                                                               |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| `/workspace/go/internal/templates/pages/sign_in/new.templ`                     | `ctx context.Context, formErrors *session.FormErrors, csrfToken string, turnstileSiteKey string` |
| `/workspace/go/internal/templates/pages/sign_in_two_factor/new.templ`          | `ctx context.Context, formErrors *session.FormErrors, csrfToken string`                          |
| `/workspace/go/internal/templates/pages/sign_in_two_factor/recovery_new.templ` | `ctx context.Context, formErrors *session.FormErrors, csrfToken string`                          |

**理由**:

- `context.Context`を明示的に渡しているが、templは`ctx`を暗黙的に提供するため不要
- 複数の引数を個別に渡しており、拡張性・可読性が低い
- 構造体ベースのパターンに統一することで、Goの慣習に従い、メンテナンス性が向上する

### 確認済み（違反なし）

以下の項目については、ガイドラインに準拠していることを確認しました：

1. **アーキテクチャ（3層アーキテクチャ）**
   - Handler/UseCaseはQueryに直接依存せず、Repositoryを経由している ✓
   - Domain/Infrastructure層はPresentation層に依存していない ✓
   - TemplatesはModelに直接依存せず、ViewModelを経由している ✓

2. **ログ出力規約**
   - すべてのログ出力で`log/slog`パッケージを使用 ✓
   - `log.Fatalf`ではなく`slog.Error` + `os.Exit(1)`を使用 ✓

3. **ハンドラーガイドライン**
   - すべてのハンドラーファイル名が標準8種類（handler.go, index.go, show.go, new.go, create.go, edit.go, update.go, delete.go）に準拠 ✓
   - メソッド名がファイル名に完全対応 ✓

4. **バリデーション/Request DTO**
   - リクエスト構造体は`request.go`に配置 ✓
   - Request DTOは形式バリデーションのみ実施（DBアクセスなし） ✓

5. **国際化（I18n）**
   - ユーザー向けメッセージはすべて`i18n.T()`で国際化 ✓

6. **depguard設定**
   - `.golangci.yml`でアーキテクチャルールが適切に強制されている ✓

### 影響範囲

以下のファイルを更新する必要があります：

#### templテンプレート引数パターン

1. **テンプレートファイル**
   - `/workspace/go/internal/templates/pages/sign_in/new.templ` - 引数を構造体ベースに変更
   - `/workspace/go/internal/templates/pages/sign_in_two_factor/new.templ` - 引数を構造体ベースに変更
   - `/workspace/go/internal/templates/pages/sign_in_two_factor/recovery_new.templ` - 引数を構造体ベースに変更

2. **ハンドラーファイル**
   - `/workspace/go/internal/handler/sign_in/new.go` - テンプレート呼び出しを更新
   - `/workspace/go/internal/handler/sign_in_two_factor/new.go` - テンプレート呼び出しを更新
   - `/workspace/go/internal/handler/sign_in_two_factor/recovery_new.go`（存在する場合）- テンプレート呼び出しを更新

## タスクリスト

### フェーズ 1: templテンプレート引数パターンの修正

- [x] **1-1**: sign_in/new.templ を構造体ベースに変更
  - `NewPageData`構造体を定義（CSRFToken, TurnstileSiteKey, FormErrors, Email）
  - テンプレート関数のシグネチャを `New(data NewPageData)` に変更
  - `context.Context`引数を削除（templが暗黙的に提供）
  - テンプレート内の変数参照を`data.XXX`形式に更新
  - ハンドラー側の呼び出しを更新
  - **想定ファイル数**: 約 2 ファイル（テンプレート 1 + ハンドラー 1）
  - **想定行数**: 約 20 行

- [x] **1-2**: sign_in_two_factor/new.templ を構造体ベースに変更
  - `NewPageData`構造体を定義（CSRFToken, FormErrors）
  - テンプレート関数のシグネチャを `New(data NewPageData)` に変更
  - `context.Context`引数を削除
  - テンプレート内の変数参照を`data.XXX`形式に更新
  - ハンドラー側の呼び出しを更新
  - **想定ファイル数**: 約 2 ファイル（テンプレート 1 + ハンドラー 1）
  - **想定行数**: 約 15 行

- [x] **1-3**: sign_in_two_factor/recovery_new.templ を構造体ベースに変更
  - `RecoveryNewPageData`構造体を定義（CSRFToken, FormErrors）
  - テンプレート関数のシグネチャを `RecoveryNew(data RecoveryNewPageData)` に変更
  - `context.Context`引数を削除
  - テンプレート内の変数参照を`data.XXX`形式に更新
  - ハンドラー側の呼び出しを更新（該当する場合）
  - **想定ファイル数**: 約 2 ファイル（テンプレート 1 + ハンドラー 1）
  - **想定行数**: 約 15 行

### フェーズ 2: 検証

- [ ] **2-1**: ローカル環境での動作確認
  - ログインページ（`/sign_in`）が正常に表示されることを確認
  - 2FAページ（`/sign_in/two_factor/new`）が正常に表示されることを確認
  - **想定ファイル数**: 0 ファイル
  - **想定行数**: 0 行

## 参考資料

- [go/CLAUDE.md - 環境変数の命名規則](/workspace/go/CLAUDE.md)
- [CLAUDE.md - プロジェクト全体のガイド](/workspace/CLAUDE.md)
