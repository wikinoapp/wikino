// initializePlatform records the OS family on <html data-os> so CSS can switch keyboard-shortcut
// glyphs between ⌘ (Mac) and Ctrl (other). The CodeMirror keymap resolves Mod-* to ⌘ on Mac and
// Ctrl elsewhere (it checks event.metaKey || event.ctrlKey), so a fixed glyph would mislead non-Mac
// users. Detection runs once on load and the hint components render both glyphs, leaving the visible
// one to CSS, so no layout shift occurs after the attribute is set.
//
// [Ja] initializePlatform は OS の種別を <html data-os> に記録し、キーボードショートカットの
// グリフを ⌘ (Mac) と Ctrl (それ以外) で CSS が切り替えられるようにする。CodeMirror のキーマップは
// Mod-* を Mac では ⌘、それ以外では Ctrl に解決する (event.metaKey || event.ctrlKey を見ている) ため、
// グリフを固定すると非 Mac ユーザーへ誤った案内になる。判定は読み込み時に 1 度だけ行い、表記
// コンポーネントは両方のグリフを描画して表示側を CSS に委ねるため、属性付与の前後でレイアウトは
// ずれない。
export function initializePlatform(): void {
  document.documentElement.dataset.os = isMac() ? "mac" : "other";
}

function isMac(): boolean {
  // navigator.platform is deprecated but still the simplest reliable signal across browsers; fall
  // back to the user agent string when it is empty.
  //
  // [Ja] navigator.platform は非推奨だが、ブラウザ横断で最も簡潔で確実なシグナルのため使う。空の
  // ときはユーザーエージェント文字列にフォールバックする。
  const platform = navigator.platform || navigator.userAgent || "";
  return /mac/i.test(platform);
}
