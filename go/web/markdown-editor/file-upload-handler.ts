import type { EditorView } from "codemirror";
import SparkMD5 from "spark-md5";
import { DirectUpload, UploadError } from "./direct-upload";
import type { UploadProgress } from "./direct-upload";
import { insertUploadPlaceholder, replacePlaceholderWithUrl, removePlaceholder } from "./upload-placeholder";

interface PresignResponse {
  directUploadUrl: string;
  directUploadHeaders: Record<string, string>;
  blobSignedId: string;
  attachmentId: string;
}

const CHUNK_SIZE = 2097152; // 2MB

async function calculateFileChecksum(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const spark = new SparkMD5.ArrayBuffer();
    const reader = new FileReader();
    let offset = 0;

    reader.onload = (event) => {
      if (!event.target?.result) {
        reject(new Error("チェックサムの計算に失敗しました"));
        return;
      }
      spark.append(event.target.result as ArrayBuffer);
      offset += CHUNK_SIZE;

      if (offset < file.size) {
        readNextChunk();
      } else {
        resolve(btoa(spark.end(true)));
      }
    };

    reader.onerror = () => {
      reject(new Error("チェックサムの計算に失敗しました"));
    };

    function readNextChunk(): void {
      const slice = file.slice(offset, offset + CHUNK_SIZE);
      reader.readAsArrayBuffer(slice);
    }

    readNextChunk();
  });
}

const FILE_SIZE_LIMITS = {
  image: 10 * 1024 * 1024,
  video: 100 * 1024 * 1024,
  other: 25 * 1024 * 1024,
} as const;

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
    "text/markdown",
    "application/json",
    "application/x-zip-compressed",
    "application/zip",
    "application/gzip",
    "application/x-gzip",
    "application/x-tar",
    "application/x-compressed-tar",
  ],
} as const;

const ALL_ALLOWED_TYPES: string[] = [
  ...ALLOWED_FILE_TYPES.image,
  ...ALLOWED_FILE_TYPES.video,
  ...ALLOWED_FILE_TYPES.document,
];

const MAX_IMAGE_DIMENSION = 10000;

export class FileUploadHandler {
  private editorView: EditorView;
  private spaceIdentifier: string;
  private csrfToken: string;
  private isTestEnvironment: boolean;

  constructor(editorView: EditorView, spaceIdentifier: string, csrfToken: string) {
    this.editorView = editorView;
    this.spaceIdentifier = spaceIdentifier;
    this.csrfToken = csrfToken;
    this.isTestEnvironment =
      navigator.userAgent.includes("HeadlessChrome") ||
      navigator.userAgent.includes("Selenium") ||
      window.location.hostname === "127.0.0.1";
  }

  async handleFileUpload(files: File[], position: number): Promise<void> {
    for (const file of files) {
      const placeholderId = insertUploadPlaceholder(this.editorView, file.name, position);

      try {
        await this.validateFile(file);

        let width: number | undefined;
        let height: number | undefined;

        if (file.type.startsWith("image/")) {
          try {
            const dimensions = await this.getImageDimensions(file);
            width = dimensions.width;
            height = dimensions.height;
          } catch (error) {
            console.warn("画像サイズの取得に失敗しました:", error);
          }
        }

        const checksum = await calculateFileChecksum(file);
        const presignData = await this.getPresignUrl(file, checksum);
        await this.uploadFile(file, presignData);

        const attachmentUrl = `/attachments/${presignData.attachmentId}`;
        replacePlaceholderWithUrl(this.editorView, placeholderId, attachmentUrl, file.name, width, height, file.type);
      } catch (error) {
        console.error("Upload error:", error);
        removePlaceholder(this.editorView, placeholderId);

        if (error instanceof UploadError) {
          this.showErrorMessage(error.message);
        } else {
          this.showErrorMessage("ファイルのアップロードに失敗しました");
        }
      }
    }
  }

  private async getPresignUrl(file: File, checksum: string): Promise<PresignResponse> {
    const endpoint = this.isTestEnvironment
      ? "/_test/attachments/presign"
      : `/s/${this.spaceIdentifier}/attachments/presign`;

    const response = await fetch(endpoint, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": this.csrfToken,
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

    return (await response.json()) as PresignResponse;
  }

  private async uploadFile(file: File, presignData: PresignResponse): Promise<void> {
    if (this.isTestEnvironment) {
      const response = await fetch("/_test/attachments/upload", {
        method: "PUT",
        headers: presignData.directUploadHeaders,
        body: file,
      });

      if (!response.ok) {
        throw new UploadError("アップロードに失敗しました");
      }
      return;
    }

    const uploader = new DirectUpload(
      file,
      presignData.directUploadUrl,
      presignData.directUploadHeaders,
      (progress: UploadProgress) => {
        console.log(`Uploading ${file.name}: ${progress.percentage}%`);
      },
    );

    await uploader.upload();
  }

  private showErrorMessage(message: string): void {
    const flashToastEvent = new CustomEvent("flash-toast:show", {
      detail: {
        type: "alert",
        messageHtml: message,
      },
    });
    document.dispatchEvent(flashToastEvent);
  }

  private async validateFile(file: File): Promise<void> {
    if (!ALL_ALLOWED_TYPES.includes(file.type)) {
      throw new UploadError("このファイル形式はアップロードできません");
    }

    let category: "image" | "video" | "other";
    if ((ALLOWED_FILE_TYPES.image as readonly string[]).includes(file.type)) {
      category = "image";
    } else if ((ALLOWED_FILE_TYPES.video as readonly string[]).includes(file.type)) {
      category = "video";
    } else {
      category = "other";
    }

    const limit = FILE_SIZE_LIMITS[category];
    if (file.size > limit) {
      const limitMB = Math.round(limit / (1024 * 1024));
      throw new UploadError(`ファイルサイズが制限（${limitMB}MB）を超えています`);
    }

    if (file.type.startsWith("image/")) {
      try {
        const dimensions = await this.getImageDimensions(file);
        if (dimensions.width > MAX_IMAGE_DIMENSION || dimensions.height > MAX_IMAGE_DIMENSION) {
          throw new UploadError(
            `画像のサイズが大きすぎます。${MAX_IMAGE_DIMENSION}×${MAX_IMAGE_DIMENSION}ピクセル以下にしてください`,
          );
        }
      } catch (error) {
        if (error instanceof UploadError) {
          throw error;
        }
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
