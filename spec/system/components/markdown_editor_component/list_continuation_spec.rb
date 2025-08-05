# typed: false
# frozen_string_literal: true

require_relative "shared_helpers"

RSpec.describe "Markdownエディター/リスト記法の自動継続", type: :system do
  include MarkdownEditorHelpers

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
    fill_in_editor(text: "- [X] 完了タスク (大文字) ")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- [X] 完了タスク (大文字) \n- [ ] ")
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

  it "リスト行の行頭で改行した場合、リスト継続せずに通常の改行になること" do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- aaa")

    # カーソルを行頭に移動
    move_cursor_to_start

    # 行頭で改行
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("\n- aaa")
  end

  it "リスト行のマーカー直前で改行した場合、リスト継続せずに改行すること" do
    visit_page_editor
    clear_editor
    # インデント付きのリスト
    fill_in_editor(text: "  - bbb")

    # カーソルをマーカー直前 (インデントの後) に移動
    page.execute_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;

        // カーソルをマーカー直前 (位置2) に移動し、フォーカスする
        editorView.dispatch({
          selection: { anchor: 2, head: 2 }
        });
        editorView.focus();
      })();
    JS

    # マーカー直前で改行
    within ".cm-content" do
      current_scope.send_keys(:enter)
    end

    editor_content = get_editor_content
    expect(editor_content).to eq("  \n  - bbb")

    # カーソルが新しい行の行頭 (マーカーの直前) にあることを確認
    cursor_position = get_cursor_position
    expect(cursor_position[:line]).to eq(2)
    expect(cursor_position[:column]).to eq(2)
  end
end
