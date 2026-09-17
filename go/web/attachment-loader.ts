// 描画されたページ本文の添付ファイルプレースホルダーを解決する。マークアップフィルタ
// (internal/markup/attachment_filter.go) はdata-attachment-idを持つプレースホルダーを出力し、
// 実URLはクライアント側で取得する。<img>/<video> は空のsrcと保留中クラスを持ち、<a> は
// href="#" を持つ。本モジュールはプレースホルダーを集め、POST /attachments/signed_urlsから
// 短命の署名付きURLをバッチ取得してsrc/hrefを埋める。
// メディアではさらにリソースの読み込み完了時に保留中クラスを読み込み済みクラスへ差し替えてCSSの
// フェードインを発火させる。アンカーはhrefを埋めるだけ。
//
// 本文がDOMに現れる経路は2つあるため、ローダーの起点も2つある。ページ表示画面は本文を
// サーバー側で描画し、初期化処理が走る時点で既にDOMにあるため、初期化時に一度だけ解決する。
// ページエディタのプレビューはhtmxが差し込み、htmx:after:settleで通知されるため、
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

  // サーバー側で描画された本文は文書と一緒に届くため、スワップイベントでは通知されない。
  void loadAttachments(document.body);
}

// handleAfterSettleは今スワップされた部分木のプレースホルダーを解決する。イベントはあらゆる
// htmxスワップから伝播するため、走査をスワップされた要素に絞ることで、プレースホルダーを持たない
// スワップ (関連ページ一覧、自動保存のOOBスワップ) が文書の残りを走査し直さないようにし、画面の
// 別の場所がスワップされたときに解決済みの本文を取得し直さないようにする。
function handleAfterSettle(event: Event): void {
  const settled = event.target;
  if (!(settled instanceof HTMLElement)) return;

  void loadAttachments(settled);
}

async function loadAttachments(root: HTMLElement): Promise<void> {
  // 前回の実行で既に解決済みのプレースホルダーはスキップし、再スワップで取得し直さないようにする。
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

  // 本文の先頭の画像はページ表示画面のLCP候補になるため、ブラウザ既定 (eager) のままにし、
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

// fetchSignedUrlsは添付ファイルIDをキーにした署名付きURLをバッチ取得する。リクエストは
// アップロード経路 (file-upload-handler.tsが /attachments/presignへPOSTするのと同じ方式) に
// 揃え、JSONボディとページから読んだX-CSRF-Tokenヘッダーを送る。本エンドポイントは
// リバースプロキシ越しに到達するRailsルートで、Rails側はCSRFをオプトアウトしている
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
      console.error("添付ファイルの署名付きURLの取得に失敗しました:", response.status);
      return null;
    }

    const data = (await response.json()) as SignedUrlsResponse;
    return data.signed_urls ?? {};
  } catch (error) {
    console.error("添付ファイルの署名付きURLの読み込み中にエラーが発生しました:", error);
    return null;
  }
}

// buildHeadersは画面がトークンを持つときだけCSRFヘッダーを付ける。ページ編集画面は
// トークンのinputを描画するが、ページ表示画面は描画しない。ゲストが到達でき、保護すべき
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

// applySignedUrlは要素の種類ごとに実URLを埋め、メディアではリソースの読み込み完了時に保留中
// クラスを読み込み済みクラスへ差し替えて、読み込み済みメディアに対してCSSのフェードインを走らせる。
// lazyは閲覧者が最初に見るとは考えにくい画像を表す (loadAttachmentsのLCP候補を参照)。URLの
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
