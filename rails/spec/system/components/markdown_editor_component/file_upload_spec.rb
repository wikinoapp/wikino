# typed: false
# frozen_string_literal: true

require_relative "shared_helpers"

RSpec.describe "Markdownエディター/ファイルアップロード機能", type: :system do
  include MarkdownEditorHelpers

  it "ページ編集画面でファイルをアップロードできること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # テスト用エンドポイントを使用するためモック不要

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
    # テスト用エンドポイントがattachmentIdを返す
    sleep 1  # アップロード処理が完了するまで待機

    # エディタ内容に画像のマークダウンが含まれることを確認
    editor_content = get_editor_content
    expect(editor_content).to include("/attachments/test-attachment")
  end

  it "複数のファイルを同時にアップロードできること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # テスト用エンドポイントを使用するためモック不要

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
    sleep 1  # アップロード処理が完了するまで待機
    editor_content = get_editor_content
    expect(editor_content).to include("test-image-1.png")
    expect(editor_content).to include("test-image-2.png")
    expect(editor_content).to include("/attachments/test-attachment")
  end

  it "エディタにファイルをドラッグ&ドロップできること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # テスト用エンドポイントを使用するためモック不要

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

    # アップロードが完了して画像が挿入されることを確認
    sleep 1  # アップロード処理が完了するまで待機
    editor_content = get_editor_content
    expect(editor_content).to include("/attachments/test-attachment")
  end

  it "ドラッグ中にドロップゾーンが表示されること", :js do
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

  it "画像をペーストできること", :js do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # テスト用エンドポイントを使用するためモック不要

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
    sleep 1  # アップロード処理が完了するまで待機
    editor_content = get_editor_content
    expect(editor_content).to include("pasted-image.png")
    expect(editor_content).to include("/attachments/test-attachment")
  end

  it "ファイルサイズが制限を超えている場合、エラーメッセージが表示されること", :js do
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

  it "サポートされていないファイル形式の場合、エラーメッセージが表示されること", :js do
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

  it "ネットワークエラーの場合、リトライが行われること", skip: "リトライ機能は現在未実装" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # テスト用エンドポイントを使用するためモック不要

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
    sleep 2
    editor_content = get_editor_content
    expect(editor_content).to include("/attachments/test-attachment")
  end
end
