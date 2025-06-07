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
import { EditorState, Transaction } from "@codemirror/state";
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
          { key: "Enter", run: this.insertNewlineAndContinueList },
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

  // リスト記法のパターンを検出する正規表現
  private static readonly LIST_PATTERNS = {
    // 順序なしリスト: '- ', '* ', '+ '
    unordered: /^(\s*)([-*+])\s+(.*)$/,
    // 順序付きリスト: '1. ', '2. ', etc.
    ordered: /^(\s*)(\d+)\.\s+(.*)$/,
  };

  // 現在行のリスト記法を検出
  private detectListPattern(line: string): {
    type: "unordered" | "ordered" | null;
    indent: string;
    marker: string;
    content: string;
    number?: number;
  } | null {
    // 順序なしリストの検出
    const unorderedMatch = line.match(PageEditorController.LIST_PATTERNS.unordered);
    if (unorderedMatch) {
      return {
        type: "unordered",
        indent: unorderedMatch[1],
        marker: unorderedMatch[2],
        content: unorderedMatch[3],
      };
    }

    // 順序付きリストの検出
    const orderedMatch = line.match(PageEditorController.LIST_PATTERNS.ordered);
    if (orderedMatch) {
      return {
        type: "ordered",
        indent: orderedMatch[1],
        marker: orderedMatch[2],
        content: orderedMatch[3],
        number: parseInt(orderedMatch[2], 10),
      };
    }

    return null;
  }

  // 継続すべきリスト記法を生成
  private generateContinuationText(listInfo: ReturnType<typeof this.detectListPattern>): string {
    if (!listInfo) return "";

    if (listInfo.type === "unordered") {
      return `${listInfo.indent}${listInfo.marker} `;
    } else if (listInfo.type === "ordered" && listInfo.number !== undefined) {
      return `${listInfo.indent}${listInfo.number + 1}. `;
    }

    return "";
  }

  // Enterキーが押されたときの処理
  private insertNewlineAndContinueList = (view: EditorView): boolean => {
    const { state } = view;
    const { from, to } = state.selection.main;
    
    // 現在の行を取得
    const line = state.doc.lineAt(from);
    const lineText = line.text;
    
    // リスト記法を検出
    const listInfo = this.detectListPattern(lineText);
    
    if (!listInfo) {
      // リスト記法でない場合は通常の改行
      return false;
    }

    // 空のリスト項目の場合（マーカーのみでコンテンツがない）
    if (listInfo.content.trim() === "") {
      // リスト記法を削除して通常の改行
      const transaction = state.update({
        changes: {
          from: line.from,
          to: line.to,
          insert: listInfo.indent,
        },
        selection: { anchor: line.from + listInfo.indent.length },
      });
      view.dispatch(transaction);
      return true;
    }

    // リスト記法を継続
    const continuationText = this.generateContinuationText(listInfo);
    const insertText = `\n${continuationText}`;
    
    const transaction = state.update({
      changes: { from: to, insert: insertText },
      selection: { anchor: to + insertText.length },
    });
    
    view.dispatch(transaction);
    return true;
  };
}
