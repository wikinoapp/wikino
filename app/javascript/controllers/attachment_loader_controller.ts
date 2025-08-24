import { Controller } from "@hotwired/stimulus";
import { post } from "@rails/request.js";

// 添付ファイルの署名付きURLを非同期で取得して置換するコントローラー
export default class extends Controller {
  connect() {
    this.loadAttachmentUrls();
  }

  async loadAttachmentUrls() {
    // ページ内の全ての添付ファイル要素を収集
    const attachmentElements = this.element.querySelectorAll("[data-attachment-id]");
    if (attachmentElements.length === 0) return;

    // 添付ファイルIDを収集（重複を除去）
    const attachmentIds = Array.from(
      new Set(
        Array.from(attachmentElements)
          .map((el) => el.getAttribute("data-attachment-id"))
          .filter(Boolean),
      ),
    );

    if (attachmentIds.length === 0) return;

    try {
      // テスト環境ではテスト用エンドポイントを使用
      const endpoint = (this.element as HTMLElement).dataset.testEndpoint || "/attachments/signed_urls";
      
      // バッチAPIで署名付きURLを取得
      const response = await post(endpoint, {
        body: { attachment_ids: attachmentIds },
        responseKind: "json",
      });

      if (!response.ok) {
        console.error("Failed to fetch signed URLs:", response.status);
        return;
      }

      const data = (await response.json) as { signed_urls: Record<string, string> };
      const signedUrls = data.signed_urls || {};

      // 各要素のURLを置換
      attachmentElements.forEach((element) => {
        const attachmentId = element.getAttribute("data-attachment-id");
        if (!attachmentId) return;

        const signedUrl = signedUrls[attachmentId];
        if (!signedUrl) return;

        // 要素のタイプに応じて適切な属性を更新
        if (element.tagName === "IMG") {
          const imgElement = element as HTMLImageElement;

          // 画像のロード完了を待ってからフェードイン
          imgElement.onload = () => {
            imgElement.classList.remove("wikino-attachment-image");
            imgElement.classList.add("wikino-attachment-image-loaded");
          };

          // loading="lazy"を追加
          imgElement.setAttribute("loading", "lazy");
          // img要素のsrc属性を更新
          imgElement.src = signedUrl;
        } else if (element.tagName === "VIDEO") {
          const videoElement = element as HTMLVideoElement;

          // 動画のロード完了を待ってからフェードイン
          videoElement.onloadeddata = () => {
            videoElement.classList.remove("wikino-attachment-video");
            videoElement.classList.add("wikino-attachment-video-loaded");
          };

          // video要素のsrc属性を更新
          videoElement.src = signedUrl;
        } else if (element.tagName === "A" && element.hasAttribute("data-attachment-link")) {
          // a要素のhref属性を更新
          (element as HTMLAnchorElement).href = signedUrl;
        }
      });
    } catch (error) {
      console.error("Error loading attachment URLs:", error);
    }
  }
}
