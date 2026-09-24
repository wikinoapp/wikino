import { stopAutosave } from "./markdown-editor";

// 編集画面の下書きアラートにある削除フォーム ([data-draft-delete-form]) の送信を引き取る。
// 確認ダイアログはフォーム自身のhx-on:submitが先に出し、キャンセルされたらsubmitイベントの
// defaultPreventedが立つ。確認されたときだけ送信を止め、自動保存を止めて送信中の保存を
// 待ってから改めて送信する。自動保存が削除の前後に走ると、削除した下書きが作り直されるため。
// イベント委譲により、要素ごとの結線を不要にしている。
export function initializeDraftDeleteForms(): void {
  document.addEventListener("submit", handleSubmit);
}

function handleSubmit(event: SubmitEvent): void {
  const form = event.target;
  if (!(form instanceof HTMLFormElement) || !form.hasAttribute("data-draft-delete-form")) {
    return;
  }
  if (event.defaultPrevented) {
    return;
  }

  event.preventDefault();

  // 待機中にボタンが再度押されて削除リクエストが重複しないようにする
  form.querySelectorAll<HTMLButtonElement>("button[type=submit]").forEach((button) => (button.disabled = true));

  // form.submit() はsubmitイベントを発火しないため、確認ダイアログとこのハンドラーを再度通らない
  void stopAutosave().then(() => {
    // 削除後に「戻る」でbfcacheから復元されると、削除済みの下書きを表示したまま自動保存が
    // 止まった画面が出るため、サーバーから取り直す
    window.addEventListener("pageshow", reloadIfRestoredFromCache, { once: true });
    form.submit();
  });
}

function reloadIfRestoredFromCache(event: PageTransitionEvent): void {
  if (event.persisted) {
    window.location.reload();
  }
}
