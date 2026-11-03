// 下書きの保存 (自動保存・下書き保存) に失敗したことを、自動保存の表示の位置で知らせる。
// edit.templの保存の表示 ([data-draft-save-status]) が、保存時刻の表示とエラーの表示
// ([data-draft-save-error]) と、エラーの文面を持つ。エラーがあるあいだ保存時刻の表示を隠すのは、
// edit.templのCSSが行う。保存時刻の表示は保存のたびにOOBスワップで差し替わり、スクリプトで
// 隠しても差し替えで戻ってしまうためである。

// SaveKindは、失敗した保存の種類。エラーの冒頭の文面を選ぶのに使う。
export type SaveKind = "autosave" | "manual";

const STATUS_SELECTOR = "[data-draft-save-status]";
const ERROR_SELECTOR = "[data-draft-save-error]";

// edit.templの下書き保存のボタンに付ける属性。htmxの応答を見て表示を切り替えるボタンを見分ける。
const SAVE_BUTTON_ATTRIBUTE = "data-draft-save-button";
const EDITOR_SELECTOR = "[data-markdown-editor]";
// components/load_more_navigation.goのdraftPageRefreshTriggerIDと同期する。
const REFRESH_TRIGGER_ID = "page-draft-refresh-trigger";

// DraftSaveRequestは、保存の要求で送ったタイトルと本文。応答を画面に反映するかは、送った内容と
// 今の入力を比べて決める。内容が同じ要求どうしは検証の結果も同じになるため、自動保存と手動保存の
// どちらが後に始まったかは問わない。
interface DraftSaveRequest {
  container: HTMLElement;
  title: string;
  body: string;
}

// 保存時刻の再取得を直前の保存の要求にひも付けるため、画面の要素ごとに最後に送った要求を持つ。
const latestRequests = new WeakMap<HTMLElement, DraftSaveRequest>();
const htmxRequests = new WeakMap<HtmxRequestContext, DraftSaveRequest>();

// beginDraftSaveは実際に送る値を記録する。手動保存はhtmxが組み立てたFormDataを渡し、
// キューで待った要求でも、送信時点の画面の値を保存したものと誤認しないようにする。
export function beginDraftSave(container: HTMLElement, data: FormData | URLSearchParams): DraftSaveRequest {
  const request = {
    container,
    title: String(data.get("title") ?? data.get("pages_edit_form[title]") ?? ""),
    body: String(data.get("body") ?? data.get("pages_edit_form[body]") ?? ""),
  };
  latestRequests.set(container, request);
  return request;
}

// isCurrentDraftSaveは、要求で送った本文が今の本文と同じかを返す。保存の成否とエラーの表示は
// 本文だけで決まるため、タイトルは比べない。タイトルの入力では自動保存が始まらないため、
// タイトルを変えたあとに届いた応答が、表示を直す最後の応答になることがある。次の自動保存が
// debounceで待っているあいだの入力も、送った本文との比較で検出する。
export function isCurrentDraftSave(request: DraftSaveRequest): boolean {
  const { container } = request;
  const body = findInput<HTMLTextAreaElement>(container.dataset.markdownEditorTextarea);
  return container.isConnected && body?.value === request.body;
}

// matchesCurrentTitleは、要求で送ったタイトルが今のタイトルと同じかを返す。
function matchesCurrentTitle(request: DraftSaveRequest): boolean {
  const title = findInput<HTMLInputElement>(request.container.dataset.markdownEditorTitle);
  return title?.value === request.title;
}

function findInput<T extends HTMLElement>(selector: string | undefined): T | null {
  return selector ? document.querySelector<T>(selector) : null;
}

// showSaveErrorは、保存の応答が利用者の直せる失敗 (本文が上限を超えた) のときにエラーを出し、
// 出したかどうかを返す。422はサーバーが返した文面を、413は本文が長すぎる旨を出す。
// それ以外の失敗は、利用者が直せるものではないため表示を変えない。
export function showSaveError(kind: SaveKind, status: number, responseText: string): boolean {
  const statusElement = document.querySelector<HTMLElement>(STATUS_SELECTOR);
  const errorElement = statusElement?.querySelector<HTMLElement>(ERROR_SELECTOR);
  if (!statusElement || !errorElement) return false;

  const reason = failureReason(statusElement, status, responseText);
  if (!reason) return false;

  const lead =
    kind === "autosave" ? statusElement.dataset.autosaveFailedMessage : statusElement.dataset.saveDraftFailedMessage;
  const text = [lead, reason].filter(Boolean).join(" ");

  // 同じ文面を書き直すと、スクリーンリーダーが同じ内容を繰り返し読み上げるため、変わったときだけ書く
  if (errorElement.textContent !== text) {
    errorElement.textContent = text;
  }
  notifyBodyValidation(true);
  return true;
}

