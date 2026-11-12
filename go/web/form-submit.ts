// 普通に送信するフォーム (htmxでリクエストしないフォーム) に、送信前の確認と送信中のボタンの無効化を付ける。
//
// - data-confirm: 値の文言で確認を出し、キャンセルされたら送信を止める
// - data-disable-on-submit: 送信を始めたら送信ボタンを無効にし、二重送信を防ぐ
//
// 文書のキャプチャフェーズで受け、文書のバブリングで送信を引き取るほかのスクリプト
// (markdown-editor/draft-delete-handler.ts・publish-handler.ts) より先に判定する。それらは確認で
// キャンセルされた送信をdefaultPreventedで見分ける。

const DISABLE_ON_SUBMIT_ATTRIBUTE = "data-disable-on-submit";

let listening = false;

export function initializeFormSubmit(): void {
  // リスナーは文書に付くため、何度呼ばれても1つだけにする。
  if (listening) return;
  listening = true;

  document.addEventListener("submit", handleSubmit, { capture: true });
}

function handleSubmit(event: SubmitEvent): void {
  const form = event.target;
  if (!(form instanceof HTMLFormElement)) return;

  const message = form.dataset.confirm;
  if (message && !window.confirm(message)) {
    event.preventDefault();
    return;
  }

  if (form.hasAttribute(DISABLE_ON_SUBMIT_ATTRIBUTE)) {
    form.querySelectorAll<HTMLButtonElement>("button[type=submit]").forEach((button) => (button.disabled = true));
  }
}
