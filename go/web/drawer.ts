// Generic overlay drawer wiring for the drawer component (components/drawer.templ). It uses event
// delegation so any number of drawers (and drawers added to the DOM later) work without re-binding:
// clicking a [data-drawer-open] opens the matching drawer, and clicking the backdrop
// ([data-drawer-close]) or pressing Esc closes it. The open button's aria-expanded is kept in sync
// with the drawer's open state.
//
// [Ja] ドロワーコンポーネント (components/drawer.templ) のための汎用オーバーレイ式ドロワーの結線。
// イベント委譲を使うため、ドロワーが何個あっても (後から DOM に追加されても) 再バインド不要で動作する。
// [data-drawer-open] のクリックで対応するドロワーを開き、背景 ([data-drawer-close]) のクリックまたは
// Esc キーで閉じる。開くボタンの aria-expanded はドロワーの開閉状態に同期させる。

export function initializeDrawers(): void {
  document.addEventListener("click", handleClick);
  document.addEventListener("keydown", handleKeyDown);
}

function handleClick(event: MouseEvent): void {
  const target = event.target as Element | null;
  if (!target) {
    return;
  }

  const opener = target.closest("[data-drawer-open]");
  if (opener) {
    const id = opener.getAttribute("data-drawer-open");
    const drawer = id ? document.getElementById(id) : null;
    if (drawer) {
      openDrawer(drawer);
    }
    return;
  }

  const backdrop = target.closest("[data-drawer-close]");
  if (backdrop) {
    const drawer = backdrop.closest("[data-drawer]");
    if (drawer) {
      closeDrawer(drawer);
    }
  }
}

function handleKeyDown(event: KeyboardEvent): void {
  if (event.key !== "Escape") {
    return;
  }

  document.querySelectorAll('[data-drawer][aria-hidden="false"]').forEach(closeDrawer);
}

function openDrawer(drawer: Element): void {
  drawer.classList.remove("hidden");
  drawer.setAttribute("aria-hidden", "false");
  setOpenButtonsExpanded(drawer, true);
}

function closeDrawer(drawer: Element): void {
  drawer.classList.add("hidden");
  drawer.setAttribute("aria-hidden", "true");
  setOpenButtonsExpanded(drawer, false);
}

// setOpenButtonsExpanded reflects the drawer's open state on every open button that controls it,
// so assistive technologies announce whether the drawer is currently expanded.
//
// [Ja] setOpenButtonsExpanded はドロワーの開閉状態を、それを操作するすべての開くボタンに反映する。
// 支援技術がドロワーの開閉状態を読み上げられるようにするため。
function setOpenButtonsExpanded(drawer: Element, expanded: boolean): void {
  if (!drawer.id) {
    return;
  }

  document
    .querySelectorAll(`[data-drawer-open="${drawer.id}"]`)
    .forEach((opener) => opener.setAttribute("aria-expanded", String(expanded)));
}
