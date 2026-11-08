import { acceptConflictDigest, whenDraftSavesSettled } from "./draft-version";
import { stopAutosave } from "./markdown-editor";

// 編集画面の公開のフォーム (公開の前提にする版の項目 [data-draft-page-digest-field] を持つフォーム) の
// 送信を引き取る。公開は下書きの版を照合するため、送信中の保存が終わってその応答の版を前提にしてから
// 送る。保存の応答を待たずに公開すると、自分の直前の保存より古い版を前提にして競合してしまうためである。
// 自動保存は止めたまま再開しない。公開の後に自動保存が走ると、公開で消した下書きが作り直されるため。
// イベント委譲により、要素ごとの結線を不要にしている。

const PUBLISH_DIGEST_FIELD_SELECTOR = "[data-draft-page-digest-field]";
const EDITOR_SELECTOR = "[data-markdown-editor]";

export function initializePublishForms(): void {
  document.addEventListener("submit", handleSubmit);
}

function handleSubmit(event: SubmitEvent): void {
  const form = event.target;
  if (!(form instanceof HTMLFormElement) || !form.querySelector(PUBLISH_DIGEST_FIELD_SELECTOR)) {
    return;
  }
  if (event.defaultPrevented) {
    return;
  }
  const container = document.querySelector<HTMLElement>(EDITOR_SELECTOR);
  if (!container) {
    return;
  }

  event.preventDefault();

  // 待機中にボタンが再度押されて公開が重複しないようにする
  form.querySelectorAll<HTMLButtonElement>("button[type=submit]").forEach((button) => (button.disabled = true));

  // form.submit() はsubmitイベントを発火しないため、このハンドラーを再度通らない
  void stopAutosave()
    .then(() => whenDraftSavesSettled(container))
    .then(() => {
      // 公開後に「戻る」でbfcacheから復元されると、公開で消した下書きを表示したまま自動保存が
      // 止まった画面が出るため、サーバーから取り直す
      window.addEventListener("pageshow", reloadIfRestoredFromCache, { once: true });
      form.submit();
    });
}

// republishConflictedDraftは、公開が競合して再表示した編集画面で、競合で受け取った今の版を前提にして
// 画面の内容で公開し直す。利用者が「このまま上書きする」を選んだときに呼ぶ。
export function republishConflictedDraft(): void {
  const container = document.querySelector<HTMLElement>(EDITOR_SELECTOR);
  const field = document.querySelector<HTMLInputElement>(PUBLISH_DIGEST_FIELD_SELECTOR);
  if (!container || !field?.form) return;

  acceptConflictDigest(container);
  // requestSubmitはsubmitイベントを発火するため、送信ボタンの無効化と保存の完了待ちを通る
  field.form.requestSubmit();
}

function reloadIfRestoredFromCache(event: PageTransitionEvent): void {
  if (event.persisted) {
    window.location.reload();
  }
}
