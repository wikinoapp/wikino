import { post } from "@rails/request.js";

// ファイルタイプごとのサイズ制限（バイト）
const FILE_SIZE_LIMITS = {
  image: 10 * 1024 * 1024,      // 10MB
  video: 100 * 1024 * 1024,     // 100MB
  other: 25 * 1024 * 1024       // 25MB
};

// 許可されるファイル形式
const ALLOWED_FILE_TYPES = {
  image: [
    "image/jpeg", "image/jpg", "image/png", "image/gif", "image/svg+xml"
  ],
  video: [
    "video/mp4", "video/webm", "video/quicktime"
  ],
  document: [
    "application/pdf",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    "application/vnd.ms-excel",
    "text/plain", "text/csv", "text/x-log", "text/markdown",
    "application/json", "application/x-zip-compressed", "application/zip",
    "application/gzip", "application/x-gzip", "application/x-tar"
  ]
};

// アップロードの進捗イベント
export interface UploadProgress {
  loaded: number;
  total: number;
  percentage: number;
}

// アップロードエラー
export class UploadError extends Error {
  constructor(message: string, public code?: string) {
    super(message);
    this.name = "UploadError";
  }
}

// DirectUploadクラス
export class DirectUpload {
  private file: File;
  private spaceIdentifier: string;
  private onProgress?: (progress: UploadProgress) => void;
  private xhr?: XMLHttpRequest;
  private retryCount = 0;
  private maxRetries = 3;
  private baseDelay = 1000; // 1秒

  constructor(
    file: File,
    spaceIdentifier: string,
    onProgress?: (progress: UploadProgress) => void
  ) {
    this.file = file;
    this.spaceIdentifier = spaceIdentifier;
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
      throw new UploadError(
        `ファイルサイズが制限（${limitMB}MB）を超えています`,
        "FILE_TOO_LARGE"
      );
    }
  }

  // ファイル形式の検証
  private validateFileType(): void {
    const allAllowedTypes = [
      ...ALLOWED_FILE_TYPES.image,
      ...ALLOWED_FILE_TYPES.video,
      ...ALLOWED_FILE_TYPES.document
    ];
    
    if (!allAllowedTypes.includes(this.file.type)) {
      throw new UploadError(
        "このファイル形式はアップロードできません",
        "INVALID_FILE_TYPE"
      );
    }
  }

  // プリサイン用URLの取得
  private async getPresignedUrl(): Promise<{
    upload_url: string;
    file_key: string;
    signed_id: string;
  }> {
    const response = await post(`/s/${this.spaceIdentifier}/attachments/presign`, {
      body: {
        filename: this.file.name,
        content_type: this.file.type,
        byte_size: this.file.size
      }
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new UploadError(
        errorData.error || "プリサイン用URLの取得に失敗しました",
        "PRESIGN_ERROR"
      );
    }

    return response.json();
  }

  // ファイルのアップロード
  private uploadFile(uploadUrl: string): Promise<void> {
    return new Promise((resolve, reject) => {
      this.xhr = new XMLHttpRequest();

      // 進捗イベント
      this.xhr.upload.addEventListener("progress", (event) => {
        if (event.lengthComputable && this.onProgress) {
          this.onProgress({
            loaded: event.loaded,
            total: event.total,
            percentage: Math.round((event.loaded / event.total) * 100)
          });
        }
      });

      // 完了イベント
      this.xhr.addEventListener("load", () => {
        if (this.xhr!.status >= 200 && this.xhr!.status < 300) {
          resolve();
        } else {
          reject(new UploadError(
            "ファイルのアップロードに失敗しました",
            "UPLOAD_FAILED"
          ));
        }
      });

      // エラーイベント
      this.xhr.addEventListener("error", () => {
        reject(new UploadError(
          "ネットワークエラーが発生しました",
          "NETWORK_ERROR"
        ));
      });

      // タイムアウトイベント
      this.xhr.addEventListener("timeout", () => {
        reject(new UploadError(
          "アップロードがタイムアウトしました",
          "TIMEOUT"
        ));
      });

      // アップロードの実行
      this.xhr.open("PUT", uploadUrl);
      this.xhr.timeout = 300000; // 5分
      this.xhr.send(this.file);
    });
  }

  // アップロード完了の通知
  private async notifyUploadComplete(fileKey: string, signedId: string): Promise<{
    id: string;
    url: string;
  }> {
    const response = await post(`/s/${this.spaceIdentifier}/attachments`, {
      body: {
        file_key: fileKey,
        signed_id: signedId
      }
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new UploadError(
        errorData.error || "アップロード完了の通知に失敗しました",
        "NOTIFICATION_ERROR"
      );
    }

    return response.json();
  }

  // リトライ処理
  private async withRetry<T>(
    operation: () => Promise<T>,
    errorCode?: string
  ): Promise<T> {
    try {
      return await operation();
    } catch (error) {
      // リトライ対象のエラーか判定
      const shouldRetry = 
        error instanceof UploadError &&
        ["NETWORK_ERROR", "TIMEOUT", "UPLOAD_FAILED"].includes(error.code || "") &&
        this.retryCount < this.maxRetries;

      if (!shouldRetry) {
        throw error;
      }

      // エクスポネンシャルバックオフ
      const delay = this.baseDelay * Math.pow(2, this.retryCount);
      this.retryCount++;

      console.log(`アップロードをリトライします (${this.retryCount}/${this.maxRetries})`);
      
      await new Promise(resolve => setTimeout(resolve, delay));
      return this.withRetry(operation, errorCode);
    }
  }

  // アップロードの実行
  async upload(): Promise<{ id: string; url: string }> {
    // バリデーション
    this.validateFileSize();
    this.validateFileType();

    try {
      // プリサイン用URLの取得
      const { upload_url, file_key, signed_id } = await this.getPresignedUrl();

      // ファイルアップロード（リトライ付き）
      await this.withRetry(() => this.uploadFile(upload_url));

      // アップロード完了通知
      const result = await this.notifyUploadComplete(file_key, signed_id);

      return result;
    } catch (error) {
      if (error instanceof UploadError) {
        throw error;
      }
      throw new UploadError(
        "予期しないエラーが発生しました",
        "UNKNOWN_ERROR"
      );
    }
  }

  // アップロードのキャンセル
  cancel(): void {
    if (this.xhr) {
      this.xhr.abort();
    }
  }
}