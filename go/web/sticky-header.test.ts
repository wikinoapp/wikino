import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeStickyHeader } from "./sticky-header";

// happy-domはレイアウトを行わないため、IntersectionObserverは監視対象を記録してコールバックを
// 取り出せるスタブに差し替える。テストはブラウザが報告するのと同じentriesを流し込むため、実際の
// ビューポート無しで固定 / 非固定の判定を検証できる。
class FakeIntersectionObserver {
  static instances: FakeIntersectionObserver[] = [];

  readonly observed: Element[] = [];

  constructor(
    readonly callback: IntersectionObserverCallback,
    readonly options?: IntersectionObserverInit,
  ) {
    FakeIntersectionObserver.instances.push(this);
  }

  observe(element: Element): void {
    this.observed.push(element);
  }

  unobserve(): void {}

  disconnect(): void {}
}

// pages/page/show.templが描画する兄弟要素の組を構築する。ヘッダーの高さが変わってもsentinelは
// 1pxのままで、通常フローから失われた高さはspacerが保持する。
function headerMarkup(title: string): string {
  return `
    <div data-sticky-header-sentinel></div>
    <div class="group sticky top-0" data-sticky-header>
      <div>
        <h1>${title}</h1>
        <div><a href="/s/space/pages/1/edit">edit</a></div>
      </div>
    </div>
    <div data-sticky-header-spacer></div>
  `;
}

function lastObserver(): FakeIntersectionObserver {
  const observer = FakeIntersectionObserver.instances.at(-1);
  if (!observer) {
    throw new Error("IntersectionObserverが作成されていません");
  }
  return observer;
}

// observer rootの上端を基準にsentinelの下端位置を報告する。ビューポートより下と上の
// sentinelはどちらも交差率0になり得るため、本番の判定はこの位置で両者を区別する。
function report(observer: FakeIntersectionObserver, sentinelBottom: number, rootTop = 1): void {
  const entries = observer.observed.map(
    (target) =>
      ({
        target,
        boundingClientRect: { bottom: sentinelBottom },
        rootBounds: { top: rootTop },
      }) as IntersectionObserverEntry,
  );
  observer.callback(entries, observer as unknown as IntersectionObserver);
}

function sentinel(): HTMLElement {
  return document.querySelector("[data-sticky-header-sentinel]") as HTMLElement;
}

function header(): HTMLElement {
  return document.querySelector("[data-sticky-header]") as HTMLElement;
}

function spacer(): HTMLElement {
  return document.querySelector("[data-sticky-header-spacer]") as HTMLElement;
}

function mockHeaderHeights(...heightPairs: Array<readonly [expanded: number, compact: number]>): void {
  const sourceHeader = header();
  let pairIndex = 0;

  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (this: HTMLElement) {
    if (this === sourceHeader) {
      return { width: 320 } as DOMRect;
    }

    const heights = heightPairs[pairIndex];
    if (!heights) {
      throw new Error("モックに用意したヘッダーの高さが残っていません");
    }

    const compact = this.hasAttribute("data-stuck");
    if (compact) {
      pairIndex += 1;
    }
    return { width: 320, height: compact ? heights[1] : heights[0] } as DOMRect;
  });
}

describe("initializeStickyHeader", () => {
  beforeEach(() => {
    FakeIntersectionObserver.instances = [];
    vi.stubGlobal("IntersectionObserver", FakeIntersectionObserver);
    document.body.innerHTML = "";
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("sentinelが上端に達したときだけヘッダーに印を付け、通常フローから失われる高さを保持する", () => {
    document.body.innerHTML = headerMarkup("Page Title");
    mockHeaderHeights([240, 81]);
    initializeStickyHeader();

    report(lastObserver(), 1);

    expect(header().hasAttribute("data-stuck")).toBe(true);
    expect(spacer().style.height).toBe("159px");
  });

  it("sentinelがビューポートより下にあるだけでは長いヘッダーに印を付けない", () => {
    document.body.innerHTML = headerMarkup("Long Page Title");
    initializeStickyHeader();

    // ビューポートより下のsentinelは、上へ通過したものと同じく交差面積が0になる。正の下端
    // 座標により、ヘッダーがsticky境界へまだ達していないことを判定できる。
    report(lastObserver(), 900);

    expect(header().hasAttribute("data-stuck")).toBe(false);
    expect(spacer().style.height).toBe("");
  });

  it("sentinelが上端より下へ戻ると印とスペーサーの高さを解除する", () => {
    document.body.innerHTML = headerMarkup("Page Title");
    mockHeaderHeights([240, 81]);
    initializeStickyHeader();

    report(lastObserver(), 1);
    report(lastObserver(), 2);

    expect(header().hasAttribute("data-stuck")).toBe(false);
    expect(spacer().style.height).toBe("");
  });

  it("ヘッダーの幅が変わらない間は計測済みの高さを使い回す", () => {
    document.body.innerHTML = headerMarkup("Page Title");
    // 高さの組を1つだけ用意するため、2回目の計測が走れば例外になる。固定境界を往復しても
    // 最初の結果を使い回すことがこれで分かる。
    mockHeaderHeights([240, 81]);
    initializeStickyHeader();

    report(lastObserver(), 1);
    report(lastObserver(), 2);
    report(lastObserver(), 1);

    expect(header().hasAttribute("data-stuck")).toBe(true);
    expect(spacer().style.height).toBe("159px");
  });

  it("固定中のヘッダーがリサイズされると失われる高さを再計測する", () => {
    document.body.innerHTML = headerMarkup("Long Page Title");
    mockHeaderHeights([240, 81], [300, 81]);
    initializeStickyHeader();
    report(lastObserver(), 1);

    let resizeCallback: FrameRequestCallback | undefined;
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      resizeCallback = callback;
      return 1;
    });

    window.dispatchEvent(new Event("resize"));
    resizeCallback?.(0);

    expect(header().hasAttribute("data-stuck")).toBe(true);
    expect(spacer().style.height).toBe("219px");
  });

  it("上端を1px内側に寄せたrootを基準にsentinelを監視する", () => {
    document.body.innerHTML = headerMarkup("Page Title");
    initializeStickyHeader();

    // 1pxのsentinelは、同じ量だけ上端を内側へ寄せたrootと交差する。threshold 0で境界通過の
    // 通知を受け、コールバックが位置からどちら側かを識別する。
    expect(lastObserver().options?.rootMargin).toBe("-1px 0px 0px 0px");
    expect(lastObserver().options?.threshold).toBe(0);
    expect(lastObserver().observed).toEqual([sentinel()]);
  });

  it("sentinel・ヘッダー・スペーサーが揃っていなければ監視を開始しない", () => {
    document.body.innerHTML = `
      <div data-sticky-header-sentinel></div>
      <div data-sticky-header><h1>Page Title</h1></div>
    `;

    initializeStickyHeader();

    expect(FakeIntersectionObserver.instances).toEqual([]);
  });
});
