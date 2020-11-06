import { startCompletion } from '@codemirror/next/autocomplete';
import { EditorView, ViewPlugin, ViewUpdate } from "@codemirror/next/view"

export const autocompletionPlugin = ViewPlugin.fromClass(class {
  constructor(readonly view: EditorView) {}
  update(update: ViewUpdate) {}
}, {
  eventHandlers: {
    compositionend(this: {view: EditorView}) {
      // Continue to open the completion menu if enter key is pressed and IME menu is closed.
      startCompletion(this.view)
    }
  } as any
})
