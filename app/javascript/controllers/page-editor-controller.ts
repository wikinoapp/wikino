import { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap } from "@codemirror/autocomplete";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import {
  defaultHighlightStyle,
  syntaxHighlighting,
  indentOnInput,
  bracketMatching,
  foldGutter,
  foldKeymap,
} from "@codemirror/language";
import { searchKeymap, highlightSelectionMatches } from "@codemirror/search";
import { EditorState } from "@codemirror/state";
import {
  keymap,
  highlightSpecialChars,
  drawSelection,
  dropCursor,
  rectangularSelection,
  crosshairCursor,
  lineNumbers,
} from "@codemirror/view";
import { Controller } from "@hotwired/stimulus";
import { EditorView } from "codemirror";

import { wikilinkCompletions } from "../page-editor/wikilink-completions";
import { insertNewlineAndContinueList } from "../page-editor/list-continuation";

export default class PageEditorController extends Controller<HTMLDivElement> {
  static targets = ["codeMirror", "textarea"];
  static values = {
    spaceIdentifier: String,
    body: String,
    autofocus: Boolean,
  };

  declare readonly autofocusValue: boolean;
  declare readonly bodyValue: string;
  declare readonly spaceIdentifierValue: string;
  declare readonly codeMirrorTarget: HTMLDivElement;
  declare readonly textareaTarget: HTMLTextAreaElement;
  editorView: EditorView;

  connect() {
    this.initializeEditor();
  }

  async initializeEditor() {
    const state = EditorState.create({
      doc: this.bodyValue,
      extensions: [
        lineNumbers(),
        highlightSpecialChars(),
        history(),
        foldGutter(),
        drawSelection(),
        dropCursor(),
        EditorState.allowMultipleSelections.of(true),
        indentOnInput(),
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        bracketMatching(),
        closeBrackets(),
        autocompletion({ override: [await wikilinkCompletions(this.spaceIdentifierValue)] }),
        rectangularSelection(),
        crosshairCursor(),
        highlightSelectionMatches(),
        keymap.of([
          // リスト記法の自動継続キーマップを最優先で配置
          { key: "Enter", run: insertNewlineAndContinueList },
          ...closeBracketsKeymap,
          ...defaultKeymap,
          ...searchKeymap,
          ...historyKeymap,
          ...foldKeymap,
          ...completionKeymap,
        ]),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            this.textareaTarget.value = update.state.doc.toString();
            this.textareaTarget.dispatchEvent(new Event("input"));
          }
        }),
      ],
    });

    this.editorView = new EditorView({
      state,
      parent: this.codeMirrorTarget,
    });

    if (this.autofocusValue) {
      this.editorView.focus();
    }
  }

  disconnect() {
    if (this.editorView) {
      this.editorView.destroy();
    }
  }
}
