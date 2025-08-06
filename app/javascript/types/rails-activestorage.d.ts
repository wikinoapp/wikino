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