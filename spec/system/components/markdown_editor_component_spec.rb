# typed: false
# frozen_string_literal: true

RSpec.describe "Markdownエディター", type: :system do
  # ファイルアップロードのモックを設定するヘルパーメソッド
  # 注意: WebMockがActiveStorageと干渉するため、このヘルパーメソッドは現在使用されていません
  private def setup_file_upload_mocks(
    presign_result: nil,
    create_result: nil,
    upload_delay: nil,
    upload_error_count: 0
  )
    # サービスクラスをモック
    presign_service = instance_double(Attachments::CreatePresignedUploadService)
    allow(Attachments::CreatePresignedUploadService).to receive(:new).and_return(presign_service)

    # プリサインURLの結果をモック
    presign_result ||= instance_double(
      Attachments::CreatePresignedUploadService::Result,
      direct_upload_url: "https://r2.example.com/upload",
      direct_upload_headers: {"Content-Type" => "image/png"},
      blob_signed_id: "test-signed-id"
    )
    allow(presign_service).to receive(:call).and_return(presign_result)

    # アタッチメント作成サービスをモック
    create_service = instance_double(Attachments::CreateService)
    allow(Attachments::CreateService).to receive(:new).and_return(create_service)

    # アタッチメント作成の結果をモック
    if create_result.nil?
      blob = instance_double(ActiveStorage::Blob, signed_id: "test-signed-id")
      active_storage_attachment = instance_double(
        ActiveStorage::Attachment,
        blob: blob
      )
      attachment_record = instance_double(
        AttachmentRecord,
        id: "test-attachment-id",
        active_storage_attachment_record: active_storage_attachment
      )
      allow(attachment_record).to receive(:generate_signed_url).and_return("https://r2.example.com/test-image.png")
      create_result = instance_double(Attachments::CreateService::Result)
      allow(create_result).to receive(:attachment_record).and_return(attachment_record)
    end
    allow(create_service).to receive(:call).and_return(create_result)
  end
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

    it "リスト項目の先頭でタブキーを押すとネストされたリスト項目になること" do
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

    it "タスクリスト項目の先頭でタブキーを押すとネストされたタスクリスト項目になること" do
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

    it "既にインデントされたリスト項目でタブキーを押すとさらにネストされること" do
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

    it "行選択 (改行文字を含む) でタブキーを押すと選択した行のみインデントが追加され、カーソル位置が正しく維持されること" do
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

    it "行選択 (改行文字を含む) でShift+タブキーを押すと選択した行のみインデントが削除され、カーソル位置が正しく維持されること" do
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

    it "複数行選択 (改行文字を含む) でタブキーを押すと選択した行のみインデントが追加されること" do
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

    it "行選択でタブキーを押すと選択範囲が維持されること" do
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

    it "行選択でタブキーを押した後、deleteキーで行全体が削除されること" do
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

    it "リスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされたリストアイテムになること" do
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

    it "タスクリスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされたタスクリストアイテムになること" do
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

    it "完了タスクリスト項目のマーカー直後にテキストがある状態でタブキーを押すとネストされた完了タスクリストアイテムになること" do
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
    # CodeMirrorの状態を直接操作してテキストを設定 (Tabテスト用)
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
    # CodeMirrorエディターのコンテンツを取得 (隠しtextareaから)
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
    page.execute_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;
        var doc = editorView.state.doc;
        var line = doc.line(#{line_number});

        // 行の開始から次の行の開始まで選択（改行文字を含む）
        var nextLineStart = #{line_number} < doc.lines
          ? doc.line(#{line_number} + 1).from
          : doc.length;

        editorView.dispatch({
          selection: { anchor: line.from, head: nextLineStart }
        });

        editorView.focus();
      })();
    JS
  end

  private def select_multiple_lines_with_newline(start_line, end_line)
    # 複数行を改行文字を含めて選択するJavaScript
    page.execute_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;
        var doc = editorView.state.doc;
        var startLine = doc.line(#{start_line});
        var endLine = doc.line(#{end_line});

        // 開始行の先頭から終了行の次の行の先頭まで選択
        var nextLineStart = #{end_line} < doc.lines
          ? doc.line(#{end_line} + 1).from
          : doc.length;

        editorView.dispatch({
          selection: { anchor: startLine.from, head: nextLineStart }
        });

        editorView.focus();
      })();
    JS
  end

  private def get_cursor_position
    # カーソル位置を取得するJavaScript
    position = page.evaluate_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;
        var cursor = editorView.state.selection.main.head;
        var line = editorView.state.doc.lineAt(cursor);

        return {
          line: line.number,
          column: cursor - line.from,
          absolutePosition: cursor
        };
      })();
    JS

    {
      line: position["line"],
      column: position["column"],
      absolute_position: position["absolutePosition"]
    }
  end

  private def get_selection_info
    # 選択範囲の情報を取得するJavaScript
    selection_info = page.evaluate_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;
        var selection = editorView.state.selection.main;
        var fromLine = editorView.state.doc.lineAt(selection.from);
        var toLine = editorView.state.doc.lineAt(selection.to);

        return {
          from: selection.from,
          to: selection.to,
          fromLine: fromLine.number,
          toLine: toLine.number,
          fromColumn: selection.from - fromLine.from,
          toColumn: selection.to - toLine.from,
          hasSelection: selection.from !== selection.to
        };
      })();
    JS

    {
      from: selection_info["from"],
      to: selection_info["to"],
      from_line: selection_info["fromLine"],
      to_line: selection_info["toLine"],
      from_column: selection_info["fromColumn"],
      to_column: selection_info["toColumn"],
      has_selection: selection_info["hasSelection"]
    }
  end

  private def set_cursor_position(line:, column:)
    # カーソル位置を設定するJavaScript
    page.execute_script(<<~JS)
      (function() {
        var editor = document.querySelector('.cm-content');
        var editorView = editor.cmView.view;
        var doc = editorView.state.doc;
        var lineInfo = doc.line(#{line});
        var position = lineInfo.from + #{column};

        editorView.dispatch({
          selection: { anchor: position, head: position }
        });
        editorView.focus();
      })();
    JS
  end

  describe "ファイルアップロード機能" do
    it "ページ編集画面でファイルをアップロードできること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      # ファイルアップロードのモックを設定
      setup_file_upload_mocks

      visit edit_page_path(space_record.identifier, page_record.number)

      # ファイルをドロップでアップロード（JavaScriptイベントを使用）
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const file = new File(['test content'], 'test-image.png', { type: 'image/png' });

        const event = new CustomEvent('file-drop', {
          detail: {
            files: [file],
            position: 0
          },
          bubbles: true
        });
        editor.dispatchEvent(event);
      JS

      # アップロードが完了するまで待機
      expect(page).to have_content("![test-image.png](https://r2.example.com/test-image.png)")
    end

    it "複数のファイルを同時にアップロードできること", skip: "WebMockがActiveStorageと干渉するため、システムスペックでのファイルアップロードテストは実行できない" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      # 複数ファイルアップロードのモックを設定
      # プリサインURLのエンドポイントをモック（複数回呼ばれることを想定）
      stub_request(:post, %r{/s/test-space/attachments/presign})
        .to_return(
          status: 200,
          body: {
            upload_url: "https://r2.example.com/upload",
            file_key: "test-file-key",
            signed_id: "test-signed-id"
          }.to_json,
          headers: {"Content-Type" => "application/json"}
        )

      # R2へのアップロードをモック（複数回）
      stub_request(:put, "https://r2.example.com/upload")
        .to_return(status: 200)

      # アタッチメント作成のエンドポイントをモック（複数ファイル用）
      call_count = 0
      stub_request(:post, %r{/s/test-space/attachments})
        .to_return do |request|
          call_count += 1
          {
            status: 200,
            body: {
              id: "test-attachment-id-#{call_count}",
              url: "https://r2.example.com/test-image-#{call_count}.png"
            }.to_json,
            headers: {"Content-Type" => "application/json"}
          }
        end

      visit edit_page_path(space_record.identifier, page_record.number)

      # 複数ファイルをドロップでアップロード（JavaScriptイベントを使用）
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const file1 = new File(['test content 1'], 'test-image-1.png', { type: 'image/png' });
        const file2 = new File(['test content 2'], 'test-image-2.png', { type: 'image/png' });

        const event = new CustomEvent('file-drop', {
          detail: {
            files: [file1, file2],
            position: 0
          },
          bubbles: true
        });
        editor.dispatchEvent(event);
      JS

      # 両方のファイルがアップロードされることを確認
      expect(page).to have_content("![test-image-1.png](https://r2.example.com/test-image-1.png)")
      expect(page).to have_content("![test-image-2.png](https://r2.example.com/test-image-2.png)")
    end

    it "アップロード中にプレースホルダーが表示されること", skip: "WebMockがActiveStorageと干渉するため、システムスペックでのファイルアップロードテストは実行できない" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      # 遅延を含むモックを設定
      setup_file_upload_mocks(upload_delay: 0.5)

      visit edit_page_path(space_record.identifier, page_record.number)

      # ファイルをドロップでアップロード（JavaScriptイベントを使用）
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const file = new File(['test content'], 'test-image.png', { type: 'image/png' });

        const event = new CustomEvent('file-drop', {
          detail: {
            files: [file],
            position: 0
          },
          bubbles: true
        });
        editor.dispatchEvent(event);
      JS

      # プレースホルダーが表示されることを確認
      expect(page).to have_content("![アップロード中: test-image.png]()")

      # アップロード完了後、URLに置換されることを確認
      expect(page).to have_content("![test-image.png](https://r2.example.com/test-image.png)")
      expect(page).not_to have_content("![アップロード中: test-image.png]()")
    end
  end

  describe "ファイルのドラッグ&ドロップ" do
    it "エディタにファイルをドラッグ&ドロップできること", skip: "WebMockがActiveStorageと干渉するため、システムスペックでのファイルアップロードテストは実行できない" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      # ファイルアップロードのモックを設定
      setup_file_upload_mocks

      visit edit_page_path(space_record.identifier, page_record.number)

      # ドラッグ&ドロップをシミュレート
      # 注: Capybaraではファイルのドラッグ&ドロップを直接シミュレートできないため、
      # JavaScriptを使用してイベントを発火させる
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const dataTransfer = new DataTransfer();
        const file = new File(['test content'], 'test-image.png', { type: 'image/png' });
        dataTransfer.items.add(file);

        // dragenterイベント
        const dragEnterEvent = new DragEvent('dragenter', {
          dataTransfer: dataTransfer,
          bubbles: true,
          cancelable: true
        });
        editor.dispatchEvent(dragEnterEvent);

        // dragoverイベント
        const dragOverEvent = new DragEvent('dragover', {
          dataTransfer: dataTransfer,
          bubbles: true,
          cancelable: true
        });
        editor.dispatchEvent(dragOverEvent);

        // dropイベント
        const dropEvent = new DragEvent('drop', {
          dataTransfer: dataTransfer,
          bubbles: true,
          cancelable: true
        });
        editor.dispatchEvent(dropEvent);
      JS

      # アップロードが完了してURLが挿入されることを確認
      expect(page).to have_content("![test-image.png](https://r2.example.com/test-image.png)")
    end

    it "ドラッグ中にドロップゾーンが表示されること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      visit edit_page_path(space_record.identifier, page_record.number)

      # dragenterイベントをシミュレート
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const dataTransfer = new DataTransfer();
        const file = new File(['test content'], 'test-image.png', { type: 'image/png' });
        dataTransfer.items.add(file);

        const dragEnterEvent = new DragEvent('dragenter', {
          dataTransfer: dataTransfer,
          bubbles: true,
          cancelable: true
        });
        editor.dispatchEvent(dragEnterEvent);
      JS

      # ドロップゾーンが表示されることを確認
      expect(page).to have_css(".cm-drop-zone")
      expect(page).to have_content("ファイルをドロップしてアップロード")

      # dragleaveイベントをシミュレート
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const dragLeaveEvent = new DragEvent('dragleave', {
          bubbles: true,
          cancelable: true
        });
        editor.dispatchEvent(dragLeaveEvent);
      JS

      # ドロップゾーンが非表示になることを確認
      expect(page).not_to have_css(".cm-drop-zone")
    end

    it "画像をペーストできること", skip: "WebMockがActiveStorageと干渉するため、システムスペックでのファイルアップロードテストは実行できない" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      # ファイルアップロードのモックを設定
      # ペースト用のモック設定
      blob = instance_double(ActiveStorage::Blob, signed_id: "test-signed-id")
      active_storage_attachment = instance_double(
        ActiveStorage::Attachment,
        blob: blob
      )
      attachment_record = instance_double(
        AttachmentRecord,
        id: "test-attachment-id",
        active_storage_attachment_record: active_storage_attachment
      )
      allow(attachment_record).to receive(:generate_signed_url).and_return("https://r2.example.com/pasted-image.png")
      result = instance_double(Attachments::CreateService::Result)
      allow(result).to receive(:attachment_record).and_return(attachment_record)
      setup_file_upload_mocks(
        create_result: result
      )

      visit edit_page_path(space_record.identifier, page_record.number)

      # エディタにフォーカス
      find(".cm-editor").click

      # クリップボードからの画像ペーストをシミュレート
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-content');
        const blob = new Blob(['test image data'], { type: 'image/png' });
        const file = new File([blob], 'pasted-image.png', { type: 'image/png' });

        const clipboardData = new DataTransfer();
        clipboardData.items.add(file);

        const pasteEvent = new ClipboardEvent('paste', {
          clipboardData: clipboardData,
          bubbles: true,
          cancelable: true
        });
        editor.dispatchEvent(pasteEvent);
      JS

      # ペーストした画像がアップロードされることを確認
      expect(page).to have_content("![pasted-image.png](https://r2.example.com/pasted-image.png)")
    end
  end

  describe "ファイルアップロードのエラーハンドリング" do
    it "ファイルサイズが制限を超えている場合、エラーメッセージが表示されること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      visit edit_page_path(space_record.identifier, page_record.number)

      # 大きなファイルのアップロードをシミュレート（11MB）
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const largeContent = new Array(11 * 1024 * 1024).fill('a').join('');
        const file = new File([largeContent], 'large-image.png', { type: 'image/png' });

        const event = new CustomEvent('file-drop', {
          detail: {
            files: [file],
            position: 0
          },
          bubbles: true
        });
        editor.dispatchEvent(event);
      JS

      # エラーメッセージが表示されることを確認
      expect(page).to have_content("ファイルサイズが制限（10MB）を超えています")
    end

    it "サポートされていないファイル形式の場合、エラーメッセージが表示されること" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      visit edit_page_path(space_record.identifier, page_record.number)

      # サポートされていないファイル形式のアップロードをシミュレート
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const file = new File(['executable content'], 'malicious.exe', { type: 'application/x-msdownload' });

        const event = new CustomEvent('file-drop', {
          detail: {
            files: [file],
            position: 0
          },
          bubbles: true
        });
        editor.dispatchEvent(event);
      JS

      # エラーメッセージが表示されることを確認
      expect(page).to have_content("このファイル形式はアップロードできません")
    end

    it "ネットワークエラーの場合、リトライが行われること", skip: "WebMockがActiveStorageと干渉するため、システムスペックでのファイルアップロードテストは実行できない" do
      user_record = create(:user_record, :with_password)
      space_record = create(:space_record, identifier: "test-space")
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, :published, space_record:, topic_record:)
      sign_in(user_record:)

      # ネットワークエラーをシミュレートするモックを設定
      setup_file_upload_mocks(upload_error_count: 2)

      visit edit_page_path(space_record.identifier, page_record.number)

      # ファイルアップロードをシミュレート
      page.execute_script(<<~JS)
        const editor = document.querySelector('.cm-editor');
        const file = new File(['test content'], 'test-image.png', { type: 'image/png' });

        const event = new CustomEvent('file-drop', {
          detail: {
            files: [file],
            position: 0
          },
          bubbles: true
        });
        editor.dispatchEvent(event);
      JS

      # リトライの後、最終的にアップロードが成功することを確認
      expect(page).to have_content("![test-image.png](https://r2.example.com/test-image.png)", wait: 10)
    end
  end
end
