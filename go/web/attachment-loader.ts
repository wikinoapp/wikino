// Resolves attachment placeholders in rendered page bodies. The markup filter
// (internal/markup/attachment_filter.go) emits placeholders carrying a data-attachment-id whose
// real URL is fetched on the client (same scheme as the Rails attachment_loader_controller):
// <img>/<video> start with an empty src and a pending class, and <a> starts with href="#". This
// module mirrors that controller for the Go frontend: it collects the placeholders, batch-requests
// short-lived signed URLs from POST /attachments/signed_urls, and fills in src/href. For media it
// also swaps the pending class for the loaded one once the resource finishes loading, so the CSS
// fade-in triggers; anchors only get their href filled.
//
// A body reaches the DOM two ways, so the loader has two entry points. The page detail screen
// renders the body on the server, which is already in the DOM when the initializers run, so it is
// resolved once at initialization. The page-editor preview is injected by htmx, which announces
// itself with htmx:after:settle, so each settled subtree is resolved as it arrives.
//
// [Ja] 描画されたページ本文の添付ファイルプレースホルダーを解決する。マークアップフィルタ
// (internal/markup/attachment_filter.go) は data-attachment-id を持つプレースホルダーを出力し、
// 実 URL はクライアント側で取得する前提になっている (Rails の attachment_loader_controller と
// 同じ方式)。<img>/<video> は空の src と保留中クラスを持ち、<a> は href="#" を持つ。本モジュールは
// その方式を Go フロントエンドに移したもので、プレースホルダーを集め、POST /attachments/signed_urls
// から短命の署名付き URL をバッチ取得して src/href を埋める。
// メディアではさらにリソースの読み込み完了時に保留中クラスを読み込み済みクラスへ差し替えて CSS の
// フェードインを発火させる。アンカーは href を埋めるだけ。
//
// 本文が DOM に現れる経路は 2 つあるため、ローダーの起点も 2 つある。ページ表示画面は本文を
// サーバー側で描画し、初期化処理が走る時点で既に DOM にあるため、初期化時に一度だけ解決する。
// ページエディタのプレビューは htmx が差し込み、htmx:after:settle で通知されるため、
// スワップされた部分木ごとに解決する。

const CSRF_TOKEN_INPUT_ID = "page-edit-csrf-token";
const SIGNED_URLS_ENDPOINT = "/attachments/signed_urls";

const IMAGE_PENDING_CLASS = "wikino-attachment-image";
const IMAGE_LOADED_CLASS = "wikino-attachment-image-loaded";
const VIDEO_PENDING_CLASS = "wikino-attachment-video";
const VIDEO_LOADED_CLASS = "wikino-attachment-video-loaded";

interface SignedUrlsResponse {
  signed_urls?: Record<string, string>;
}

export function initializeAttachmentLoader(): void {
  document.addEventListener("htmx:after:settle", handleAfterSettle);

  // A server-rendered body arrives with the document itself, so no swap event ever announces it.
  //
  // [Ja] サーバー側で描画された本文は文書と一緒に届くため、スワップイベントでは通知されない。
  void loadAttachments(document.body);
}

// handleAfterSettle resolves the placeholders of the subtree that just settled. The event bubbles
// from every htmx swap, so scoping the scan to the swapped element keeps swaps that carry no
// placeholders (the related-page listings, the autosave OOB swaps) from rescanning the rest of the
// document, and keeps a resolved body from being refetched when something else on the screen swaps.
//
// [Ja] handleAfterSettle は今スワップされた部分木のプレースホルダーを解決する。イベントはあらゆる
// htmx スワップから伝播するため、走査をスワップされた要素に絞ることで、プレースホルダーを持たない
// スワップ (関連ページ一覧、自動保存の OOB スワップ) が文書の残りを走査し直さないようにし、画面の
// 別の場所がスワップされたときに解決済みの本文を取得し直さないようにする。
function handleAfterSettle(event: Event): void {
  const settled = event.target;
  if (!(settled instanceof HTMLElement)) return;

  void loadAttachments(settled);
}

async function loadAttachments(root: HTMLElement): Promise<void> {
  // Skip placeholders already resolved by a previous run so a re-settle does not refetch them.
  //
  // [Ja] 前回の実行で既に解決済みのプレースホルダーはスキップし、再スワップで取得し直さないようにする。
  const elements = Array.from(root.querySelectorAll<HTMLElement>("[data-attachment-id]")).filter(
    (el) => !isResolved(el),
  );
  if (elements.length === 0) return;

  const attachmentIds = Array.from(
    new Set(elements.map((el) => el.dataset.attachmentId).filter((id): id is string => Boolean(id))),
  );
  if (attachmentIds.length === 0) return;

  const signedUrls = await fetchSignedUrls(attachmentIds);
  if (!signedUrls) return;

  // The first image of a body is the LCP candidate of the page detail screen, so it keeps the
  // browser default (eager) while every later image is lazy-loaded. A placeholder carries no
  // dimensions until it resolves, which collapses the layout and leaves a viewport test with
  // nothing to measure, so document order is the signal available here.
  //
  // [Ja] 本文の先頭の画像はページ表示画面の LCP 候補になるため、ブラウザ既定 (eager) のままにし、
  // 以降の画像を遅延読み込みにする。プレースホルダーは解決されるまで寸法を持たず、潰れたレイアウト
  // ではビューポート判定に測るものが無いため、ここで使える手がかりは文書順になる。
  const lcpCandidate = elements.find((el) => el instanceof HTMLImageElement);

  for (const el of elements) {
    const attachmentId = el.dataset.attachmentId;
    if (!attachmentId) continue;

    const signedUrl = signedUrls[attachmentId];
    if (!signedUrl) continue;

    applySignedUrl(el, signedUrl, el !== lcpCandidate);
  }
}

