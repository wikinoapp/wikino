// Marks a sticky header while it is pinned to the top of the viewport. The page detail screen
// (pages/page/show.templ) renders a stable sentinel before [data-sticky-header] and a spacer after
// it. Once the sentinel passes the viewport's top edge, this module sets data-stuck on the header
// and gives the spacer the height lost when the title shrinks into a compact bar.
//
// Observing the sentinel rather than the variable-height header keeps the state independent of the
// title's wrapping and of the compact styles themselves. Its bottom crossing the observer root's
// top edge means the adjacent sticky header has reached top: 0; a sentinel below the viewport does
// not satisfy that geometric check even though its intersection ratio is also zero.
//
// [Ja] スティッキーヘッダーが画面上端に固定されている間、そのことを表す印を付ける。ページ表示画面
// (pages/page/show.templ) は [data-sticky-header] の前に寸法が変わらない sentinel、後ろに spacer を
// 描画する。sentinel がビューポート上端を通過すると、本モジュールはヘッダーへ data-stuck を付け、
// タイトルがコンパクトなバーへ縮んで失われた高さを spacer に設定する。
//
// 可変高のヘッダーではなく sentinel を監視することで、タイトルの折り返しやコンパクト化するスタイル
// 自体から状態判定を独立させる。sentinel の下端が observer root の上端を通過したことは、隣接する
// sticky ヘッダーが top: 0 に達したことを意味する。ビューポートより下にある sentinel も交差率は 0 だが、
// この位置判定は満たさない。

const SENTINEL_SELECTOR = "[data-sticky-header-sentinel]";
const HEADER_SELECTOR = "[data-sticky-header]";
const SPACER_SELECTOR = "[data-sticky-header-spacer]";

interface StickyHeaderElements {
  header: HTMLElement;
  spacer: HTMLElement;

  // Last measurement, kept so that crossing the sticky boundary repeatedly does not re-measure.
  //
  // [Ja] 直前の計測結果。固定境界を何度もまたいでも計測し直さないために保持する。
  measured?: MeasuredSpacing;
}

// The height the spacer has to hold, together with the header width it was measured at.
//
// [Ja] spacer に保持させる高さと、それを計測したときのヘッダー幅。
interface MeasuredSpacing {
  width: number;
  lostHeight: number;
}

let destroyStickyHeader: (() => void) | undefined;

// The server renders each sentinel, header, and spacer as adjacent siblings. Keeping that
// relationship structural avoids global IDs and lets a page carry more than one group safely.
//
// [Ja] サーバーは sentinel、header、spacer を隣接する兄弟要素として描画する。この構造上の関係を
// 使うことでグローバル ID を避け、1 ページに複数の組があっても安全に扱える。
export function initializeStickyHeader(): void {
  destroyStickyHeader?.();
  destroyStickyHeader = undefined;

  const elementsBySentinel = new Map<Element, StickyHeaderElements>();
  const sentinels = document.querySelectorAll<HTMLElement>(SENTINEL_SELECTOR);

  for (const sentinel of sentinels) {
    const header = sentinel.nextElementSibling;
    const spacer = header?.nextElementSibling;
    if (
      !(header instanceof HTMLElement) ||
      !header.matches(HEADER_SELECTOR) ||
      !(spacer instanceof HTMLElement) ||
      !spacer.matches(SPACER_SELECTOR)
    ) {
      continue;
    }

    elementsBySentinel.set(sentinel, { header, spacer });
  }

  if (elementsBySentinel.size === 0) {
    return;
  }

  const observer = new IntersectionObserver((entries) => handleIntersection(entries, elementsBySentinel), {
    // Pulling the root's top edge down by 1px makes the zero-area crossing occur at the sticky
    // header's top: 0 boundary. Geometry, rather than the ratio alone, determines which side
    // crossed.
    //
    // [Ja] root の上端を 1px 下げ、面積が 0 になる交差を sticky ヘッダーの top: 0 境界に
    // 合わせる。どちら側から交差したかは比率だけでなく位置から判定する。
    rootMargin: "-1px 0px 0px 0px",
    threshold: 0,
  });

  for (const sentinel of elementsBySentinel.keys()) {
    observer.observe(sentinel);
  }

  // Re-measure the expanded height after a viewport resize because a long title can wrap onto a
  // different number of lines. The cached measurement is dropped rather than compared: a resize
  // also moves the layout across the md breakpoint, which changes the height the compact bar
  // reserves at an unchanged header width. A header that is currently pinned is measured again
  // right away, in one animation frame, before paint.
  //
  // [Ja] ビューポートのリサイズ後は長いタイトルの折り返し行数が変わり得るため、展開時の高さを
  // 再計測する。キャッシュは比較せずに捨てる。リサイズは md ブレークポイントをまたぐこともあり、
  // その場合はヘッダー幅が同じでもコンパクトなバーが確保する高さが変わるためである。固定中の
  // ヘッダーはその場で、1 回の animation frame 内、描画前に計測し直す。
  let resizeFrame: number | undefined;
  const handleResize = (): void => {
    if (resizeFrame !== undefined) {
      window.cancelAnimationFrame(resizeFrame);
    }

    resizeFrame = window.requestAnimationFrame(() => {
      resizeFrame = undefined;
      for (const elements of elementsBySentinel.values()) {
        elements.measured = undefined;
        if (elements.header.hasAttribute("data-stuck")) {
          applyStuck(elements);
        }
      }
    });
  };
  window.addEventListener("resize", handleResize);

  destroyStickyHeader = () => {
    observer.disconnect();
    window.removeEventListener("resize", handleResize);
    if (resizeFrame !== undefined) {
      window.cancelAnimationFrame(resizeFrame);
    }
  };
}

