// APIリファレンス (/api/reference/v1) で、OpenAPI記述をRedocで描画する。
// Wikinoの画面とは別のページで読み込むため、main.jsとは別にバンドルする
import { createElement } from "react";
import { createRoot } from "react-dom/client";
import { RedocStandalone } from "redoc";

const container = document.getElementById("api-reference");
if (container?.dataset.specUrl) {
  // フォールバックはRedocの描画で置き換わる。OpenAPI記述を読み込めなかったときは、
  // Redocのエラー表示の前に戻してOpenAPI記述へのリンクを残す
  const fallback = document.getElementById("api-reference-fallback");
  createRoot(container).render(
    createElement(RedocStandalone, {
      specUrl: container.dataset.specUrl,
      onLoaded: (error) => {
        if (error && fallback) {
          container.before(fallback);
        }
      },
    }),
  );
}