// clearSaveErrorは、保存に成功したときにエラーを消し、保存時刻の表示に戻す。
export function clearSaveError(): void {
  const errorElement = document.querySelector<HTMLElement>(`${STATUS_SELECTOR} ${ERROR_SELECTOR}`);
  if (errorElement && errorElement.textContent !== "") {
    errorElement.textContent = "";
  }
  notifyBodyValidation(false);
}

// CodeMirrorの属性はエディタ側で更新する。保存の成否と本文のエラー状態を同じタイミングで伝える。
function notifyBodyValidation(invalid: boolean): void {
  document
    .querySelector(EDITOR_SELECTOR)
    ?.dispatchEvent(new CustomEvent("draft-save-validation", { detail: { invalid } }));
}

// initializeSaveStatusは、下書き保存のボタンが送ったhtmxの要求の結果で、エラーの表示を切り替える。
// ボタンはhx-swap="none" のため、失敗の応答はhtmxが画面に入れず、ここで表示する。
export function initializeSaveStatus(): void {
  document.addEventListener("htmx:before:request", handleBeforeRequest);
  document.addEventListener("htmx:after:request", handleAfterRequest);
}

// hxは応答のHX-* ヘッダーで、htmxがHX-Redirectの移動先をredirectに入れる。
interface HtmxRequestContext {
  request?: { body?: FormData | URLSearchParams | null };
  response?: { status: number };
  hx?: { redirect?: string };
  text?: string;
}

function handleBeforeRequest(event: Event): void {
  const source = event.target;
  if (!(source instanceof Element)) return;
  const ctx = (event as CustomEvent<{ ctx?: HtmxRequestContext }>).detail?.ctx;
  const container = document.querySelector<HTMLElement>(EDITOR_SELECTOR);
  if (!ctx || !container) return;

  if (source.hasAttribute(SAVE_BUTTON_ATTRIBUTE) && ctx.request?.body) {
    htmxRequests.set(ctx, beginDraftSave(container, ctx.request.body));
  } else if (source.id === REFRESH_TRIGGER_ID) {
    // 自動保存後のGETも元の保存要求にひも付け、遅い再取得が保存時刻を巻き戻すのを防ぐ。
    const request = latestRequests.get(container);
    if (request) htmxRequests.set(ctx, request);
  }
}

function handleAfterRequest(event: Event): void {
  const source = event.target;
  if (!(source instanceof Element)) return;

  const ctx = (event as CustomEvent<{ ctx?: HtmxRequestContext }>).detail?.ctx;
  const request = ctx && htmxRequests.get(ctx);
  if (!request) return;
  if (!isCurrentDraftSave(request)) {
    // alpha8ではこのイベントを取り消すと、OOBスワップとHX-Redirectも行われない。
    event.preventDefault();
    return;
  }
  if (!source.hasAttribute(SAVE_BUTTON_ATTRIBUTE)) return;
  const status = ctx?.response?.status;
  if (status === undefined) return;

  if (status >= 200 && status < 300) {
    clearSaveError();
    // 編集提案の作成画面へは、送ったタイトルで保存した下書きを持って移る。送ったあとにタイトルを
    // 変えていたら、変えたタイトルを失わないよう移らずに留まる。
    if (ctx?.hx?.redirect && !matchesCurrentTitle(request)) {
      event.preventDefault();
    }
    return;
  }
  showSaveError("manual", status, ctx?.text ?? "");
}

function failureReason(statusElement: HTMLElement, status: number, responseText: string): string {
  if (status === 422) {
    return responseText.trim();
  }
  if (status === 413) {
    // 413は本文を読む前に送るときの上限で返るため、応答の本文 (エラーの画面) は使わない
    return statusElement.dataset.bodyTooLargeMessage ?? "";
  }
  return "";
}
