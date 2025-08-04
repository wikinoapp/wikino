# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "File Upload", type: :system do
  it "ページ編集画面でファイルをアップロードできること" do
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

    # ファイルを選択してアップロード
    file_path = Rails.root.join("spec/fixtures/files/test-image.png")
    attach_file("file-input", file_path, make_visible: true)

    # アップロードが完了するまで待機
    expect(page).to have_content("![test-image.png](https://r2.example.com/test-image.png)")
  end

  it "複数のファイルを同時にアップロードできること" do
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

    # 複数ファイルを選択してアップロード
    file_path1 = Rails.root.join("spec/fixtures/files/test-image-1.png")
    file_path2 = Rails.root.join("spec/fixtures/files/test-image-2.png")
    attach_file("file-input", [file_path1, file_path2], make_visible: true, multiple: true)

    # 両方のファイルがアップロードされることを確認
    expect(page).to have_content("![test-image-1.png](https://r2.example.com/test-image-1.png)")
    expect(page).to have_content("![test-image-2.png](https://r2.example.com/test-image-2.png)")
  end

  it "アップロード中にプレースホルダーが表示されること" do
    user_record = create(:user_record, :with_password)
    space_record = create(:space_record, identifier: "test-space")
    topic_record = create(:topic_record, space_record:)
    space_member_record = create(:space_member_record, user_record:, space_record:)
    create(:topic_member_record, space_record:, topic_record:, space_member_record:)
    page_record = create(:page_record, :published, space_record:, topic_record:)
    sign_in(user_record:)

    # 遅延を含むモックを設定
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

    # アップロードに遅延を設定
    stub_request(:put, "https://r2.example.com/upload")
      .to_return(status: 200) do |request|
        sleep 0.5 # 遅延をシミュレート
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

    # ファイルを選択
    file_path = Rails.root.join("spec/fixtures/files/test-image.png")
    attach_file("file-input", file_path, make_visible: true)

    # プレースホルダーが表示されることを確認
    expect(page).to have_content("![アップロード中: test-image.png]()")

    # アップロード完了後、URLに置換されることを確認
    expect(page).to have_content("![test-image.png](https://r2.example.com/test-image.png)")
    expect(page).not_to have_content("![アップロード中: test-image.png]()")
  end
end