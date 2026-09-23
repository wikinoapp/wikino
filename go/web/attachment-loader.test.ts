import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeAttachmentLoader } from "./attachment-loader";

const SIGNED_URLS_ENDPOINT = "/attachments/signed_urls";

const SIGNED_URLS: Record<string, string> = {
  "att-image": "https://storage.example.dev/signed/cover.png",
  "att-second-image": "https://storage.example.dev/signed/figure.png",
  "att-video": "https://storage.example.dev/signed/clip.mp4",
  "att-doc": "https://storage.example.dev/signed/document.pdf",
};

// インライン画像・インライン動画・ファイルリンクを持つ本文に対して、マークアップフィルタ
// (internal/markup/attachment_filter.go) が出力するDOMを組み立てる。各要素は未解決の状態
// (空のsrc / href="#") から始まり、署名付きURLのキーとなる添付ファイルIDを持つ。画像は同じIDを
// 繰り返すアンカーに包まれており、これがリクエストでのIDの重複排除の対象になる。
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

// Railsのエンドポイントを、ローダーがレスポンスから読む最小限で代替する。引数を宣言して
// いるのは、記録された呼び出しをローダーが渡した形のまま型付けするため。
function stubFetch() {
  const fetchMock = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit) => ({
    ok: true,
    json: async () => ({ signed_urls: SIGNED_URLS }),
  }));
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

// ローダーは結果を待たずに走るため、fetchとそのボディ読み取りのマイクロタスクを消化してから
// 検証する。スタブは即座に解決するため、マクロタスクを1つ挟めば双方が完了した後になる。
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

  // ページ表示画面は本文をサーバー側で描画するため、初期化処理が走る時点でプレースホルダーは
  // 既に文書の中にある。差し込むスワップは起きないため、この解決が無いと本文は滞在中ずっと空のsrcと
  // href="#" のままになる。
  it("初期化時にサーバーで描画された本文のプレースホルダーを解決する", async () => {
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

  it("IDの重複を除き、本文の各添付ファイルを1回ずつ取得する", async () => {
    document.body.innerHTML = bodyMarkup();
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(SIGNED_URLS_ENDPOINT, expect.anything());
    expect(requestBody(fetchMock).attachment_ids).toEqual(["att-image", "att-video", "att-doc"]);
  });

  // ページ表示画面の本文は主要コンテンツであり、その先頭の画像は画面のLCP候補になるため
  // 遅延させてはならない。2枚目以降は、それを収める長さの本文であればファーストビューの外にある。
  it("本文の先頭画像は即時読み込みとし、以降の画像を遅延読み込みする", async () => {
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

  it("文書にプレースホルダーが無ければリクエストを送らない", async () => {
    document.body.innerHTML = `<div class="wikino-markdown"><p>本文だけのページ</p></div>`;
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();

    expect(fetchMock).not.toHaveBeenCalled();
  });

  // ページ表示画面はゲストが到達できCSRFトークンのinputを描画しないが、ページ編集画面は
  // 描画する。エンドポイントはRails側のCSRF検証をオプトアウトしているため、ゲストからの
  // リクエストは空のトークンを載せるのではなく、そのまま送られる必要がある。
  it("トークンの入力要素を描画しない画面ではCSRFヘッダーを付けない", async () => {
    document.body.innerHTML = bodyMarkup();
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(requestHeaders(fetchMock)).not.toHaveProperty("X-CSRF-Token");
  });

  it("トークンの入力要素を描画する画面ではCSRFヘッダーを付ける", async () => {
    document.body.innerHTML = `
      <input type="hidden" id="page-edit-csrf-token" value="token-from-the-editor">
      ${bodyMarkup()}
    `;
    const fetchMock = stubFetch();

    initializeAttachmentLoader();
    await flush();

    expect(requestHeaders(fetchMock)["X-CSRF-Token"]).toBe("token-from-the-editor");
  });

  // ページ編集画面のプレビューはhtmx経由で届き、settleイベントがその唯一の通知手段になる。
  it("htmxでスワップされた部分木のプレースホルダーを解決する", async () => {
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

  // ページ表示画面は解決済みの本文をそのままに、関連ページ一覧をhtmxでスワップする。settleの
  // たびに文書全体を走査すると、「もっと見る」を押すたびに本文の短命な署名付きURLを取得し直すことに
  // なる。
  it("画面の別の部分がスワップされても解決済みの本文を再取得しない", async () => {
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
