import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeDraftDeleteForms } from "./draft-delete-handler";
import { stopAutosave } from "./markdown-editor";

vi.mock("./markdown-editor", () => ({
  stopAutosave: vi.fn(),
}));

// pages/page/edit.templの下書きアラート内の削除フォームを写す
function formMarkup(): string {
  return `
    <form data-draft-delete-form action="/s/test-space/pages/1/draft_page?redirect_to=page" method="POST">
      <input type="hidden" name="_method" value="DELETE">
      <button type="submit">Delete draft</button>
    </form>
    <form id="other-form" action="/other" method="POST">
      <button type="submit">Other</button>
    </form>
  `;
}

describe("initializeDraftDeleteForms", () => {
  let resolveStop: () => void;

  beforeAll(() => {
    // リスナーはdocumentに委譲するため、テスト全体で1度だけ登録する
    initializeDraftDeleteForms();
  });

  beforeEach(() => {
    document.body.innerHTML = formMarkup();
    vi.mocked(stopAutosave).mockReturnValue(new Promise<void>((resolve) => (resolveStop = resolve)));
  });

  afterEach(() => {
    document.body.innerHTML = "";
    vi.restoreAllMocks();
    vi.mocked(stopAutosave).mockReset();
  });

  function submit(form: HTMLFormElement, cancelled = false): SubmitEvent {
    const event = new SubmitEvent("submit", { bubbles: true, cancelable: true });
    if (cancelled) {
      // 確認ダイアログでキャンセルされたとき、フォーム自身のhx-on:submitが送信を止める
      form.addEventListener("submit", (e) => e.preventDefault(), { once: true });
    }
    form.dispatchEvent(event);
    return event;
  }

  // happy-domのPageTransitionEventは初期化オプションのpersistedを反映しないため、プロパティを直接与える
  function pageshowEvent(persisted: boolean): Event {
    return Object.defineProperty(new Event("pageshow"), "persisted", { value: persisted });
  }

  it("自動保存を止め終えてからフォームを送信する", async () => {
    const form = document.querySelector<HTMLFormElement>("[data-draft-delete-form]")!;
    const submitSpy = vi.spyOn(form, "submit").mockImplementation(() => {});

    const event = submit(form);

    expect(event.defaultPrevented).toBe(true);
    expect(stopAutosave).toHaveBeenCalledTimes(1);
    expect(submitSpy).not.toHaveBeenCalled();
    expect(form.querySelector("button")?.disabled).toBe(true);

    resolveStop();
    await vi.waitFor(() => expect(submitSpy).toHaveBeenCalledTimes(1));
  });

  it("削除の送信後にbfcacheから復元されたら再読み込みする", async () => {
    const form = document.querySelector<HTMLFormElement>("[data-draft-delete-form]")!;
    const submitSpy = vi.spyOn(form, "submit").mockImplementation(() => {});
    const reloadSpy = vi.spyOn(window.location, "reload").mockImplementation(() => {});

    submit(form);
    resolveStop();
    await vi.waitFor(() => expect(submitSpy).toHaveBeenCalledTimes(1));

    window.dispatchEvent(pageshowEvent(true));

    expect(reloadSpy).toHaveBeenCalledTimes(1);
  });

  it("bfcacheからの復元でなければ再読み込みしない", async () => {
    const form = document.querySelector<HTMLFormElement>("[data-draft-delete-form]")!;
    const submitSpy = vi.spyOn(form, "submit").mockImplementation(() => {});
    const reloadSpy = vi.spyOn(window.location, "reload").mockImplementation(() => {});

    submit(form);
    resolveStop();
    await vi.waitFor(() => expect(submitSpy).toHaveBeenCalledTimes(1));

    window.dispatchEvent(pageshowEvent(false));

    expect(reloadSpy).not.toHaveBeenCalled();
  });

  it("確認ダイアログでキャンセルされたら自動保存を止めない", () => {
    const form = document.querySelector<HTMLFormElement>("[data-draft-delete-form]")!;
    const submitSpy = vi.spyOn(form, "submit").mockImplementation(() => {});

    submit(form, true);

    expect(stopAutosave).not.toHaveBeenCalled();
    expect(submitSpy).not.toHaveBeenCalled();
    expect(form.querySelector("button")?.disabled).toBe(false);
  });

  it("削除フォーム以外の送信には関与しない", () => {
    const form = document.getElementById("other-form") as HTMLFormElement;

    const event = submit(form);

    expect(event.defaultPrevented).toBe(false);
    expect(stopAutosave).not.toHaveBeenCalled();
  });
});
