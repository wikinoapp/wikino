declare module "@rails/activestorage" {
  export interface Blob {
    signed_id: string;
    url?: string;
  }

  export class DirectUpload {
    constructor(file: File, url: string);
    create(
      callback: (error: Error | null, blob: Blob) => void,
      beforeStorage?: (request: XMLHttpRequest) => void
    ): void;
  }

  export function start(): void;
}

declare module "@rails/activestorage/src/file_checksum" {
  export class FileChecksum {
    static create(file: File, callback: (error: string | null, checksum?: string) => void): void;
  }
}