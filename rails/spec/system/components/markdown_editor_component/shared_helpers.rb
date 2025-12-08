# typed: false
# frozen_string_literal: true

module MarkdownEditorHelpers
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
      blob_signed_id: "test-signed-id",
      attachment_id: "test-attachment-id"
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

  private def visit_page_editor
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record)
    space_member_record = create(:space_member_record, space_record:, user_record:)
    topic_record = create(:topic_record, space_record:)
    page_record = create(:page_record, space_record:, topic_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)

    sign_in(user_record:)
    visit "/s/#{space_record.identifier}/pages/#{page_record.number}/edit"
  end
end
