import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeFormSubmit } from "./form-submit";

// 送信の既定の動作 (画面の遷移) はhappy-domで起きないため、送信が止められたかをdefaultPreventedで見る。
function submit(form: HTMLFormElement): SubmitEvent {
  const event = new SubmitEvent("submit", { bubbles: true, cancelable: true });
  form.dispatchEvent(event);
  return event;
}

// happy-domはconfirmを持たないため、答えを決めたconfirmを差し込む。
function stubConfirm(answer: boolean): ReturnType<typeof vi.fn> {
  const confirm = vi.fn(() => answer);
  vi.stubGlobal("confirm", confirm);
  return confirm;
}

function form(): HTMLFormElement {
  return document.querySelector("form") as HTMLFormElement;
}

function submitButton(): HTMLButtonElement {
  return document.querySelector("button[type=submit]") as HTMLButtonElement;
}

describe("initializeFormSubmit", () => {
  beforeEach(() => {
    initializeFormSubmit();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    document.body.innerHTML = "";
  });

  describe("data-confirm", () => {
    beforeEach(() => {
      document.body.innerHTML = `
        <form method="POST" action="/drafts/1" data-confirm="この下書きを削除しますか?">
          <button type="submit">削除</button>
        </form>
      `;
    });

    it("確認の文言を出し、承認されたら送信を止めない", () => {
      const confirm = stubConfirm(true);

      const event = submit(form());

      expect(confirm).toHaveBeenCalledWith("この下書きを削除しますか?");
      expect(event.defaultPrevented).toBe(false);
    });

    it("キャンセルされたら送信を止める", () => {
      stubConfirm(false);

      expect(submit(form()).defaultPrevented).toBe(true);
    });

    // 文書のバブリングで送信を引き取るスクリプト (下書きの削除など) は、キャンセルをdefaultPreventedで
    // 見分けるため、それらより先に判定する。
    it("文書のバブリングのリスナーより先にキャンセルを伝える", () => {
      stubConfirm(false);
      let preventedBeforeBubble: boolean | undefined;
      const listener = (event: Event) => (preventedBeforeBubble = event.defaultPrevented);
      document.addEventListener("submit", listener);

      submit(form());
      document.removeEventListener("submit", listener);

      expect(preventedBeforeBubble).toBe(true);
    });
  });

  describe("data-disable-on-submit", () => {
    beforeEach(() => {
      document.body.innerHTML = `
        <form method="POST" action="/sign_in" data-disable-on-submit>
          <button type="submit">ログインする</button>
        </form>
      `;
    });

    it("送信を止めずに送信ボタンを無効にする", () => {
      const event = submit(form());

      expect(event.defaultPrevented).toBe(false);
      expect(submitButton().disabled).toBe(true);
    });
  });

  it("どちらの属性も無いフォームには触れない", () => {
    const confirm = stubConfirm(true);
    document.body.innerHTML = `
      <form method="POST" action="/pages">
        <button type="submit">保存</button>
      </form>
    `;

    const event = submit(form());

    expect(confirm).not.toHaveBeenCalled();
    expect(event.defaultPrevented).toBe(false);
    expect(submitButton().disabled).toBe(false);
  });

  it("確認でキャンセルされたら送信ボタンを無効にしない", () => {
    stubConfirm(false);
    document.body.innerHTML = `
      <form method="POST" action="/tokens/1" data-confirm="取り消しますか?" data-disable-on-submit>
        <button type="submit">取り消す</button>
      </form>
    `;

    submit(form());

    expect(submitButton().disabled).toBe(false);
  });
});
