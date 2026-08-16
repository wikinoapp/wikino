import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { initializeStickyHeader } from "./sticky-header";

// happy-dom performs no layout, so IntersectionObserver is replaced with a stub that records what
// it was asked to watch and hands back the callback. The tests then feed it the entries a browser
// would report, which is what lets the pinned / not-pinned rule be checked without a real viewport.
//
// [Ja] happy-dom はレイアウトを行わないため、IntersectionObserver は監視対象を記録してコールバックを
// 取り出せるスタブに差し替える。テストはブラウザが報告するのと同じ entries を流し込むため、実際の
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

// Build the sibling group pages/page/show.templ renders. The sentinel remains one pixel tall while
// the header changes height, and the spacer retains the height lost from normal flow.
//
// [Ja] pages/page/show.templ が描画する兄弟要素の組を構築する。ヘッダーの高さが変わっても sentinel は
// 1px のままで、通常フローから失われた高さは spacer が保持する。
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
    throw new Error("no IntersectionObserver was created");
  }
  return observer;
}

// Report the sentinel's bottom edge relative to the observer root's top edge. A sentinel below the
// viewport and one above it can both have a zero intersection ratio, so the production rule uses
// this geometry to distinguish them.
//
// [Ja] observer root の上端を基準に sentinel の下端位置を報告する。ビューポートより下と上の
// sentinel はどちらも交差率 0 になり得るため、本番の判定はこの位置で両者を区別する。
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
      throw new Error("no mocked header heights remain");
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

  it("marks the header only after the sentinel reaches the top and preserves the lost flow height", () => {
    document.body.innerHTML = headerMarkup("Page Title");
    mockHeaderHeights([240, 81]);
    initializeStickyHeader();

    report(lastObserver(), 1);

    expect(header().hasAttribute("data-stuck")).toBe(true);
    expect(spacer().style.height).toBe("159px");
  });

  it("does not mark a long header merely because the sentinel is below the viewport", () => {
    document.body.innerHTML = headerMarkup("Long Page Title");
    initializeStickyHeader();

    // A sentinel below the viewport has the same zero intersection area as one that passed above
    // it. Its positive bottom coordinate proves the header has not reached the sticky boundary.
    //
    // [Ja] ビューポートより下の sentinel は、上へ通過したものと同じく交差面積が 0 になる。正の下端
    // 座標により、ヘッダーが sticky 境界へまだ達していないことを判定できる。
    report(lastObserver(), 900);

    expect(header().hasAttribute("data-stuck")).toBe(false);
    expect(spacer().style.height).toBe("");
  });

  it("clears the mark and spacer when the sentinel returns below the top", () => {
    document.body.innerHTML = headerMarkup("Page Title");
    mockHeaderHeights([240, 81]);
    initializeStickyHeader();

    report(lastObserver(), 1);
    report(lastObserver(), 2);

    expect(header().hasAttribute("data-stuck")).toBe(false);
    expect(spacer().style.height).toBe("");
  });

  it("reuses the measured height while the header keeps its width", () => {
    document.body.innerHTML = headerMarkup("Page Title");
    // Only one pair of heights is mocked, so a second measurement would throw. Crossing the
    // boundary back and forth therefore has to reuse the first result.
    //
    // [Ja] 高さの組を 1 つだけ用意するため、2 回目の計測が走れば例外になる。固定境界を往復しても
    // 最初の結果を使い回すことがこれで分かる。
    mockHeaderHeights([240, 81]);
    initializeStickyHeader();

    report(lastObserver(), 1);
    report(lastObserver(), 2);
    report(lastObserver(), 1);

    expect(header().hasAttribute("data-stuck")).toBe(true);
    expect(spacer().style.height).toBe("159px");
  });

  it("re-measures the lost height after a stuck header is resized", () => {
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

  it("watches the sentinel against a root shrunk by 1px at the top", () => {
    document.body.innerHTML = headerMarkup("Page Title");
    initializeStickyHeader();

    // The 1px sentinel crosses a root whose top is inset by the same amount. Threshold zero asks
    // for the boundary crossing; the callback then uses its position to identify the side.
    //
    // [Ja] 1px の sentinel は、同じ量だけ上端を内側へ寄せた root と交差する。threshold 0 で境界通過の
    // 通知を受け、コールバックが位置からどちら側かを識別する。
    expect(lastObserver().options?.rootMargin).toBe("-1px 0px 0px 0px");
    expect(lastObserver().options?.threshold).toBe(0);
    expect(lastObserver().observed).toEqual([sentinel()]);
  });

  it("creates no observer without a complete sentinel, header, and spacer group", () => {
    document.body.innerHTML = `
      <div data-sticky-header-sentinel></div>
      <div data-sticky-header><h1>Page Title</h1></div>
    `;

    initializeStickyHeader();

    expect(FakeIntersectionObserver.instances).toEqual([]);
  });
});