function handleIntersection(
  entries: IntersectionObserverEntry[],
  elementsBySentinel: Map<Element, StickyHeaderElements>,
): void {
  for (const entry of entries) {
    const elements = elementsBySentinel.get(entry.target);
    if (!elements) {
      continue;
    }

    const rootTop = entry.rootBounds?.top ?? 0;
    setStuck(elements, entry.boundingClientRect.bottom <= rootTop);
  }
}

function setStuck(elements: StickyHeaderElements, stuck: boolean): void {
  if (stuck) {
    if (!elements.header.hasAttribute("data-stuck")) {
      applyStuck(elements);
    }
    return;
  }

  if (elements.header.hasAttribute("data-stuck")) {
    delete elements.header.dataset.stuck;
    elements.spacer.style.removeProperty("height");
  }
}

// Give the spacer the height the header loses from normal flow, then mark the header as pinned.
// The real header is not compacted until the spacer is ready, so no intermediate layout can move
// the scroll anchor back above the sentinel. Resize recalculation runs through here too, which is
// why the mark is set rather than assumed.
//
// The previous measurement is reused while the header keeps the same width. Crossing the sticky
// boundary back and forth is ordinary scrolling, and each measurement forces two layouts. A resize
// drops the cache outright, so this comparison only has to catch a width change that arrives
// without one, such as a scrollbar appearing.
//
// [Ja] ヘッダーが通常フローから失う高さを spacer に持たせてから、ヘッダーへ固定中の印を付ける。
// spacer の準備ができるまで実ヘッダーをコンパクト化しないため、中間レイアウトがスクロールアンカーを
// sentinel より上へ戻すことはない。リサイズ時の再計算もここを通るため、印は前提とせず毎回設定する。
//
// ヘッダーの幅が変わらない間は前回の計測結果を使い回す。固定境界の往復は通常のスクロールで起きるうえ、
// 1 回の計測で 2 回のレイアウトを強制するためである。リサイズ時はキャッシュを捨てるので、この比較が
// 拾うのはリサイズを伴わない幅の変化 (スクロールバーの出現など) だけになる。
function applyStuck(elements: StickyHeaderElements): void {
  const { header, spacer } = elements;
  const width = header.getBoundingClientRect().width;
  const cached = elements.measured;
  const measured = cached && cached.width === width ? cached : { width, lostHeight: measureLostHeight(header, width) };
  elements.measured = measured;

  if (measured.lostHeight > 0) {
    spacer.style.height = `${measured.lostHeight}px`;
  } else {
    spacer.style.removeProperty("height");
  }

  header.dataset.stuck = "";
}

// Measure off-flow clones in both states and return how much height normal flow gives up when the
// header compacts.
//
// [Ja] 通常フロー外の clone で両状態を計測し、ヘッダーがコンパクトになるときに通常フローが手放す
// 高さを返す。
function measureLostHeight(header: HTMLElement, width: number): number {
  const expandedHeight = measureHeaderClone(header, width, false);
  const compactHeight = measureHeaderClone(header, width, true);
  return Math.max(0, expandedHeight - compactHeight);
}

// A clone has the same responsive classes and inherited root variables as the real header, while
// fixed positioning removes it from document flow. IDs and the sticky-header marker are stripped
// so the short-lived clone cannot become a selector target; inert and aria-hidden keep its copied
// controls outside interaction and the accessibility tree.
//
// [Ja] clone は実ヘッダーと同じレスポンシブクラスと root 変数を継承し、fixed 配置により文書フロー
// から外れる。短時間存在する clone が selector の対象にならないよう ID と sticky-header の印を外し、
// inert と aria-hidden により複製された操作要素を対話とアクセシビリティツリーの対象外にする。
function measureHeaderClone(header: HTMLElement, width: number, stuck: boolean): number {
  const clone = header.cloneNode(true) as HTMLElement;
  clone.removeAttribute("data-sticky-header");
  if (stuck) {
    clone.dataset.stuck = "";
  } else {
    delete clone.dataset.stuck;
  }
  clone.querySelectorAll("[id]").forEach((element) => element.removeAttribute("id"));
  clone.inert = true;
  clone.setAttribute("aria-hidden", "true");
  clone.style.position = "fixed";
  clone.style.top = "0";
  clone.style.left = "0";
  clone.style.width = `${width}px`;
  clone.style.visibility = "hidden";
  clone.style.pointerEvents = "none";
  clone.style.zIndex = "-1";

  document.body.append(clone);
  try {
    return clone.getBoundingClientRect().height;
  } finally {
    clone.remove();
  }
}
