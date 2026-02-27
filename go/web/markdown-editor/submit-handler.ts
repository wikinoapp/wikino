import { EditorView } from "codemirror";

export function handleSubmitShortcut(view: EditorView): boolean {
  const editorElement = view.dom;
  const form = editorElement.closest("form");

  if (!form) {
    return false;
  }

  const submitButton = form.querySelector(
    'button[type="submit"]',
  ) as HTMLButtonElement;
  if (submitButton && !submitButton.disabled) {
    submitButton.click();
    return true;
  }

  return false;
}
