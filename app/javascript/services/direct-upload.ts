// Active StorageのDirectUploadは使用せず、独自実装を使用

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

// DirectUploadクラス（独自実装）
export class DirectUploadWrapper {
  private file: File;
  private directUploadUrl: string;
  private directUploadHeaders: Record<string, string>;
  private onProgress?: (progress: UploadProgress) => void;
  private xhr?: XMLHttpRequest;

  constructor(
    file: File,
    directUploadUrl: string,
    directUploadHeaders: Record<string, string> = {},
    onProgress?: (progress: UploadProgress) => void,
  ) {
    this.file = file;
    this.directUploadUrl = directUploadUrl;
    this.directUploadHeaders = directUploadHeaders;
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
      this.xhr = new XMLHttpRequest();
      this.xhr.open("PUT", this.directUploadUrl, true);
      this.xhr.responseType = "text";

      // Active Storageが生成したヘッダーを設定
      for (const [key, value] of Object.entries(this.directUploadHeaders)) {
        this.xhr.setRequestHeader(key, value);
      }

      // 進捗イベントの設定
      if (this.onProgress) {
        this.xhr.upload.addEventListener("progress", (event) => {
          if (event.lengthComputable) {
            this.onProgress!({
              loaded: event.loaded,
              total: event.total,
              percentage: Math.round((event.loaded / event.total) * 100),
            });
          }
        });
      }

      // 成功時のハンドリング
      this.xhr.addEventListener("load", () => {
        if (this.xhr!.status >= 200 && this.xhr!.status < 300) {
          // アップロード成功
          // URLはpresignレスポンスから取得済みなので、ここではダミーのIDを返す
          resolve({
            id: "uploaded",
            url: this.directUploadUrl.split("?")[0], // クエリパラメータを除いたURL
          });
        } else {
          reject(new UploadError(`アップロードに失敗しました (ステータス: ${this.xhr!.status})`, "UPLOAD_FAILED"));
        }
      });

      // エラー時のハンドリング
      this.xhr.addEventListener("error", () => {
        reject(new UploadError("ネットワークエラーが発生しました", "NETWORK_ERROR"));
      });

      // ファイルを送信
      this.xhr.send(this.file);
    });
  }

  // アップロードのキャンセル
  cancel(): void {
    if (this.xhr) {
      this.xhr.abort();
    }
  }
}

// 既存のコードとの互換性のためのエクスポート
export { DirectUploadWrapper as DirectUpload };
