import { Controller } from "@hotwired/stimulus"

// 添付ファイルの署名付きURLを非同期で取得して置換するコントローラー
export default class extends Controller {
  static values = {
    csrfToken: String
  }

  declare csrfTokenValue: string

  connect() {
    this.loadAttachmentUrls()
  }

  async loadAttachmentUrls() {
    // ページ内の全ての添付ファイル要素を収集
    const attachmentElements = this.element.querySelectorAll('[data-attachment-id]')
    if (attachmentElements.length === 0) return

    // 添付ファイルIDを収集（重複を除去）
    const attachmentIds = Array.from(new Set(
      Array.from(attachmentElements).map(el => el.getAttribute('data-attachment-id')).filter(Boolean)
    ))

    if (attachmentIds.length === 0) return

    try {
      // バッチAPIで署名付きURLを取得
      const response = await fetch('/attachments/signed_urls', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': this.csrfTokenValue
        },
        body: JSON.stringify({ attachment_ids: attachmentIds })
      })

      if (!response.ok) {
        console.error('Failed to fetch signed URLs:', response.status)
        return
      }

      const data = await response.json()
      const signedUrls = data.signed_urls || {}

      // 各要素のURLを置換
      attachmentElements.forEach(element => {
        const attachmentId = element.getAttribute('data-attachment-id')
        if (!attachmentId) return

        const signedUrl = signedUrls[attachmentId]
        if (!signedUrl) return

        // 要素のタイプに応じて適切な属性を更新
        if (element.tagName === 'IMG') {
          // img要素のsrc属性を更新
          (element as HTMLImageElement).src = signedUrl
          // loading="lazy"を追加
          element.setAttribute('loading', 'lazy')
        } else if (element.tagName === 'VIDEO') {
          // video要素のsrc属性を更新
          (element as HTMLVideoElement).src = signedUrl
        } else if (element.tagName === 'A' && element.hasAttribute('data-attachment-link')) {
          // a要素のhref属性を更新
          (element as HTMLAnchorElement).href = signedUrl
        }

        // プレースホルダークラスを削除
        element.classList.remove('wikino-attachment-placeholder')
      })
    } catch (error) {
      console.error('Error loading attachment URLs:', error)
    }
  }
}