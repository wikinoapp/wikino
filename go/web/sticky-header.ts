// スティッキーヘッダーが画面上端に固定されている間、そのことを表す印を付ける。ページ表示画面
// (pages/page/show.templ) は [data-sticky-header] の前に寸法が変わらないsentinel、後ろにspacerを
// 描画する。sentinelがビューポート上端を通過すると、本モジュールはヘッダーへdata-stuckを付け、
// タイトルがコンパクトなバーへ縮んで失われた高さをspacerに設定する。
//
// 可変高のヘッダーではなくsentinelを監視することで、タイトルの折り返しやコンパクト化するスタイル
// 自体から状態判定を独立させる。sentinelの下端がobserver rootの上端を通過したことは、隣接する
// stickyヘッダーがtop: 0に達したことを意味する。ビューポートより下にあるsentinelも交差率は0だが、
// この位置判定は満たさない。

const SENTINEL_SELECTOR = "[data-sticky-header-sentinel]";
const HEADER_SELECTOR = "[data-sticky-header]";
const SPACER_SELECTOR = "[data-sticky-header-spacer]";

interface StickyHeaderElements {
  header: HTMLElement;
  spacer: HTMLElement;

  // 直前の計測結果。固定境界を何度もまたいでも計測し直さないために保持する。
  measured?: MeasuredSpacing;
}

// spacerに保持させる高さと、それを計測したときのヘッダー幅。
interface MeasuredSpacing {
  width: number;
  lostHeight: number;
}

let destroyStickyHeader: (() => void) | undefined;

// サーバーはsentinel、header、spacerを隣接する兄弟要素として描画する。この構造上の関係を
// 使うことでグローバルIDを避け、1ページに複数の組があっても安全に扱える。
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
    // rootの上端を1px下げ、面積が0になる交差をstickyヘッダーのtop: 0境界に
    // 合わせる。どちら側から交差したかは比率だけでなく位置から判定する。
    rootMargin: "-1px 0px 0px 0px",
    threshold: 0,
  });

  for (const sentinel of elementsBySentinel.keys()) {
    observer.observe(sentinel);
  }

  // ビューポートのリサイズ後は長いタイトルの折り返し行数が変わり得るため、展開時の高さを
  // 再計測する。キャッシュは比較せずに捨てる。リサイズはmdブレークポイントをまたぐこともあり、
  // その場合はヘッダー幅が同じでもコンパクトなバーが確保する高さが変わるためである。固定中の
  // ヘッダーはその場で、1回のanimation frame内、描画前に計測し直す。
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

// ヘッダーが通常フローから失う高さをspacerに持たせてから、ヘッダーへ固定中の印を付ける。
// spacerの準備ができるまで実ヘッダーをコンパクト化しないため、中間レイアウトがスクロールアンカーを
// sentinelより上へ戻すことはない。リサイズ時の再計算もここを通るため、印は前提とせず毎回設定する。
//
// ヘッダーの幅が変わらない間は前回の計測結果を使い回す。固定境界の往復は通常のスクロールで起きるうえ、
// 1回の計測で2回のレイアウトを強制するためである。リサイズ時はキャッシュを捨てるので、この比較が
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

// 通常フロー外のcloneで両状態を計測し、ヘッダーがコンパクトになるときに通常フローが手放す
// 高さを返す。
function measureLostHeight(header: HTMLElement, width: number): number {
  const expandedHeight = measureHeaderClone(header, width, false);
  const compactHeight = measureHeaderClone(header, width, true);
  return Math.max(0, expandedHeight - compactHeight);
}

// cloneは実ヘッダーと同じレスポンシブクラスとroot変数を継承し、fixed配置により文書フロー
// から外れる。短時間存在するcloneがselectorの対象にならないようIDとsticky-headerの印を外し、
// inertとaria-hiddenにより複製された操作要素を対話とアクセシビリティツリーの対象外にする。
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
