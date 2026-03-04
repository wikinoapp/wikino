export interface UploadProgress {
  loaded: number;
  total: number;
  percentage: number;
}

export class UploadError extends Error {
  constructor(
    message: string,
    public code?: string,
  ) {
    super(message);
    this.name = "UploadError";
  }
}

export class DirectUpload {
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

  async upload(): Promise<{ id: string; url: string }> {
    return new Promise((resolve, reject) => {
      this.xhr = new XMLHttpRequest();
      this.xhr.open("PUT", this.directUploadUrl, true);
      this.xhr.responseType = "text";

      for (const [key, value] of Object.entries(this.directUploadHeaders)) {
        this.xhr.setRequestHeader(key, value);
      }

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

      this.xhr.addEventListener("load", () => {
        if (this.xhr!.status >= 200 && this.xhr!.status < 300) {
          resolve({
            id: "uploaded",
            url: this.directUploadUrl.split("?")[0]!,
          });
        } else {
          reject(new UploadError(`アップロードに失敗しました (ステータス: ${this.xhr!.status})`, "UPLOAD_FAILED"));
        }
      });

      this.xhr.addEventListener("error", () => {
        reject(new UploadError("ネットワークエラーが発生しました", "NETWORK_ERROR"));
      });

      this.xhr.send(this.file);
    });
  }

  cancel(): void {
    if (this.xhr) {
      this.xhr.abort();
    }
  }
}
