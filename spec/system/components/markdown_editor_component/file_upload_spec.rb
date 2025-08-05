# typed: false
# frozen_string_literal: true

require_relative "shared_helpers"

RSpec.describe "Markdownエディター/ファイルアップロード機能", type: :system do
  include MarkdownEditorHelpers

  describe "基本的なファイルアップロード" do
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
