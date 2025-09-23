import { Controller } from "@hotwired/stimulus";

export default class extends Controller {
  static targets = ["dialog", "trigger"];

  declare readonly dialogTarget: HTMLDialogElement;
  declare readonly triggerTarget: HTMLElement;

  connect() {
    // ダイアログはIDで直接参照される
  }

  open(event: Event) {
    console.log("!!!!!!!!!!!! open dialog");
    // event.preventDefault();

    // // フォームのデータを取得
    // const form = document.querySelector("#pages_edit_form") as HTMLFormElement;
    // if (form) {
    //   const titleInput = form.querySelector('input[name="pages_edit_form[title]"]') as HTMLInputElement;
    //   const bodyTextarea = form.querySelector('textarea[name="pages_edit_form[body]"]') as HTMLTextAreaElement;

    //   // ダイアログフォームにデータをセット
    //   const dialogForm = this.dialogContentTarget.querySelector("form") as HTMLFormElement;
    //   if (dialogForm) {
    //     const pageTitleInput = dialogForm.querySelector(
    //       'input[name="edit_suggestions_create_form[page_title]"]',
    //     ) as HTMLInputElement;
    //     const pageBodyInput = dialogForm.querySelector(
    //       'input[name="edit_suggestions_create_form[page_body]"]',
    //     ) as HTMLInputElement;

    //     if (pageTitleInput && titleInput) {
    //       pageTitleInput.value = titleInput.value;
    //     }
    //     if (pageBodyInput && bodyTextarea) {
    //       pageBodyInput.value = bodyTextarea.value;
    //     }
    //   }
    // }

    this.dialogTarget.showModal();
  }

  close(event?: Event) {
    if (event) {
      event.preventDefault();
    }
    this.dialogTarget.close();
  }

  handleSubmit(event: CustomEvent) {
    // Turboの送信が成功したらダイアログを閉じる
    const detail = event.detail;
    if (detail.success) {
      this.close();
    }
  }

  toggleNewSuggestionFields(event: Event) {
    const target = event.target as HTMLInputElement;

    if (target.value === "") {
      // 新しい編集提案を作成
      this.showNewSuggestionFields();
    } else {
      // 既存の編集提案に追加
      this.showExistingSuggestionFields();
    }
  }

  private showNewSuggestionFields() {
    const newFields = document.getElementById("new-suggestion-fields");
    const existingFields = document.getElementById("existing-suggestion-fields");

    if (newFields) {
      newFields.classList.remove("hidden");
    }
    if (existingFields) {
      existingFields.classList.add("hidden");
    }
  }

  private showExistingSuggestionFields() {
    const newFields = document.getElementById("new-suggestion-fields");
    const existingFields = document.getElementById("existing-suggestion-fields");

    if (newFields) {
      newFields.classList.add("hidden");
    }
    if (existingFields) {
      existingFields.classList.remove("hidden");
    }
  }
}
