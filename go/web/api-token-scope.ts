// 公開APIのトークンのスコープを選ぶチェックボックス (components/api_token_scope.templ) に振る舞いを付ける。
//
// 書き込み・削除のスコープは同じリソースの読み取りを含む。含むスコープを選んだら、含まれるスコープを
// 選択済みにして無効にし、どのスコープに含まれるかの注記を出す。外したら利用者が選んでいた状態に戻す。
// 初めの状態はサーバーが描画するため、ここでは選択の変更に追従するだけにする。含む関係はサーバーが
// data属性で渡し、ここでは表を持たない。

const CHECKBOX_SELECTOR = "input[type=checkbox][data-api-token-scope]";
const IMPLIED_NOTE_SELECTOR = "[data-api-token-scope-implied-note]";

// 含まれて無効にする前に利用者が選んでいた状態。サーバーが無効にして描画したものは記録が無く、
// 外したときは選んでいない状態に戻す (無効なチェックボックスは送信されず、選んでいたかが分からないため)。
const checkedBeforeImplied = new WeakMap<HTMLInputElement, boolean>();

let listening = false;

export function initializeAPITokenScopes(): void {
  // リスナーは文書に付くため、何度呼ばれても1つだけにする。
  if (listening) return;
  listening = true;

  document.addEventListener("change", handleChange);
}

function handleChange(event: Event): void {
  const target = event.target;
  if (!(target instanceof HTMLInputElement) || !target.matches(CHECKBOX_SELECTOR)) return;

  update(groupOf(target));
}

// 同じフォームの同じ名前のチェックボックスを1つの並びとして扱う。
function groupOf(checkbox: HTMLInputElement): HTMLInputElement[] {
  const root = checkbox.form ?? document;
  return Array.from(root.querySelectorAll<HTMLInputElement>(CHECKBOX_SELECTOR)).filter(
    (element) => element.name === checkbox.name,
  );
}

function update(checkboxes: HTMLInputElement[]): void {
  const checkedImpliers = new Set(checkboxes.filter((c) => c.checked && !c.disabled).map((c) => c.value));
  const impliedBy = new Map<string, Set<string>>();
  for (const checkbox of checkboxes) {
    if (!checkedImpliers.has(checkbox.value)) continue;

    for (const implied of impliesOf(checkbox)) {
      const impliers = impliedBy.get(implied) ?? new Set<string>();
      impliers.add(checkbox.value);
      impliedBy.set(implied, impliers);
    }
  }

  for (const checkbox of checkboxes) {
    applyImplied(checkbox, impliedBy.get(checkbox.value) ?? new Set());
  }
}

function impliesOf(checkbox: HTMLInputElement): string[] {
  return (checkbox.dataset.apiTokenScopeImplies ?? "").split(" ").filter((scope) => scope !== "");
}

function applyImplied(checkbox: HTMLInputElement, impliers: Set<string>): void {
  const implied = impliers.size > 0;
  if (implied && !checkbox.disabled) {
    checkedBeforeImplied.set(checkbox, checkbox.checked);
    checkbox.checked = true;
    checkbox.disabled = true;
  } else if (!implied && checkbox.disabled) {
    checkbox.checked = checkedBeforeImplied.get(checkbox) ?? false;
    checkbox.disabled = false;
    checkedBeforeImplied.delete(checkbox);
  }

  updateNotes(checkbox, impliers);
}

// 注記は含みうるスコープごとにサーバーが描画しているため、表示を切り替える。隠した注記も
// aria-describedbyで参照すると読み上げられるため、表示している注記だけを参照させる。
function updateNotes(checkbox: HTMLInputElement, impliers: Set<string>): void {
  const container = checkbox.closest("label")?.parentElement;
  if (!container) return;

  const describedBy = new Set((checkbox.getAttribute("aria-describedby") ?? "").split(" ").filter((id) => id !== ""));
  for (const note of container.querySelectorAll<HTMLElement>(IMPLIED_NOTE_SELECTOR)) {
    const shown = impliers.has(note.dataset.apiTokenScopeImpliedNote ?? "");
    note.hidden = !shown;
    if (shown) {
      describedBy.add(note.id);
    } else {
      describedBy.delete(note.id);
    }
  }

  if (describedBy.size > 0) {
    checkbox.setAttribute("aria-describedby", Array.from(describedBy).join(" "));
  } else {
    checkbox.removeAttribute("aria-describedby");
  }
}
