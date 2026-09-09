import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeAttachmentLoader } from "./attachment-loader";

const SIGNED_URLS_ENDPOINT = "/attachments/signed_urls";

const SIGNED_URLS: Record<string, string> = {
  "att-image": "https://storage.example.dev/signed/cover.png",
  "att-second-image": "https://storage.example.dev/signed/figure.png",
  "att-video": "https://storage.example.dev/signed/clip.mp4",
  "att-doc": "https://storage.example.dev/signed/document.pdf",
};

// Build the markup the attachment filter (internal/markup/attachment_filter.go) emits for a body
// holding an inline image, an inline video and a file link. Every element starts unresolved (empty
// src / href="#") and carries the attachment id its signed URL is keyed by; the image is wrapped in
// an anchor that repeats the same id, which is what makes the request deduplicate ids.
//
// [Ja] インライン画像・インライン動画・ファイルリンクを持つ本文に対して、マークアップフィルタ
// (internal/markup/attachment_filter.go) が出力する DOM を組み立てる。各要素は未解決の状態
// (空の src / href="#") から始まり、署名付き URL のキーとなる添付ファイル ID を持つ。画像は同じ ID を
// 繰り返すアンカーに包まれており、これがリクエストでの ID の重複排除の対象になる。
function bodyMarkup(): string {
  return `
    <div class="wikino-markdown">
      <a href="#" data-attachment-id="att-image" data-attachment-link="true" class="wikino-attachment-image-link">
        <img src="" data-attachment-id="att-image" data-attachment-type="image" class="wikino-attachment-image" alt="cover.png">
      </a>
      <video src="" data-attachment-id="att-video" data-attachment-type="video" class="wikino-attachment-video" controls></video>
      <a href="#" data-attachment-id="att-doc" data-attachment-link="true">document.pdf</a>
    </div>
  `;
}

// Stand in for the Rails endpoint with the minimum the loader reads off the response. The
// parameters are declared so the recorded calls stay typed as the loader passes them.
//
// [Ja] Rails のエンドポイントを、ローダーがレスポンスから読む最小限で代替する。引数を宣言して
// いるのは、記録された呼び出しをローダーが渡した形のまま型付けするため。
function stubFetch() {
  const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => ({
    ok: true,
    json: async () => ({ signed_urls: SIGNED_URLS }),
  }));
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

