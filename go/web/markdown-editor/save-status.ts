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
  return true;
}

// clearSaveErrorは、保存に成功したときにエラーを消し、保存時刻の表示に戻す。
export function clearSaveError(): void {
  const errorElement = document.querySelector<HTMLElement>(`${STATUS_SELECTOR} ${ERROR_SELECTOR}`);
  if (errorElement && errorElement.textContent !== "") {
    errorElement.textContent = "";
  }
}

// initializeSaveStatusは、下書き保存のボタンが送ったhtmxの要求の結果で、エラーの表示を切り替える。
// ボタンはhx-swap="none" のため、失敗の応答はhtmxが画面に入れず、ここで表示する。
export function initializeSaveStatus(): void {
  document.addEventListener("htmx:after:request", handleAfterRequest);
}

interface HtmxRequestContext {
  response?: { status: number };
  text?: string;
}

function handleAfterRequest(event: Event): void {
  const source = event.target;
  if (!(source instanceof Element) || !source.hasAttribute(SAVE_BUTTON_ATTRIBUTE)) return;

  const ctx = (event as CustomEvent<{ ctx?: HtmxRequestContext }>).detail?.ctx;
  const status = ctx?.response?.status;
  if (status === undefined) return;

  if (status >= 200 && status < 300) {
    clearSaveError();
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
