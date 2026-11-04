// 編集画面が保存の前提にする下書きの版 (サーバーの `DraftPage.ContentDigest()`) を持ち回る。
// 保存 (自動保存・下書き保存) には前提の版を添え、サーバーは版が今の版と違えば上書きせず、
// 409と今の版を返す。保存が成功するたびに応答のETagで版を置き換え、次の保存の前提にする。
//
// 保存は1件ずつ送る。前の保存の応答を待たずに次を送ると、次の保存が前の保存より古い版を
// 前提にすることになり、自分の直前の保存と競合してしまうためである。
//
// 公開のフォームも同じ版を前提にして送るため、前提の版が変わるたびにフォームの版の項目を書き換える。

import { beginHtmxDraftSave } from "./save-status";

// edit.templの下書き保存のボタンに付ける属性。htmxが送る要求を見分ける。
const SAVE_BUTTON_ATTRIBUTE = "data-draft-save-button";
const EDITOR_SELECTOR = "[data-markdown-editor]";

// 手動保存 (htmx) で版を送る項目名。公開のフォームの `title`・`body` に並べる。
const MANUAL_SAVE_DIGEST_FIELD = "draft_page_digest";

// edit.templの公開のフォームにある、公開の前提にする版の項目。
const PUBLISH_DIGEST_FIELD_SELECTOR = "[data-draft-page-digest-field]";

// DRAFT_SAVE_CONFLICT_EVENTは、保存が競合したときにエディタの要素で発火するイベントの名前。
export const DRAFT_SAVE_CONFLICT_EVENT = "draft-save-conflict";

interface DraftVersion {
  digest: string;
  // 409で受け取った今の版。競合していないあいだはnull。
  conflictDigest: string | null;
  // 最後に順番待ちに入れた保存。次の保存はこれが終わってから送る。
  queue: Promise<unknown>;
}

const versions = new WeakMap<HTMLElement, DraftVersion>();

// registerDraftVersionは、編集画面が開いたときの版をエディタの要素に結び付ける。公開が競合して
// 再表示した編集画面では、競合で受け取った今の版 (conflictDigest) も渡し、競合したまま始める。
export function registerDraftVersion(
  container: HTMLElement,
  digest: string,
  conflictDigest: string | null = null,
): void {
  versions.set(container, { digest, conflictDigest, queue: Promise.resolve() });
  syncPublishDigestField(digest);
}

// isDraftConflictedは、保存が競合したまま利用者が選んでいないかを返す。
export function isDraftConflicted(container: HTMLElement): boolean {
  return versions.get(container)?.conflictDigest != null;
}

// acceptConflictDigestは、競合で受け取った今の版を前提にし直す。利用者が「このまま上書きする」を
// 選んだときに呼び、次の保存で画面の内容をその版に上書きする。
export function acceptConflictDigest(container: HTMLElement): void {
  const version = versions.get(container);
  if (!version || version.conflictDigest === null) return;
  version.digest = version.conflictDigest;
  version.conflictDigest = null;
  syncPublishDigestField(version.digest);
}

// whenDraftSavesSettledは、順番待ちに入っている保存がすべて終わったら解決する。保存の成否は問わない。
// 公開の前に呼び、公開が自分の直前の保存より古い版を前提にしないようにする。
export function whenDraftSavesSettled(container: HTMLElement): Promise<void> {
  const version = versions.get(container);
  if (!version) return Promise.resolve();
  return version.queue.then(() => undefined);
}

// sendDraftSaveは、前の保存が終わってから、その時点の前提の版でsendを呼ぶ。応答の版で前提の版を
// 置き換え、409なら競合として記録してエディタの要素にイベントを発火する。
// sendは、待っているあいだに送る必要が無くなったらnullを返して送らずに済ませる。
export function sendDraftSave<T extends Response | null>(
  container: HTMLElement,
  send: (digest: string) => Promise<T>,
): Promise<T> {
  const version = versions.get(container);
  if (!version) return send("");

  const saving = version.queue.then(async () => {
    const response = await send(version.digest);
    if (response) applyResponse(container, version, response);
    return response;
  });
  // 失敗した保存も順番待ちを止めないよう、次の保存は成否を問わず待つ
  version.queue = saving.catch(() => undefined);
  return saving;
}

function applyResponse(container: HTMLElement, version: DraftVersion, response: Response): void {
  const digest = parseETag(response.headers.get("ETag"));
  if (response.ok) {
    if (digest !== null) {
      version.digest = digest;
      syncPublishDigestField(digest);
    }
    return;
  }
  if (response.status === 409) {
    // 今の版が分からないときは前提の版のままにし、上書きを選んでも再び競合を知らせる
    version.conflictDigest = digest ?? version.digest;
    container.dispatchEvent(new CustomEvent(DRAFT_SAVE_CONFLICT_EVENT, { bubbles: true }));
  }
}

// syncPublishDigestFieldは、公開のフォームが前提にする版を、保存の前提の版に揃える。
function syncPublishDigestField(digest: string): void {
  document.querySelectorAll<HTMLInputElement>(PUBLISH_DIGEST_FIELD_SELECTOR).forEach((field) => {
    field.value = digest;
  });
}

// parseETagは、ETagのヘッダーから版を取り出す。前段のプロキシが付けることのある `W/` は外す。
function parseETag(value: string | null): string | null {
  const match = value?.trim().match(/^(?:W\/)?"([^"]*)"$/);
  return match?.[1] ?? null;
}

// htmxが要求ごとに作る文脈。request.bodyは送るフォームの内容で、fetchはhtmxが送るのに使う関数。
interface HtmxRequestContext {
  request?: { body?: FormData | URLSearchParams | null };
  fetch?: (input: string, init: RequestInit) => Promise<Response>;
}

// initializeDraftVersionは、下書き保存のボタンが送るhtmxの要求を、自動保存と同じ順番待ちに入れ、
// 送る直前の入力と前提の版を添えて、その内容を保存の表示の判定に記録する。htmxは要求ごとに
// ctx.fetchで送るため、それを差し替える。
export function initializeDraftVersion(): void {
  document.addEventListener("htmx:before:request", handleBeforeRequest);
}

function handleBeforeRequest(event: Event): void {
  const source = event.target;
  if (!(source instanceof Element) || !source.hasAttribute(SAVE_BUTTON_ATTRIBUTE)) return;
  const ctx = (event as CustomEvent<{ ctx?: HtmxRequestContext }>).detail?.ctx;
  const container = document.querySelector<HTMLElement>(EDITOR_SELECTOR);
  const body = ctx?.request?.body;
  if (!ctx || !container || !body) return;

  const fetchRequest = ctx.fetch ?? window.fetch.bind(window);
  ctx.fetch = (input, init) =>
    sendDraftSave(container, (digest) => {
      // 待っているあいだの編集も含め、前提の版と同じ時点の入力を送る。
      const titleSelector = container.dataset.markdownEditorTitle;
      const bodySelector = container.dataset.markdownEditorTextarea;
      const title = titleSelector ? document.querySelector<HTMLInputElement>(titleSelector) : null;
      const textarea = bodySelector ? document.querySelector<HTMLTextAreaElement>(bodySelector) : null;
      if (title) body.set("title", title.value);
      if (textarea) body.set("body", textarea.value);
      body.set(MANUAL_SAVE_DIGEST_FIELD, digest);
      beginHtmxDraftSave(ctx, container, body);
      return fetchRequest(input, init);
    });
}
