import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { whenDraftSavesSettled } from "./draft-version";
import { stopAutosave } from "./markdown-editor";
import { initializePublishForms } from "./publish-handler";

vi.mock("./markdown-editor", () => ({
  stopAutosave: vi.fn(),
}));
vi.mock("./draft-version", () => ({
  acceptConflictDigest: vi.fn(),
  whenDraftSavesSettled: vi.fn(),
}));

// pages/page/edit.templの公開のフォームと、版の項目を持たない編集提案の更新のフォームを写す
function formMarkup(): string {
  return `
    <div data-markdown-editor></div>
    <form id="publish-form" action="/s/test-space/pages/1" method="POST">
      <input type="hidden" name="draft_page_digest" value="v1" data-draft-page-digest-field>
      <button type="submit">Publish</button>
    </form>
    <form id="suggestion-form" action="/s/test-space/suggestions/1/pages/1" method="POST">
      <button type="submit">Update suggestion</button>
    </form>
  `;
}

describe("initializePublishForms", () => {
  let resolveStop: () => void;

  beforeAll(() => {
    // リスナーはdocumentに委譲するため、テスト全体で1度だけ登録する
    initializePublishForms();
  });

  beforeEach(() => {
    document.body.innerHTML = formMarkup();
    vi.mocked(stopAutosave).mockReturnValue(new Promise<void>((resolve) => (resolveStop = resolve)));
    vi.mocked(whenDraftSavesSettled).mockResolvedValue(undefined);
  });

  afterEach(() => {
    document.body.innerHTML = "";
    vi.restoreAllMocks();
    vi.mocked(stopAutosave).mockReset();
    vi.mocked(whenDraftSavesSettled).mockReset();
  });

  function submit(form: HTMLFormElement, cancelled = false): SubmitEvent {
    const event = new SubmitEvent("submit", { bubbles: true, cancelable: true });
    if (cancelled) {
      // フォーム自身のリスナーが先に送信を止めたときを写す
      form.addEventListener("submit", (e) => e.preventDefault(), { once: true });
    }
    form.dispatchEvent(event);
    return event;
  }

  // happy-domのPageTransitionEventは初期化オプションのpersistedを反映しないため、プロパティを直接与える
  function pageshowEvent(persisted: boolean): Event {
    return Object.defineProperty(new Event("pageshow"), "persisted", { value: persisted });
  }

  function publishForm(): HTMLFormElement {
    return document.getElementById("publish-form") as HTMLFormElement;
  }

  it("公開の送信後にbfcacheから復元されたら再読み込みする", async () => {
    const submitSpy = vi.spyOn(publishForm(), "submit").mockImplementation(() => {});
    const reloadSpy = vi.spyOn(window.location, "reload").mockImplementation(() => {});

    submit(publishForm());
    resolveStop();
    await vi.waitFor(() => expect(submitSpy).toHaveBeenCalledTimes(1));

    window.dispatchEvent(pageshowEvent(true));

    expect(reloadSpy).toHaveBeenCalledTimes(1);
  });

  it("bfcacheからの復元でなければ再読み込みしない", async () => {
    const submitSpy = vi.spyOn(publishForm(), "submit").mockImplementation(() => {});
    const reloadSpy = vi.spyOn(window.location, "reload").mockImplementation(() => {});

    submit(publishForm());
    resolveStop();
    await vi.waitFor(() => expect(submitSpy).toHaveBeenCalledTimes(1));

    window.dispatchEvent(pageshowEvent(false));

    expect(reloadSpy).not.toHaveBeenCalled();
  });

  it("送信が取り消されていたら自動保存を止めない", () => {
    const submitSpy = vi.spyOn(publishForm(), "submit").mockImplementation(() => {});

    submit(publishForm(), true);

    expect(stopAutosave).not.toHaveBeenCalled();
    expect(submitSpy).not.toHaveBeenCalled();
    expect(publishForm().querySelector("button")?.disabled).toBe(false);
  });

  it("版の項目を持たないフォーム (編集提案の更新) の送信には関与しない", () => {
    const form = document.getElementById("suggestion-form") as HTMLFormElement;

    const event = submit(form);

    expect(event.defaultPrevented).toBe(false);
    expect(stopAutosave).not.toHaveBeenCalled();
    expect(form.querySelector("button")?.disabled).toBe(false);
  });
});
