import { DirectUpload } from "@rails/activestorage";

// ファイルタイプごとのサイズ制限（バイト）
const FILE_SIZE_LIMITS = {
  image: 10 * 1024 * 1024, // 10MB
  video: 100 * 1024 * 1024, // 100MB
  other: 25 * 1024 * 1024, // 25MB
};

// 許可されるファイル形式
const ALLOWED_FILE_TYPES = {
  image: ["image/jpeg", "image/jpg", "image/png", "image/gif", "image/svg+xml"],
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
  ],
};

// アップロードの進捗イベント
export interface UploadProgress {
  loaded: number;
  total: number;
  percentage: number;
}

// アップロードエラー
export class UploadError extends Error {
  constructor(
    message: string,
    public code?: string,
  ) {
    super(message);
    this.name = "UploadError";
  }
}

// DirectUploadラッパークラス
export class DirectUploadWrapper {
  private file: File;
  private directUploadUrl: string;
  private onProgress?: (progress: UploadProgress) => void;
  private activeStorageUpload?: DirectUpload;

  constructor(file: File, directUploadUrl: string, onProgress?: (progress: UploadProgress) => void) {
    this.file = file;
    this.directUploadUrl = directUploadUrl;
    this.onProgress = onProgress;
  }

  // ファイルタイプの判定
  private getFileCategory(): "image" | "video" | "other" {
    if (ALLOWED_FILE_TYPES.image.includes(this.file.type)) {
      return "image";
    }
    if (ALLOWED_FILE_TYPES.video.includes(this.file.type)) {
      return "video";
    }
    return "other";
  }

  // ファイルサイズの検証
  private validateFileSize(): void {
    const category = this.getFileCategory();
    const limit = FILE_SIZE_LIMITS[category];

    if (this.file.size > limit) {
      const limitMB = Math.round(limit / (1024 * 1024));
      throw new UploadError(`ファイルサイズが制限（${limitMB}MB）を超えています`, "FILE_TOO_LARGE");
    }
  }

  // ファイル形式の検証
  private validateFileType(): void {
    const allAllowedTypes = [...ALLOWED_FILE_TYPES.image, ...ALLOWED_FILE_TYPES.video, ...ALLOWED_FILE_TYPES.document];

    if (!allAllowedTypes.includes(this.file.type)) {
      throw new UploadError("このファイル形式はアップロードできません", "INVALID_FILE_TYPE");
    }
  }

  // アップロードの実行
  async upload(): Promise<{ id: string; url: string }> {
    // バリデーション
    this.validateFileSize();
    this.validateFileType();

    return new Promise((resolve, reject) => {
      this.activeStorageUpload = new DirectUpload(this.file, this.directUploadUrl);

      // Active Storageのダイレクトアップロードコールバック
      const directUploadWillStoreFileWithXHR = (request: XMLHttpRequest) => {
        // 進捗イベントの設定
        request.upload.addEventListener("progress", (event) => {
          if (event.lengthComputable && this.onProgress) {
            this.onProgress({
              loaded: event.loaded,
              total: event.total,
              percentage: Math.round((event.loaded / event.total) * 100),
            });
          }
        });
      };

      // アップロードの実行
      this.activeStorageUpload.create((error, blob) => {
        if (error) {
          console.error("Upload error:", error);
          reject(new UploadError("ファイルのアップロードに失敗しました", "UPLOAD_FAILED"));
        } else {
          // Active Storageのblobオブジェクトから必要な情報を取得
          resolve({
            id: blob.signed_id,
            url: blob.url || "", // URLが利用可能な場合
          });
        }
      }, directUploadWillStoreFileWithXHR);
    });
  }

  // アップロードのキャンセル
  cancel(): void {
    // Active Storageの DirectUpload にはキャンセルメソッドがないため、
    // 実装上の制限となります
    console.warn("DirectUpload does not support cancellation");
  }
}

// 既存のコードとの互換性のためのエクスポート
export { DirectUploadWrapper as DirectUpload };