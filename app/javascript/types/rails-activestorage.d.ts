declare module "@rails/activestorage" {
  export function start(): void;
  export class DirectUpload {
    constructor(file: File, url: string, delegate?: DirectUploadDelegate);
    create(callback: (error: Error | null, blob?: Blob) => void): void;
  }

  interface DirectUploadDelegate {
    directUploadWillCreateBlobWithXHR?(xhr: XMLHttpRequest): void;
    directUploadWillStoreFileWithXHR?(xhr: XMLHttpRequest): void;
  }

  interface Blob {
    id: string;
    key: string;
    filename: string;
    content_type: string;
    byte_size: number;
    checksum: string;
    signed_id: string;
  }
}

declare module "@rails/activestorage/src/file_checksum" {
  export class FileChecksum {
    static create(file: File, callback: (error: Error | null, checksum?: string) => void): void;
  }
}
