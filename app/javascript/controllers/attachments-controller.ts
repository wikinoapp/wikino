import { Controller } from "@hotwired/stimulus"
import { destroy } from "@rails/request.js"

export default class extends Controller {
  static values = { 
    spaceIdentifier: String,
    deleteConfirmation: String
  }

  declare readonly spaceIdentifierValue: string
  declare readonly deleteConfirmationValue: string

  delete(event: Event) {
    const button = event.currentTarget as HTMLButtonElement
    const attachmentId = button.dataset.attachmentId
    const filename = button.dataset.filename

    if (!attachmentId) {
      console.error("No attachment ID found")
      return
    }

    // 削除確認ダイアログ
    if (!confirm(this.deleteConfirmationValue)) {
      return
    }

    // ボタンを無効化
    button.disabled = true

    // 削除リクエストを送信
    destroy(`/s/${this.spaceIdentifierValue}/settings/attachments/${attachmentId}`, {
      responseKind: "json"
    })
    .then((response: any) => {
      if (response.ok) {
        // 成功時は要素をフェードアウトして削除
        const attachmentElement = document.querySelector(`[data-attachment-id="${attachmentId}"]`)
        if (attachmentElement) {
          attachmentElement.classList.add("opacity-0", "transition-opacity", "duration-300")
          setTimeout(() => {
            attachmentElement.remove()
            
            // もし添付ファイルがなくなったら、メッセージを表示
            const attachmentsList = document.querySelector(".grid")
            if (attachmentsList && attachmentsList.children.length === 0) {
              attachmentsList.innerHTML = `
                <div class="text-center py-12">
                  <p class="text-muted-foreground">${this.noAttachmentsMessage()}</p>
                </div>
              `
            }
          }, 300)
        }
      } else {
        // エラー時はボタンを再度有効化
        button.disabled = false
        alert("ファイルの削除に失敗しました")
      }
    })
    .catch((error: any) => {
      console.error("Error deleting attachment:", error)
      button.disabled = false
      alert("ファイルの削除に失敗しました")
    })
  }


  private noAttachmentsMessage(): string {
    // 言語に応じてメッセージを切り替え
    const lang = document.documentElement.lang || "ja"
    return lang === "en" ? "No attachments found" : "添付ファイルが見つかりません"
  }
}