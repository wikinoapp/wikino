// initializePlatformはOSの種別を <html data-os> に記録し、キーボードショートカットの
// グリフを ⌘ (Mac) とCtrl (それ以外) でCSSが切り替えられるようにする。CodeMirrorのキーマップは
// Mod-* をMacでは ⌘、それ以外ではCtrlに解決する (event.metaKey || event.ctrlKeyを見ている) ため、
// グリフを固定すると非Macユーザーへ誤った案内になる。判定は読み込み時に1度だけ行い、表記
// コンポーネントは両方のグリフを描画して表示側をCSSに委ねるため、属性付与の前後でレイアウトは
// ずれない。
export function initializePlatform(): void {
  document.documentElement.dataset.os = isMac() ? "mac" : "other";
}

function isMac(): boolean {
  // navigator.platformは非推奨だが、ブラウザ横断で最も簡潔で確実なシグナルのため使う。空の
  // ときはユーザーエージェント文字列にフォールバックする。
  const platform = navigator.platform || navigator.userAgent || "";
  return /mac/i.test(platform);
}
