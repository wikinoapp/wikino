# typed: false
# frozen_string_literal: true

require_relative "shared_helpers"

RSpec.describe "Markdownエディター/リスト記法の自動継続", type: :system do
  include MarkdownEditorHelpers

  it "順序なしリスト記法を入力してEnterキーを押すと次の行にもリスト記法が追加されること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- 最初のアイテム")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- 最初のアイテム\n- ")
  end

  it "順序付きリスト記法を入力してEnterキーを押すと次の行に番号がインクリメントされたリスト記法が追加されること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "1. 最初のアイテム")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("1. 最初のアイテム\n2. ")
  end

  it "空のリスト項目でEnterキーを押すとリスト記法が終了すること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- ")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("")
  end

  it "インデント付きリスト記法でEnterキーを押すとインデントが維持されること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "  - インデント付きアイテム")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("  - インデント付きアイテム\n  - ")
  end

  it "異なるマーカー (*、+) でも正常に動作すること", :js do
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

  it "GitHubタスクリスト記法 (未完了) を入力してEnterキーを押すと次の行にも未完了タスクが追加されること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- [ ] 未完了タスク")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- [ ] 未完了タスク\n- [ ] ")
  end

  it "GitHubタスクリスト記法 (完了) を入力してEnterキーを押すと次の行に未完了タスクが追加されること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- [x] 完了タスク")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- [x] 完了タスク\n- [ ] ")
  end

  it "GitHubタスクリスト記法 (完了・大文字X) を入力してEnterキーを押すと次の行に未完了タスクが追加されること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- [X] 完了タスク (大文字) ")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- [X] 完了タスク (大文字) \n- [ ] ")
  end

  it "インデント付きタスクリスト記法でEnterキーを押すとインデントが維持されること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "  - [ ] インデント付きタスク")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("  - [ ] インデント付きタスク\n  - [ ] ")
  end

  it "空のタスクリスト項目でEnterキーを押すとタスクリスト記法が終了すること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- [ ] ")
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("")
  end

  it "異なるマーカー (*、+) でもタスクリスト記法が正常に動作すること", :js do
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

  it "リスト行の行頭で改行した場合、リスト継続せずに通常の改行になること", :js do
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

  it "リスト行のマーカー直前で改行した場合、リスト継続せずに改行すること", :js do
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

  it "ネストされた空のリスト項目でEnterキーを押すと1階層上のリストを継続すること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- aaa")
    press_enter_in_editor
    # 自動で"- "が追加されるので、それを削除してインデント付きリストにする
    press_tab_in_editor
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- aaa\n- ")
  end

  it "2階層ネストされた空のリスト項目でEnterキーを押すと1階層上のリストを継続すること", :js do
    visit_page_editor
    clear_editor
    # CodeMirrorに直接設定
    set_editor_content(text: "- aaa\n  - bbb\n    - ")
    # カーソルを最後に設定
    page.execute_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;
        var doc = editorView.state.doc;
        var lastPos = doc.length;
        editorView.dispatch({
          selection: { anchor: lastPos, head: lastPos }
        });
        editorView.focus();
      })();
    JS
    # 空のリスト項目で改行
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- aaa\n  - bbb\n  - ")
  end

  it "ネストされたリストの継続と終了が正しく動作すること", :js do
    visit_page_editor
    clear_editor
    # CodeMirrorに直接設定（2階層のリストとその後の空項目）
    set_editor_content(text: "- aaa\n  - bbb\n  - ")
    # カーソルを最後に設定
    page.execute_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;
        var doc = editorView.state.doc;
        var lastPos = doc.length;
        editorView.dispatch({
          selection: { anchor: lastPos, head: lastPos }
        });
        editorView.focus();
      })();
    JS
    # 空のネストされたリスト項目で改行（1階層上に戻る）
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- aaa\n  - bbb\n- ")

    # 空のトップレベルリスト項目で改行（リスト終了）
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- aaa\n  - bbb\n")
  end

  it "ネストされたタスクリストの空項目でEnterキーを押すと1階層上のタスクリストを継続すること", :js do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "- [ ] タスク1")
    press_enter_in_editor
    press_tab_in_editor
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("- [ ] タスク1\n- [ ] ")
  end

  it "ネストされた順序付きリストの空項目でEnterキーを押すと1階層上の順序付きリストを継続すること", :js do
    visit_page_editor
    clear_editor
    # CodeMirrorに直接設定
    set_editor_content(text: "1. 項目1\n  1. ")
    # カーソルを最後に設定
    page.execute_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;
        var doc = editorView.state.doc;
        var lastPos = doc.length;
        editorView.dispatch({
          selection: { anchor: lastPos, head: lastPos }
        });
        editorView.focus();
      })();
    JS
    # 空のリスト項目で改行
    press_enter_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("1. 項目1\n1. ")
  end
end
