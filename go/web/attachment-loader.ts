// Resolves attachment placeholders in the page-editor preview. The markup filter
// (internal/markup/attachment_filter.go) emits placeholders carrying a data-attachment-id whose
// real URL is fetched on the client (same scheme as the Rails attachment_loader_controller):
// <img>/<video> start with an empty src and a pending class, and <a> starts with href="#". This
// module mirrors that controller for the Go frontend: after the preview HTML is swapped in, it
// collects the placeholders, batch-requests short-lived signed URLs from POST
// /attachments/signed_urls, and fills in src/href. For media it also swaps the pending class for
// the loaded one once the resource finishes loading, so the CSS fade-in triggers; anchors only get
// their href filled.
//
// The preview body is injected by htmx (innerHTML swap into #page-edit-preview-content), so the
// loader runs on htmx:after:settle rather than at DOMContentLoaded like the other initializers.
//
// [Ja] ページエディタのプレビューの添付ファイルプレースホルダーを解決する。マークアップフィルタ
// (internal/markup/attachment_filter.go) は data-attachment-id を持つプレースホルダーを出力し、
// 実 URL はクライアント側で取得する前提になっている (Rails の attachment_loader_controller と
// 同じ方式)。<img>/<video> は空の src と保留中クラスを持ち、<a> は href="#" を持つ。本モジュールは
// その方式を Go フロントエンドに移したもので、プレビュー HTML がスワップされた後にプレースホルダーを
// 集め、POST /attachments/signed_urls から短命の署名付き URL をバッチ取得して src/href を埋める。
// メディアではさらにリソースの読み込み完了時に保留中クラスを読み込み済みクラスへ差し替えて CSS の
// フェードインを発火させる。アンカーは href を埋めるだけ。
//
// プレビュー本文は htmx が差し込む (#page-edit-preview-content への innerHTML スワップ) ため、
// ローダーは他の初期化処理のような DOMContentLoaded ではなく htmx:after:settle で起動する。

const PREVIEW_CONTENT_ID = "page-edit-preview-content";
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
}

// handleAfterSettle runs the loader only when the settle is the preview swap. The event bubbles from
// every htmx swap (including the autosave OOB swaps), so it is filtered to the preview container,
// which is the innerHTML swap target and therefore the element the event fires on.
//
// [Ja] handleAfterSettle はプレビューのスワップのときだけローダーを走らせる。イベントは (自動保存の
// OOB スワップを含む) あらゆる htmx スワップから伝播するため、innerHTML スワップの対象であり
// イベントの発火元でもあるプレビューコンテナに絞り込む。
function handleAfterSettle(event: Event): void {
  const previewContent = document.getElementById(PREVIEW_CONTENT_ID);
  if (!previewContent || event.target !== previewContent) return;

  void loadAttachments(previewContent);
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

  for (const el of elements) {
    const attachmentId = el.dataset.attachmentId;
    if (!attachmentId) continue;

    const signedUrl = signedUrls[attachmentId];
    if (!signedUrl) continue;

    applySignedUrl(el, signedUrl);
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
      headers: {
        "Content-Type": "application/json",
        "X-CSRF-Token": readCsrfToken(),
      },
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

function readCsrfToken(): string {
  const input = document.getElementById(CSRF_TOKEN_INPUT_ID);
  return input instanceof HTMLInputElement ? input.value : "";
}

// applySignedUrl fills in the real URL per element type and, for media, swaps the pending class for
// the loaded class once the resource finishes loading so the CSS fade-in runs on fully-loaded media.
//
// [Ja] applySignedUrl は要素の種類ごとに実 URL を埋め、メディアではリソースの読み込み完了時に保留中
// クラスを読み込み済みクラスへ差し替えて、読み込み済みメディアに対して CSS のフェードインを走らせる。
function applySignedUrl(el: HTMLElement, signedUrl: string): void {
  if (el instanceof HTMLImageElement) {
    el.addEventListener(
      "load",
      () => {
        el.classList.remove(IMAGE_PENDING_CLASS);
        el.classList.add(IMAGE_LOADED_CLASS);
      },
      { once: true },
    );
    el.loading = "lazy";
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
