# typed: false
# frozen_string_literal: true

require_relative "shared_helpers"

RSpec.describe "Markdownエディター/タブキーによるインデント機能", type: :system do
  include MarkdownEditorHelpers

  it "タブキーを押すと半角スペース2つが挿入されること", js: true do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "テスト")
    press_tab_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("テスト  ")
  end

  it "行頭でタブキーを押すと行頭に半角スペース2つが挿入されること", js: true do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "テスト")
    # カーソルを行頭に移動
    move_cursor_to_start
    press_tab_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("  テスト")
  end

  it "Shift+タブキーを押すとインデントが削除されること", js: true do
    visit_page_editor
    clear_editor
    fill_in_editor(text: "  インデント付きテキスト")
    # カーソルを行頭に移動
    move_cursor_to_start
    press_shift_tab_in_editor

    editor_content = get_editor_content
    expect(editor_content).to eq("インデント付きテキスト")
  end

  it "リスト項目の先頭でタブキーを押すとネストされたリスト項目になること", js: true do
    visit_page_editor
    clear_editor
    # リスト項目を作成
    set_editor_content(text: "- a\n- ")

    # カーソルを2行目のマーカーの直後に移動
    set_cursor_position(line: 2, column: 2)

    # タブキーを押す
    press_tab_in_editor

    # 内容の確認
    editor_content = get_editor_content
    expect(editor_content).to eq("- a\n  - ")
  end

  it "タスクリスト項目の先頭でタブキーを押すとネストされたタスクリスト項目になること", js: true do
    visit_page_editor
    clear_editor
    # タスクリスト項目を作成
    set_editor_content(text: "- [ ] タスク1\n- [ ] ")

    # カーソルを2行目のタスクマーカーの直後に移動
    set_cursor_position(line: 2, column: 6)

    # タブキーを押す
    press_tab_in_editor

    # 内容の確認
    editor_content = get_editor_content
    expect(editor_content).to eq("- [ ] タスク1\n  - [ ] ")
  end

  it "既にインデントされたリスト項目でタブキーを押すとさらにネストされること", js: true do
    visit_page_editor
    clear_editor
    # インデント付きリスト項目を作成
    set_editor_content(text: "  - インデント1\n  - ")

    # カーソルを2行目のマーカーの直後に移動
    set_cursor_position(line: 2, column: 4)

    # タブキーを押す
    press_tab_in_editor

    # 内容の確認
    editor_content = get_editor_content
    expect(editor_content).to eq("  - インデント1\n    - ")
  end

  it "行選択 (改行文字を含む) でタブキーを押すと選択した行のみインデントが追加され、カーソル位置が正しく維持されること", js: true do
    visit_page_editor
    clear_editor
    # 複数行のテキストを作成
    set_editor_content(text: "- a\n- b\n- c")

    # 2行目を改行文字を含めて選択する (トリプルクリック相当)
    select_line_with_newline(2)
    press_tab_in_editor

    # 内容の確認
    editor_content = get_editor_content
    expect(editor_content).to eq("- a\n  - b\n- c")

    # カーソル位置の確認 (3行目の先頭にあることを確認)
    cursor_position = get_cursor_position
    expect(cursor_position[:line]).to eq(3)
    expect(cursor_position[:column]).to eq(0)
  end

  it "行選択 (改行文字を含む) でShift+タブキーを押すと選択した行のみインデントが削除され、カーソル位置が正しく維持されること", js: true do
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

    # カーソル位置の確認 (3行目の先頭にあることを確認)
    cursor_position = get_cursor_position
    expect(cursor_position[:line]).to eq(3)
    expect(cursor_position[:column]).to eq(0)
  end

  it "複数行選択 (改行文字を含む) でタブキーを押すと選択した行のみインデントが追加されること", js: true do
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

    # カーソル位置の確認 (4行目の先頭にあることを確認)
    cursor_position = get_cursor_position
    expect(cursor_position[:line]).to eq(4)
    expect(cursor_position[:column]).to eq(0)
  end

  it "行選択でタブキーを押すと選択範囲が維持されること", js: true do
    visit_page_editor
    clear_editor
    # 複数行のテキストを作成
    set_editor_content(text: "- a\n- b\n- c")

    # 2行目を改行文字を含めて選択する
    select_line_with_newline(2)
    press_tab_in_editor

    # 内容の確認
    editor_content = get_editor_content
    expect(editor_content).to eq("- a\n  - b\n- c")

    # 選択範囲の確認 (2行目が行の先頭から選択されていることを確認)
    selection_info = get_selection_info
    expect(selection_info[:has_selection]).to be true
    expect(selection_info[:from_line]).to eq(2)
    expect(selection_info[:from_column]).to eq(0) # 行の先頭から選択
    expect(selection_info[:to_line]).to eq(3)
    expect(selection_info[:to_column]).to eq(0)
  end

  it "行選択でタブキーを押した後、deleteキーで行全体が削除されること", js: true do
    visit_page_editor
    clear_editor
    # 複数行のテキストを作成
    set_editor_content(text: "- a\n- b\n- c")

    # 2行目を改行文字を含めて選択する
    select_line_with_newline(2)
    press_tab_in_editor

    # Deleteキーを押す
    within ".cm-content" do
      current_scope.send_keys(:delete)
    end

    # 内容の確認 (2行目が完全に削除されていること)
    editor_content = get_editor_content
    expect(editor_content).to eq("- a\n- c")
  end

  it "リスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされたリストアイテムになること", js: true do
    visit_page_editor
    clear_editor
    # リスト項目を作成
    set_editor_content(text: "- a\n- b")

    # カーソルを2行目のマーカーの直後（bの前）に移動
    set_cursor_position(line: 2, column: 2)

    # タブキーを押す
    press_tab_in_editor

    # 内容の確認
    editor_content = get_editor_content
    expect(editor_content).to eq("- a\n  - b")

    # カーソル位置の確認（新しいネストされたリストマーカーの直後）
    cursor_position = get_cursor_position
    expect(cursor_position[:line]).to eq(2)
    expect(cursor_position[:column]).to eq(4)
  end

  it "タスクリスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされたタスクリストアイテムになること", js: true do
    visit_page_editor
    clear_editor
    # タスクリスト項目を作成
    set_editor_content(text: "- [ ] タスク1\n- [ ] タスク2")

    # カーソルを2行目のタスクマーカーの直後（タスク2の前）に移動
    set_cursor_position(line: 2, column: 6)

    # タブキーを押す
    press_tab_in_editor

    # 内容の確認
    editor_content = get_editor_content
    expect(editor_content).to eq("- [ ] タスク1\n  - [ ] タスク2")

    # カーソル位置の確認（新しいネストされたタスクリストマーカーの直後）
    cursor_position = get_cursor_position
    expect(cursor_position[:line]).to eq(2)
    expect(cursor_position[:column]).to eq(8)
  end

  it "完了タスクリスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされた完了タスクリストアイテムになること", js: true do
    visit_page_editor
    clear_editor
    # 完了タスクリスト項目を作成
    set_editor_content(text: "- [ ] aaa\n- [x] bbb\n- [ ] ccc")

    # カーソルを2行目のタスクマーカーの直後（bbbの前）に移動
    set_cursor_position(line: 2, column: 6)

    # タブキーを押す
    press_tab_in_editor

    # 内容の確認
    editor_content = get_editor_content
    expect(editor_content).to eq("- [ ] aaa\n  - [x] bbb\n- [ ] ccc")

    # カーソル位置の確認（新しいネストされたタスクリストマーカーの直後）
    cursor_position = get_cursor_position
    expect(cursor_position[:line]).to eq(2)
    expect(cursor_position[:column]).to eq(8)
  end
end
