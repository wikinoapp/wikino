import { EditorView } from "codemirror";
import { DirectUpload, UploadError, UploadProgress } from "./direct-upload";
import { calculateFileChecksum } from "../utils/file-checksum";
import {
  insertUploadPlaceholder,
  replacePlaceholderWithUrl,
  removePlaceholder,
} from "../markdown-editor/upload-placeholder";

interface PresignResponse {
  directUploadUrl: string;
  directUploadHeaders: Record<string, string>;
  blobSignedId: string;
}

export class FileUploadHandler {
  private editorView: EditorView;
  private spaceIdentifier: string;

  constructor(editorView: EditorView, spaceIdentifier: string) {
    this.editorView = editorView;
    this.spaceIdentifier = spaceIdentifier;
  }

  async handleFileUpload(files: File[], position: number): Promise<void> {
    for (const file of files) {
      // アップロード中のプレースホルダーを挿入
      const placeholderId = insertUploadPlaceholder(this.editorView, file.name, position);

      try {
        // ファイルのMD5チェックサムを計算
        const checksum = await calculateFileChecksum(file);

        // プリサイン用URLを取得
        const presignData = await this.getPresignUrl(file, checksum);

        // ファイルをアップロード
        await this.uploadFile(file, presignData);

        // アップロード成功後、Active StorageのURLを生成
        const attachmentUrl = `/rails/active_storage/blobs/redirect/${presignData.blobSignedId}/${encodeURIComponent(file.name)}`;

        // 画像ファイルの場合、幅と高さを取得
        let width: number | undefined;
        let height: number | undefined;

        if (file.type.startsWith("image/")) {
          try {
            const dimensions = await this.getImageDimensions(file);
            width = dimensions.width;
            height = dimensions.height;
          } catch (error) {
            console.warn("画像サイズの取得に失敗しました:", error);
            // サイズ取得に失敗しても処理を続行
          }
        }

        // プレースホルダーをURLに置換
        replacePlaceholderWithUrl(this.editorView, placeholderId, attachmentUrl, file.name, width, height, file.type);
      } catch (error) {
        console.error("Upload error:", error);
        removePlaceholder(this.editorView, placeholderId);

        // エラーメッセージの表示
        if (error instanceof UploadError) {
          this.showErrorMessage(error.message);
        } else {
          this.showErrorMessage("ファイルのアップロードに失敗しました");
        }
      }
    }
  }

  private async getPresignUrl(file: File, checksum: string): Promise<PresignResponse> {
    const response = await fetch(`/s/${this.spaceIdentifier}/attachments/presign`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": document.querySelector('meta[name="csrf-token"]')?.getAttribute("content") || "",
      },
      body: JSON.stringify({
        filename: file.name,
        content_type: file.type,
        byte_size: file.size,
        checksum: checksum,
      }),
    });

    if (!response.ok) {
      throw new UploadError("プリサイン用URLの取得に失敗しました");
    }

    return response.json();
  }

  private async uploadFile(file: File, presignData: PresignResponse): Promise<void> {
    const uploader = new DirectUpload(
      file,
      presignData.directUploadUrl,
      presignData.directUploadHeaders,
      (progress: UploadProgress) => {
        // 進捗状況のログ（必要に応じてUIに反映）
        console.log(`Uploading ${file.name}: ${progress.percentage}%`);
      },
    );

    await uploader.upload();
  }

  private showErrorMessage(message: string): void {
    // エラーメッセージを表示（一時的な実装）
    // TODO: 適切なエラー通知システムに置き換える
    const notification = document.createElement("div");
    notification.textContent = message;
    notification.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      background: #ef4444;
      color: white;
      padding: 12px 24px;
      border-radius: 6px;
      box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
      z-index: 10000;
    `;

    document.body.appendChild(notification);

    setTimeout(() => {
      notification.remove();
    }, 5000);
  }

  private async getImageDimensions(file: File): Promise<{ width: number; height: number }> {
    return new Promise((resolve, reject) => {
      const img = new Image();
      const url = URL.createObjectURL(file);

      img.onload = () => {
        URL.revokeObjectURL(url);
        resolve({
          width: img.naturalWidth,
          height: img.naturalHeight,
        });
      };

      img.onerror = () => {
        URL.revokeObjectURL(url);
        reject(new Error("画像の読み込みに失敗しました"));
      };

      img.src = url;
    });
  }
}
