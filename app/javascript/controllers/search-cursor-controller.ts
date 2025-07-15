import { Controller } from "@hotwired/stimulus";

// space:フィルターがある場合にカーソル位置を調整するコントローラー
export default class extends Controller<HTMLInputElement> {
  static values = { hasSpaceFilter: Boolean };

  declare hasSpaceFilterValue: boolean;

  connect() {
    if (this.hasSpaceFilterValue) {
      this.positionCursorAfterSpaceFilters();
    }
  }

  // space:フィルターの後にカーソルを移動し、必要に応じてスペースを追加
  private positionCursorAfterSpaceFilters() {
    const input = this.element;
    const value = input.value;

    if (!value) return;

    // space:パターンをすべて検索
    const spaceMatches = value.match(/space:\S+/g);
    if (!spaceMatches) return;

    // space:フィルター以外のキーワードがあるかチェック
    const valueWithoutSpaceFilters = value.replace(/space:\S+/g, "").trim();
    if (valueWithoutSpaceFilters.length > 0) {
      // 検索キーワードがある場合はデフォルトのカーソル位置（先頭）にする
      input.setSelectionRange(0, 0);
      return;
    }

    // 最後のspace:フィルターの終了位置を計算
    let lastSpaceFilterEnd = 0;
    spaceMatches.forEach((match) => {
      const index = value.indexOf(match, lastSpaceFilterEnd);
      if (index !== -1) {
        lastSpaceFilterEnd = index + match.length;
      }
    });

    // space:フィルターの後にスペースがない場合は追加
    if (lastSpaceFilterEnd < value.length && value[lastSpaceFilterEnd] !== " ") {
      const newValue = value.slice(0, lastSpaceFilterEnd) + " " + value.slice(lastSpaceFilterEnd);
      input.value = newValue;
      lastSpaceFilterEnd += 1;
    } else if (lastSpaceFilterEnd === value.length) {
      // space:フィルターで終わっている場合はスペースを追加
      input.value = value + " ";
      lastSpaceFilterEnd += 1;
    } else {
      // 既にスペースがある場合はそのスペースの後に移動
      lastSpaceFilterEnd += 1;
    }

    // カーソル位置を設定
    input.setSelectionRange(lastSpaceFilterEnd, lastSpaceFilterEnd);
  }
}
