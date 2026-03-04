# コードレビュー: page-edit-8-4

## レビュー情報

| 項目                       | 内容                                                       |
| -------------------------- | ---------------------------------------------------------- |
| レビュー日                 | 2026-02-27                                                 |
| 対象ブランチ               | page-edit-8-4                                              |
| ベースブランチ             | page-edit                                                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md（タスク 8-4） |
| 変更ファイル数             | 6 ファイル                                                 |
| 変更行数（実装）           | +208 / -1 行（3 ファイル）                                 |
| 変更行数（テスト）         | +459 / -0 行（2 ファイル）                                 |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド（JavaScript/TypeScript関連）
- [@CLAUDE.md#コメントのガイドライン](/workspace/CLAUDE.md) - コメントのガイドライン

## 変更ファイル一覧

### 実装ファイル

- [ ] `go/web/markdown-editor/file-drop-handler.ts`
- [x] `go/web/markdown-editor/paste-handler.ts`
- [x] `go/web/markdown-editor/markdown-editor.ts`

### テストファイル

- [x] `go/e2e/tests/file-upload.spec.ts`
- [x] `go/e2e/tests/paste.spec.ts`

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-migration.md`

## ファイルごとのレビュー結果

### `go/web/markdown-editor/file-drop-handler.ts`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md](/workspace/CLAUDE.md) - コーディング規約
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- **イベントリスナーのクリーンアップ**: `setupEventListeners()` でイベントリスナーを `this.handleDragEnter.bind(this)` で登録しているが、`bind()` は毎回新しい関数参照を生成するため、`destroy()` で `removeEventListener` による解除ができない。現在の `destroy()` は `hideDropZone()` のみ呼んでいる。

  ```typescript
  // 問題のあるコード (file-drop-handler.ts:14-21)
  setupEventListeners() {
    const dom = this.view.dom;
    dom.addEventListener("dragenter", this.handleDragEnter.bind(this));
    dom.addEventListener("dragleave", this.handleDragLeave.bind(this));
    dom.addEventListener("dragover", this.handleDragOver.bind(this));
    dom.addEventListener("drop", this.handleDrop.bind(this));
  }

  destroy() {
    this.hideDropZone();
  }
  ```

  **修正案**:

  ```typescript
  private boundHandlers: {
    dragenter: (e: DragEvent) => void;
    dragleave: (e: DragEvent) => void;
    dragover: (e: DragEvent) => void;
    drop: (e: DragEvent) => void;
  };

  constructor(view: EditorView) {
    this.view = view;
    this.boundHandlers = {
      dragenter: this.handleDragEnter.bind(this),
      dragleave: this.handleDragLeave.bind(this),
      dragover: this.handleDragOver.bind(this),
      drop: this.handleDrop.bind(this),
    };
    this.setupEventListeners();
  }

  setupEventListeners() {
    const dom = this.view.dom;
    dom.addEventListener("dragenter", this.boundHandlers.dragenter);
    dom.addEventListener("dragleave", this.boundHandlers.dragleave);
    dom.addEventListener("dragover", this.boundHandlers.dragover);
    dom.addEventListener("drop", this.boundHandlers.drop);
  }

  destroy() {
    this.hideDropZone();
    const dom = this.view.dom;
    dom.removeEventListener("dragenter", this.boundHandlers.dragenter);
    dom.removeEventListener("dragleave", this.boundHandlers.dragleave);
    dom.removeEventListener("dragover", this.boundHandlers.dragover);
    dom.removeEventListener("drop", this.boundHandlers.drop);
  }
  ```

  **補足**: CodeMirror の ViewPlugin が destroy されるとき、通常は DOM 要素自体も破棄されるためリスナーもガベージコレクションされる。そのため実害は小さいが、明示的なクリーンアップはベストプラクティスとして推奨される。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り、bound 関数を保持して destroy 時に解除する
  - [ ] 現状のままにする（CodeMirror の DOM ライフサイクルに依存）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/web/markdown-editor/paste-handler.ts`: ファイルタイプ判定の一元化

**ステータス**: 要確認

**現状**:

`paste-handler.ts` の `isAcceptedFileType()` と `file-upload-handler.ts` の `ALLOWED_FILE_TYPES` / `ALL_ALLOWED_TYPES` が異なるファイルタイプ判定ロジックを持っている。

`paste-handler.ts` のみが受け入れるタイプ:

- `application/x-rar-compressed`
- `application/x-7z-compressed`
- `image/*`（広範な一致。upload handler は jpeg/png/gif/svg+xml/webp のみ）
- `video/*`（広範な一致。upload handler は mp4/webm/quicktime のみ）
- `text/*`（広範な一致。upload handler は plain/csv/markdown のみ）
- `application/vnd.ms-*`（広範な一致。upload handler は ms-excel のみ）

ペーストハンドラーがファイルを受け入れ（`event.preventDefault()` で通常のペースト動作を抑止）→ アップロードハンドラーが拒否 → エラーメッセージ表示、という流れになる。通常のペースト動作がブロックされた上でエラーになるため、ユーザーにとって混乱する可能性がある。

**提案**:

`file-upload-handler.ts` の `ALL_ALLOWED_TYPES` を `paste-handler.ts` でも再利用する。

```typescript
// paste-handler.ts
import { ALL_ALLOWED_TYPES } from "./file-upload-handler";

function isAcceptedFileType(mimeType: string): boolean {
  return ALL_ALLOWED_TYPES.includes(mimeType);
}
```

ただし、現在 `ALL_ALLOWED_TYPES` は export されていないため、export の追加が必要。

**メリット**:

- ファイルタイプ判定ロジックの一元化（Single Source of Truth）
- ペーストハンドラーが受け入れないファイルは通常のペースト動作に委ねられる
- サポートされないファイルをペーストしたときの混乱がなくなる

**トレードオフ**:

- `paste-handler.ts` から `file-upload-handler.ts` への依存が増える
- 現状でも、ペーストハンドラーで受け入れた後にアップロードハンドラーでバリデーションされるため、実害は限定的（エラーメッセージは表示される）
- 広範な MIME タイプ判定（`image/*` など）を維持したい場合は、upload handler 側でも同様の広範な判定に変更する必要がある

**対応方針**:

- [x] 提案通り変更する（`ALL_ALLOWED_TYPES` を export して再利用）
- [ ] 現状のまま（ペーストハンドラーは緩やかなフィルタとして機能させる）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Comment

**総評**:

タスク 8-4（エディタのドラッグ&ドロップ・ペーストアップロードの追加 + E2Eテスト）の実装が作業計画書に従って適切に行われている。

**良い点**:

- 関心の分離が明確: `file-drop-handler.ts`（DOM イベント検出）、`paste-handler.ts`（クリップボードイベント検出）、`file-upload-handler.ts`（アップロードロジック）がカスタムイベントで疎結合
- CodeMirror の ViewPlugin パターンを正しく使用
- E2Eテストが充実しており、ドロップゾーンの表示/非表示、イベントディスパッチ、プレースホルダー挿入、エラーケース（不正なファイル形式、サイズ超過）、既存テキストの保持など幅広いケースをカバー
- `dragCounter` パターンによるネストされた drag enter/leave の正しいハンドリング

**改善が必要な点**:

- イベントリスナーのクリーンアップ（軽微）
- ファイルタイプ判定のペーストハンドラーとアップロードハンドラー間の不一致（設計改善の提案として記載）
