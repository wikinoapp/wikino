# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "File Upload Error Handling", type: :system do
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

  it "ネットワークエラーの場合、リトライが行われること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # プリサイン成功のモック
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

    # アップロードエラーの後、成功するモック
    upload_attempt = 0
    stub_request(:put, "https://r2.example.com/upload")
      .to_return do |request|
        upload_attempt += 1
        if upload_attempt < 3
          { status: 500 } # エラーを返す
        else
          { status: 200 } # 3回目で成功
        end
      end

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

  it "プリサインURLの取得に失敗した場合、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # プリサインエラーのモック
    stub_request(:post, %r{/s/test-space/attachments/presign})
      .to_return(
        status: 401,
        body: { error: "認証エラー" }.to_json,
        headers: { "Content-Type" => "application/json" }
      )

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

    # エラーメッセージが表示されることを確認
    expect(page).to have_content("認証エラー")
  end

  it "アップロード完了通知に失敗した場合、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # プリサイン成功のモック
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

    # アップロード成功のモック
    stub_request(:put, "https://r2.example.com/upload")
      .to_return(status: 200)

    # 完了通知エラーのモック
    stub_request(:post, %r{/s/test-space/attachments})
      .to_return(
        status: 500,
        body: { error: "サーバーエラー" }.to_json,
        headers: { "Content-Type" => "application/json" }
      )

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

    # エラーメッセージが表示されることを確認
    expect(page).to have_content("サーバーエラー")
  end

  it "同時に複数のファイルアップロードでエラーが発生した場合、エラーメッセージが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    visit edit_page_path(space_record.identifier, page_record.number)

    # 複数のファイル（一部が無効）のアップロードをシミュレート
    page.execute_script(<<~JS)
      const editor = document.querySelector('.cm-editor');
      const validFile = new File(['test content'], 'valid.png', { type: 'image/png' });
      const invalidFile = new File(['executable'], 'invalid.exe', { type: 'application/x-msdownload' });
      const largeFile = new File([new Array(11 * 1024 * 1024).fill('a').join('')], 'large.png', { type: 'image/png' });
      
      const event = new CustomEvent('file-drop', {
        detail: {
          files: [validFile, invalidFile, largeFile],
          position: 0
        },
        bubbles: true
      });
      editor.dispatchEvent(event);
    JS

    # 各エラーメッセージが表示されることを確認
    expect(page).to have_content("このファイル形式はアップロードできません")
    expect(page).to have_content("ファイルサイズが制限（10MB）を超えています")
  end
end