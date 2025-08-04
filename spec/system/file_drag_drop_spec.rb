# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "File Drag and Drop", type: :system do
  it "エディタにファイルをドラッグ&ドロップできること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # ファイルアップロードのモックを設定
    stub_request(:post, %r{/s/test-space/attachments/presign})
      .to_return(
        status: 200,
        body: {
          upload_url: "https://r2.example.com/upload",
          file_key: "test-file-key",
          signed_id: "test-signed-id"
        }.to_json,
        headers: { "Content-Type" => "application/json" }
      )

    stub_request(:put, "https://r2.example.com/upload")
      .to_return(status: 200)

    stub_request(:post, %r{/s/test-space/attachments})
      .to_return(
        status: 200,
        body: {
          id: "test-attachment-id",
          url: "https://r2.example.com/test-image.png"
        }.to_json,
        headers: { "Content-Type" => "application/json" }
      )

    visit edit_page_path(space_record.identifier, page_record.number)

    # エディタ要素を取得
    editor = find(".cm-editor")

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

    # エディタ要素を取得
    editor = find(".cm-editor")

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

  it "複数ファイルを同時にドラッグ&ドロップできること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # 複数ファイルアップロードのモックを設定
    stub_request(:post, %r{/s/test-space/attachments/presign})
      .to_return(
        status: 200,
        body: {
          upload_url: "https://r2.example.com/upload",
          file_key: "test-file-key",
          signed_id: "test-signed-id"
        }.to_json,
        headers: { "Content-Type" => "application/json" }
      )

    stub_request(:put, "https://r2.example.com/upload")
      .to_return(status: 200)

    stub_request(:post, %r{/s/test-space/attachments})
      .to_return(
        { status: 200,
          body: {
            id: "test-attachment-id-1",
            url: "https://r2.example.com/test-image-1.png"
          }.to_json,
          headers: { "Content-Type" => "application/json" } },
        { status: 200,
          body: {
            id: "test-attachment-id-2",
            url: "https://r2.example.com/test-image-2.png"
          }.to_json,
          headers: { "Content-Type" => "application/json" } }
      )

    visit edit_page_path(space_record.identifier, page_record.number)

    # 複数ファイルのドラッグ&ドロップをシミュレート
    page.execute_script(<<~JS)
      const editor = document.querySelector('.cm-editor');
      const dataTransfer = new DataTransfer();
      const file1 = new File(['test content 1'], 'test-image-1.png', { type: 'image/png' });
      const file2 = new File(['test content 2'], 'test-image-2.png', { type: 'image/png' });
      dataTransfer.items.add(file1);
      dataTransfer.items.add(file2);
      
      const dropEvent = new DragEvent('drop', {
        dataTransfer: dataTransfer,
        bubbles: true,
        cancelable: true
      });
      editor.dispatchEvent(dropEvent);
    JS

    # 両方のファイルがアップロードされることを確認
    expect(page).to have_content("![test-image-1.png](https://r2.example.com/test-image-1.png)")
    expect(page).to have_content("![test-image-2.png](https://r2.example.com/test-image-2.png)")
  end

  it "画像をペーストできること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # ファイルアップロードのモックを設定
    stub_request(:post, %r{/s/test-space/attachments/presign})
      .to_return(
        status: 200,
        body: {
          upload_url: "https://r2.example.com/upload",
          file_key: "test-file-key",
          signed_id: "test-signed-id"
        }.to_json,
        headers: { "Content-Type" => "application/json" }
      )

    stub_request(:put, "https://r2.example.com/upload")
      .to_return(status: 200)

    stub_request(:post, %r{/s/test-space/attachments})
      .to_return(
        status: 200,
        body: {
          id: "test-attachment-id",
          url: "https://r2.example.com/pasted-image.png"
        }.to_json,
        headers: { "Content-Type" => "application/json" }
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