// The loader is fire-and-forget, so wait out the microtasks of the fetch and of reading its body
// before asserting. A macrotask tick runs after both have settled, since the stub resolves at once.
//
// [Ja] ローダーは結果を待たずに走るため、fetch とそのボディ読み取りのマイクロタスクを消化してから
// 検証する。スタブは即座に解決するため、マクロタスクを 1 つ挟めば双方が完了した後になる。
function flush(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

function requestBody(fetchMock: ReturnType<typeof stubFetch>): { attachment_ids: string[] } {
  const [, init] = fetchMock.mock.calls[0] ?? [];
  return JSON.parse(String(init?.body)) as { attachment_ids: string[] };
}

function requestHeaders(fetchMock: ReturnType<typeof stubFetch>): Record<string, string> {
  const [, init] = fetchMock.mock.calls[0] ?? [];
  return (init?.headers ?? {}) as Record<string, string>;
}

describe("initializeAttachmentLoader", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // The page detail screen renders the body on the server, so the placeholders are already in the
  // document when the initializers run: nothing swaps them in, and without this pass the body keeps
  // its empty src and href="#" for the whole visit.
  //
  // [Ja] ページ表示画面は本文をサーバー側で描画するため、初期化処理が走る時点でプレースホルダーは
  // 既に文書の中にある。差し込むスワップは起きないため、この解決が無いと本文は滞在中ずっと空の src と
  // href="#" のままになる。
  it("resolves the placeholders of a server-rendered body at initialization", async () => {
    document.body.innerHTML = bodyMarkup();
    stubFetch();

    initializeAttachmentLoader();
    await flush();

    const image = document.querySelector("img") as HTMLImageElement;
    const video = document.querySelector("video") as HTMLVideoElement;
    const imageLink = document.querySelector("a.wikino-attachment-image-link") as HTMLAnchorElement;
    const documentLink = document.querySelector('a[data-attachment-id="att-doc"]') as HTMLAnchorElement;

    expect(image.src).toBe(SIGNED_URLS["att-image"]);
    expect(video.src).toBe(SIGNED_URLS["att-video"]);
    expect(imageLink.href).toBe(SIGNED_URLS["att-image"]);
    expect(documentLink.href).toBe(SIGNED_URLS["att-doc"]);
  });

  it("requests every attachment of the body once, deduplicating repeated ids", async () => {
    document.body.innerHTML = bodyMarkup();
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(SIGNED_URLS_ENDPOINT, expect.anything());
    expect(requestBody(fetchMock).attachment_ids).toEqual(["att-image", "att-video", "att-doc"]);
  });

  // The body of the page detail screen is the main content, so its first image is the screen's LCP
  // candidate and must not be deferred. Later images are below the fold on any body long enough to
  // hold them.
  //
  // [Ja] ページ表示画面の本文は主要コンテンツであり、その先頭の画像は画面の LCP 候補になるため
  // 遅延させてはならない。2 枚目以降は、それを収める長さの本文であればファーストビューの外にある。
  it("leaves the first image of the body eager and lazy-loads the images after it", async () => {
    document.body.innerHTML = `
      <div class="wikino-markdown">
        <img src="" data-attachment-id="att-image" class="wikino-attachment-image" alt="cover.png">
        <p>本文</p>
        <img src="" data-attachment-id="att-second-image" class="wikino-attachment-image" alt="figure.png">
      </div>
    `;
    stubFetch();

    initializeAttachmentLoader();
    await flush();

    const [first, second] = Array.from(document.querySelectorAll("img"));

    expect(first?.src).toBe(SIGNED_URLS["att-image"]);
    expect(first?.loading).not.toBe("lazy");
    expect(second?.src).toBe(SIGNED_URLS["att-second-image"]);
    expect(second?.loading).toBe("lazy");
  });

  it("sends no request when the document holds no placeholders", async () => {
    document.body.innerHTML = `<div class="wikino-markdown"><p>本文だけのページ</p></div>`;
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();

    expect(fetchMock).not.toHaveBeenCalled();
  });

  // The page detail screen is reachable by guests and renders no CSRF token input, while the page
  // editor renders one. The endpoint opts out of Rails-side forgery protection, so the guest request
  // must go out on its own rather than carrying an empty token.
  //
  // [Ja] ページ表示画面はゲストが到達でき CSRF トークンの input を描画しないが、ページ編集画面は
  // 描画する。エンドポイントは Rails 側の CSRF 検証をオプトアウトしているため、ゲストからの
  // リクエストは空のトークンを載せるのではなく、そのまま送られる必要がある。
  it("omits the CSRF header on a screen that renders no token input", async () => {
    document.body.innerHTML = bodyMarkup();
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(requestHeaders(fetchMock)).not.toHaveProperty("X-CSRF-Token");
  });

  it("sends the CSRF header on a screen that renders the token input", async () => {
    document.body.innerHTML = `
      <input type="hidden" id="page-edit-csrf-token" value="token-from-the-editor">
      ${bodyMarkup()}
    `;
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();

    expect(requestHeaders(fetchMock)["X-CSRF-Token"]).toBe("token-from-the-editor");
  });

  // The page-editor preview arrives through htmx, and the settle event is the only announcement it
  // gets.
  //
  // [Ja] ページ編集画面のプレビューは htmx 経由で届き、settle イベントがその唯一の通知手段になる。
  it("resolves the placeholders of a subtree swapped in by htmx", async () => {
    document.body.innerHTML = `<div id="page-edit-preview-content"></div>`;
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();
    expect(fetchMock).not.toHaveBeenCalled();

    const preview = document.getElementById("page-edit-preview-content") as HTMLElement;
    preview.innerHTML = bodyMarkup();
    preview.dispatchEvent(new Event("htmx:after:settle", { bubbles: true }));
    await flush();

    const image = document.querySelector("img") as HTMLImageElement;
    expect(image.src).toBe(SIGNED_URLS["att-image"]);
  });

  // The page detail screen swaps its related-page listings through htmx while the resolved body
  // stays in place. Scanning the whole document on each settle would hand the body's short-lived
  // signed URLs back to the endpoint on every "load more".
  //
  // [Ja] ページ表示画面は解決済みの本文をそのままに、関連ページ一覧を htmx でスワップする。settle の
  // たびに文書全体を走査すると、「もっと見る」を押すたびに本文の短命な署名付き URL を取得し直すことに
  // なる。
  it("does not refetch a resolved body when another part of the screen settles", async () => {
    document.body.innerHTML = `${bodyMarkup()}<div id="page-link-list-pagination"></div>`;
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const pagination = document.getElementById("page-link-list-pagination") as HTMLElement;
    pagination.dispatchEvent(new Event("htmx:after:settle", { bubbles: true }));
    await flush();

    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
