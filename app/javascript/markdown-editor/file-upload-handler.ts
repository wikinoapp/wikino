import { EditorView } from "codemirror";
import { DirectUpload, UploadError, UploadProgress } from "./direct-upload";
import { calculateFileChecksum } from "../utils/file-checksum";
import {
  insertUploadPlaceholder,
  replacePlaceholderWithUrl,
  removePlaceholder,
} from "./upload-placeholder";

interface PresignResponse {
  directUploadUrl: string;
  directUploadHeaders: Record<string, string>;
  blobSignedId: string;
  attachmentId: string;
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
        // ファイルの検証（サイズ、タイプ、画像の次元）
        await this.validateFile(file);

        // 画像ファイルの場合、事前に幅と高さを取得
        let width: number | undefined;
        let height: number | undefined;

        if (file.type.startsWith("image/")) {
          try {
            const dimensions = await this.getImageDimensions(file);
            width = dimensions.width;
            height = dimensions.height;
          } catch (error) {
            console.warn("画像サイズの取得に失敗しました:", error);
            // サイズ取得に失敗しても処理を続行（画像以外のファイルとして扱う）
          }
        }

        // ファイルのMD5チェックサムを計算
        const checksum = await calculateFileChecksum(file);

        // プリサイン用URLを取得
        const presignData = await this.getPresignUrl(file, checksum);

        // ファイルをアップロード
        await this.uploadFile(file, presignData);

        // アップロード成功後、添付ファイルのURLを生成
        const attachmentUrl = `/attachments/${presignData.attachmentId}`;

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
    // flash-toast-controllerを使ってエラーメッセージを表示
    const flashToastEvent = new CustomEvent("flash-toast:show", {
      detail: {
        type: "alert",
        messageHtml: message,
      },
    });
    document.dispatchEvent(flashToastEvent);
  }

  private async validateFile(file: File): Promise<void> {
    // ファイルサイズの検証
    const FILE_SIZE_LIMITS = {
      image: 10 * 1024 * 1024, // 10MB
      video: 100 * 1024 * 1024, // 100MB
      other: 25 * 1024 * 1024, // 25MB
    };

    // ファイルタイプの検証
    const ALLOWED_FILE_TYPES = {
      image: ["image/jpeg", "image/jpg", "image/png", "image/gif", "image/svg+xml", "image/webp"],
      video: ["video/mp4", "video/webm", "video/quicktime"],
      document: [
        "application/pdf",
        "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        "application/vnd.openxmlformats-officedocument.presentationml.presentation",
        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        "application/vnd.ms-excel",
        "text/plain",
        "text/csv",
        "text/x-log",
        "text/markdown",
        "application/json",
        "application/x-zip-compressed",
        "application/zip",
        "application/gzip",
        "application/x-gzip",
        "application/x-tar",
        "application/x-compressed-tar",
      ],
    };

    // ファイルタイプのチェック
    const allAllowedTypes = [...ALLOWED_FILE_TYPES.image, ...ALLOWED_FILE_TYPES.video, ...ALLOWED_FILE_TYPES.document];
    if (!allAllowedTypes.includes(file.type)) {
      throw new UploadError("このファイル形式はアップロードできません");
    }

    // ファイルサイズのチェック
    let category: "image" | "video" | "other";
    if (ALLOWED_FILE_TYPES.image.includes(file.type)) {
      category = "image";
    } else if (ALLOWED_FILE_TYPES.video.includes(file.type)) {
      category = "video";
    } else {
      category = "other";
    }

    const limit = FILE_SIZE_LIMITS[category];
    if (file.size > limit) {
      const limitMB = Math.round(limit / (1024 * 1024));
      throw new UploadError(`ファイルサイズが制限（${limitMB}MB）を超えています`);
    }

    // 画像の次元検証
    if (file.type.startsWith("image/")) {
      try {
        const dimensions = await this.getImageDimensions(file);
        const MAX_IMAGE_DIMENSION = 10000;
        if (dimensions.width > MAX_IMAGE_DIMENSION || dimensions.height > MAX_IMAGE_DIMENSION) {
          throw new UploadError(
            `画像のサイズが大きすぎます。${MAX_IMAGE_DIMENSION}×${MAX_IMAGE_DIMENSION}ピクセル以下にしてください`,
          );
        }
      } catch (error) {
        if (error instanceof UploadError) {
          throw error;
        }
        // 画像の次元取得に失敗した場合はログを出力して続行
        console.warn("画像の次元取得に失敗しました:", error);
      }
    }
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