function isResolved(el: HTMLElement): boolean {
  return el.classList.contains(IMAGE_LOADED_CLASS) || el.classList.contains(VIDEO_LOADED_CLASS);
}

// fetchSignedUrls batch-requests signed URLs, keyed by attachment id. It sends the request the
// same way as the upload flow (file-upload-handler.ts POSTing to /attachments/presign): a JSON
// body plus an X-CSRF-Token header read from the page. This endpoint is a Rails route reached
// through the reverse proxy, and Rails opts out of CSRF for it (skip_forgery_protection), so the
// header is carried for consistency with the upload path rather than verified here.
//
// [Ja] fetchSignedUrls は添付ファイル ID をキーにした署名付き URL をバッチ取得する。リクエストは
// アップロード経路 (file-upload-handler.ts が /attachments/presign へ POST するのと同じ方式) に
// 揃え、JSON ボディとページから読んだ X-CSRF-Token ヘッダーを送る。本エンドポイントは
// リバースプロキシ越しに到達する Rails ルートで、Rails 側は CSRF をオプトアウトしている
// (skip_forgery_protection) ため、ヘッダーはここで検証されるのではなくアップロード経路との
// 一貫性のために付けている。
async function fetchSignedUrls(attachmentIds: string[]): Promise<Record<string, string> | null> {
  try {
    const response = await fetch(SIGNED_URLS_ENDPOINT, {
      method: "POST",
      headers: buildHeaders(),
      body: JSON.stringify({ attachment_ids: attachmentIds }),
    });

    if (!response.ok) {
      console.error("Failed to fetch attachment signed URLs:", response.status);
      return null;
    }

    const data = (await response.json()) as SignedUrlsResponse;
    return data.signed_urls ?? {};
  } catch (error) {
    console.error("Error loading attachment signed URLs:", error);
    return null;
  }
}

// buildHeaders adds the CSRF header only when the screen carries a token. The page editor renders
// the token input, while the page detail screen renders none: it is reachable by guests, who have
// no session to protect. Sending an empty header would claim a token the page does not hold, and
// the endpoint accepts the request without one either way.
//
// [Ja] buildHeaders は画面がトークンを持つときだけ CSRF ヘッダーを付ける。ページ編集画面は
// トークンの input を描画するが、ページ表示画面は描画しない。ゲストが到達でき、保護すべき
// セッションを持たないためである。空のヘッダーを送るとページが持っていないトークンを名乗ることに
// なり、いずれにせよエンドポイントはトークン無しでもリクエストを受け付ける。
function buildHeaders(): HeadersInit {
  const headers: Record<string, string> = { "Content-Type": "application/json" };

  const csrfToken = readCsrfToken();
  if (csrfToken !== "") {
    headers["X-CSRF-Token"] = csrfToken;
  }

  return headers;
}

function readCsrfToken(): string {
  const input = document.getElementById(CSRF_TOKEN_INPUT_ID);
  return input instanceof HTMLInputElement ? input.value : "";
}

// applySignedUrl fills in the real URL per element type and, for media, swaps the pending class for
// the loaded class once the resource finishes loading so the CSS fade-in runs on fully-loaded media.
// lazy marks an image the viewer is unlikely to see first (see the LCP candidate in
// loadAttachments); the URL is assigned last so the fetch starts with the loading mode decided.
//
// [Ja] applySignedUrl は要素の種類ごとに実 URL を埋め、メディアではリソースの読み込み完了時に保留中
// クラスを読み込み済みクラスへ差し替えて、読み込み済みメディアに対して CSS のフェードインを走らせる。
// lazy は閲覧者が最初に見るとは考えにくい画像を表す (loadAttachments の LCP 候補を参照)。URL の
// 代入を最後に置くのは、読み込みモードを決めた状態で取得を開始させるため。
function applySignedUrl(el: HTMLElement, signedUrl: string, lazy: boolean): void {
  if (el instanceof HTMLImageElement) {
    el.addEventListener(
      "load",
      () => {
        el.classList.remove(IMAGE_PENDING_CLASS);
        el.classList.add(IMAGE_LOADED_CLASS);
      },
      { once: true },
    );
    if (lazy) {
      el.loading = "lazy";
    }
    el.src = signedUrl;
    return;
  }

  if (el instanceof HTMLVideoElement) {
    el.addEventListener(
      "loadeddata",
      () => {
        el.classList.remove(VIDEO_PENDING_CLASS);
        el.classList.add(VIDEO_LOADED_CLASS);
      },
      { once: true },
    );
    el.src = signedUrl;
    return;
  }

  if (el instanceof HTMLAnchorElement) {
    el.href = signedUrl;
  }
}
