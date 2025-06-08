# typed: false
# frozen_string_literal: true

RSpec.describe "Markdownエディター", type: :system do
  describe "Wikiリンクの補完候補" do
    it "Wikiリンクの補完候補が表示されること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      page_record = create(:page_record, space_record:)
      topic_record = page_record.topic_record
      space_member_record = create(:space_member_record, space_record:, user_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      create(:page_record, :published, space_record:, topic_record:, title: "Other Page 1")
      create(:page_record, :published, space_record:, topic_record:, title: "Other Page 2")

      sign_in(user_record:)
      visit "/s/#{space_record.identifier}/pages/#{page_record.number}/edit"

      fill_in_editor(text: "[[Page")

      autocomplete_element = find(".cm-tooltip-autocomplete")
      visible_texts = autocomplete_element.find_css(".cm-completionLabel").map(&:visible_text)

      expect(visible_texts).to eq([
        "#{topic_record.name}/Other Page 2",
        "#{topic_record.name}/Other Page 1"
      ])
    end
  end

  describe "リスト記法の自動継続" do
    it "順序なしリスト記法を入力してEnterキーを押すと次の行にもリスト記法が追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- 最初のアイテム")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("- 最初のアイテム\n- ")
    end

    it "順序付きリスト記法を入力してEnterキーを押すと次の行に番号がインクリメントされたリスト記法が追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "1. 最初のアイテム")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("1. 最初のアイテム\n2. ")
    end

    it "空のリスト項目でEnterキーを押すとリスト記法が終了すること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- ")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("")
    end

    it "インデント付きリスト記法でEnterキーを押すとインデントが維持されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "  - インデント付きアイテム")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("  - インデント付きアイテム\n  - ")
    end

    it "異なるマーカー (*、+) でも正常に動作すること" do
      visit_page_editor
      clear_editor
      # * マーカーのテスト
      fill_in_editor(text: "* アスタリスクマーカー")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("* アスタリスクマーカー\n* ")

      # エディターをクリア
      clear_editor

      # + マーカーのテスト
      fill_in_editor(text: "+ プラスマーカー")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("+ プラスマーカー\n+ ")
    end

    it "GitHubタスクリスト記法 (未完了) を入力してEnterキーを押すと次の行にも未完了タスクが追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- [ ] 未完了タスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("- [ ] 未完了タスク\n- [ ] ")
    end

    it "GitHubタスクリスト記法 (完了) を入力してEnterキーを押すと次の行に未完了タスクが追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- [x] 完了タスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("- [x] 完了タスク\n- [ ] ")
    end

    it "GitHubタスクリスト記法 (完了・大文字X) を入力してEnterキーを押すと次の行に未完了タスクが追加されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- [X] 完了タスク（大文字）")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("- [X] 完了タスク（大文字）\n- [ ] ")
    end

    it "インデント付きタスクリスト記法でEnterキーを押すとインデントが維持されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "  - [ ] インデント付きタスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("  - [ ] インデント付きタスク\n  - [ ] ")
    end

    it "空のタスクリスト項目でEnterキーを押すとタスクリスト記法が終了すること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "- [ ] ")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("")
    end

    it "異なるマーカー (*、+) でもタスクリスト記法が正常に動作すること" do
      visit_page_editor
      clear_editor
      # * マーカーのテスト
      fill_in_editor(text: "* [ ] アスタリスクタスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("* [ ] アスタリスクタスク\n* [ ] ")

      # エディターをクリア
      clear_editor

      # + マーカーのテスト
      fill_in_editor(text: "+ [x] プラスタスク")
      press_enter_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("+ [x] プラスタスク\n+ [ ] ")
    end

    private def visit_page_editor
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      page_record = create(:page_record, space_record:)
      topic_record = page_record.topic_record
      space_member_record = create(:space_member_record, space_record:, user_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)

      sign_in(user_record:)
      visit "/s/#{space_record.identifier}/pages/#{page_record.number}/edit"
    end
  end

  describe "タブキーによるインデント機能" do
    it "タブキーを押すと半角スペース2つが挿入されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "テスト")
      press_tab_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("テスト  ")
    end

    it "行頭でタブキーを押すと行頭に半角スペース2つが挿入されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "テスト")
      # カーソルを行頭に移動
      move_cursor_to_start
      press_tab_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("  テスト")
    end

    it "Shift+タブキーを押すとインデントが削除されること" do
      visit_page_editor
      clear_editor
      fill_in_editor(text: "  インデント付きテキスト")
      # カーソルを行頭に移動
      move_cursor_to_start
      press_shift_tab_in_editor

      editor_content = get_editor_content
      expect(editor_content).to eq("インデント付きテキスト")
    end

    it "行選択（改行文字を含む）でタブキーを押すと選択した行のみインデントが追加され、カーソル位置が正しく維持されること" do
      visit_page_editor
      clear_editor
      # 複数行のテキストを作成
      set_editor_content(text: "- a\n- b\n- c")
      
      # 2行目を改行文字を含めて選択する（トリプルクリック相当）
      select_line_with_newline(2)
      press_tab_in_editor

      # 内容の確認
      editor_content = get_editor_content
      expect(editor_content).to eq("- a\n  - b\n- c")
      
      # カーソル位置の確認（3行目の先頭にあることを確認）
      cursor_position = get_cursor_position
      expect(cursor_position[:line]).to eq(3)
      expect(cursor_position[:column]).to eq(0)
    end

    it "行選択（改行文字を含む）でShift+タブキーを押すと選択した行のみインデントが削除され、カーソル位置が正しく維持されること" do
      visit_page_editor
      clear_editor
      # インデント付きの複数行テキストを作成
      set_editor_content(text: "- a\n  - b\n- c")
      
      # 2行目を改行文字を含めて選択する
      select_line_with_newline(2)
      press_shift_tab_in_editor

      # 内容の確認
      editor_content = get_editor_content
      expect(editor_content).to eq("- a\n- b\n- c")
      
      # カーソル位置の確認（3行目の先頭にあることを確認）
      cursor_position = get_cursor_position
      expect(cursor_position[:line]).to eq(3)
      expect(cursor_position[:column]).to eq(0)
    end

    it "複数行選択（改行文字を含む）でタブキーを押すと選択した行のみインデントが追加されること" do
      visit_page_editor
      clear_editor
      # 複数行のテキストを作成
      set_editor_content(text: "行1\n行2\n行3\n行4")
      
      # 2〜3行目を改行文字を含めて選択する
      select_multiple_lines_with_newline(2, 3)
      press_tab_in_editor

      # 内容の確認
      editor_content = get_editor_content
      expect(editor_content).to eq("行1\n  行2\n  行3\n行4")
      
      # カーソル位置の確認（4行目の先頭にあることを確認）
      cursor_position = get_cursor_position
      expect(cursor_position[:line]).to eq(4)
      expect(cursor_position[:column]).to eq(0)
    end


    private def visit_page_editor
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record)
      page_record = create(:page_record, space_record:)
      topic_record = page_record.topic_record
      space_member_record = create(:space_member_record, space_record:, user_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)

      sign_in(user_record:)
      visit "/s/#{space_record.identifier}/pages/#{page_record.number}/edit"
    end
  end

  private def fill_in_editor(text:)
    within ".cm-content" do
      current_scope.click
      current_scope.send_keys(text)
    end
  end

  private def set_editor_content(text:)
    # CodeMirrorの状態を直接操作してテキストを設定（Tabテスト用）
    page.execute_script(
      "arguments[0].cmView.view.dispatch({ changes: { from: 0, to: arguments[0].cmView.view.state.doc.length, insert: arguments[1] } });",
      find(".cm-content"),
      text
    )
  end

  private def press_enter_in_editor
    within ".cm-content" do
      current_scope.send_keys(:enter)
    end
  end

  private def get_editor_content
    # CodeMirrorエディターのコンテンツを取得（隠しtextareaから）
    page.evaluate_script("document.querySelector('[data-markdown-editor-target=\"textarea\"]').value")
  end

  private def clear_editor
    # エディターの内容をすべてクリア
    within ".cm-content" do
      current_scope.send_keys([:control, "a"])
      current_scope.send_keys(:delete)
    end
  end

  private def press_tab_in_editor
    within ".cm-content" do
      current_scope.send_keys(:tab)
    end
  end

  private def press_shift_tab_in_editor
    within ".cm-content" do
      current_scope.send_keys([:shift, :tab])
    end
  end

  private def move_cursor_to_start
    within ".cm-content" do
      current_scope.send_keys([:control, :home])
    end
  end

  private def select_all_in_editor
    within ".cm-content" do
      current_scope.send_keys([:control, "a"])
    end
  end

  private def select_line_with_newline(line_number)
    # 特定の行を改行文字を含めて選択するJavaScript
    page.execute_script("(function() { var editor = document.querySelector('.cm-content'); var editorView = editor.cmView.view; var doc = editorView.state.doc; var line = doc.line(#{line_number}); var nextLineStart = #{line_number} < doc.lines ? doc.line(#{line_number} + 1).from : doc.length; editorView.dispatch({ selection: { anchor: line.from, head: nextLineStart } }); editorView.focus(); })();")
  end

  private def select_multiple_lines_with_newline(start_line, end_line)
    # 複数行を改行文字を含めて選択するJavaScript
    page.execute_script("(function() { var editor = document.querySelector('.cm-content'); var editorView = editor.cmView.view; var doc = editorView.state.doc; var startLine = doc.line(#{start_line}); var endLine = doc.line(#{end_line}); var nextLineStart = #{end_line} < doc.lines ? doc.line(#{end_line} + 1).from : doc.length; editorView.dispatch({ selection: { anchor: startLine.from, head: nextLineStart } }); editorView.focus(); })();")
  end

  private def get_cursor_position
    # カーソル位置を取得するJavaScript
    position = page.evaluate_script("(function() { var editor = document.querySelector('.cm-content'); var editorView = editor.cmView.view; var cursor = editorView.state.selection.main.head; var line = editorView.state.doc.lineAt(cursor); return { line: line.number, column: cursor - line.from, absolutePosition: cursor }; })();")
    
    {
      line: position["line"],
      column: position["column"],
      absolute_position: position["absolutePosition"]
    }
  end
end
