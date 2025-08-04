declare module "@rails/request.js" {
  interface RequestOptions {
    body?: any;
    headers?: Record<string, string>;
    responseKind?: "json" | "text" | "html";
  }

  interface FetchResponse {
    ok: boolean;
    status: number;
    statusText: string;
    headers: Headers;
    redirected: boolean;
    type: ResponseType;
    url: string;
    json: Promise<any>; // jsonはプロパティで、Promiseを返す
    text(): Promise<string>;
    response: Response;
  }

  export function get(url: string, options?: RequestOptions): Promise<FetchResponse>;
  export function post(url: string, options?: RequestOptions): Promise<FetchResponse>;
  export function put(url: string, options?: RequestOptions): Promise<FetchResponse>;
  export function patch(url: string, options?: RequestOptions): Promise<FetchResponse>;
  export function destroy(url: string, options?: RequestOptions): Promise<FetchResponse>;
}