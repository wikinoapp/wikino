# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "POST /s/:space_identifier/attachments/presign", type: :request do
  it "ログインしていない場合、サインインページにリダイレクトすること" do
    space_record = FactoryBot.create(:space_record)

    post attachment_presign_path(space_identifier: space_record.identifier),
      params: {
        filename: "test.png",
        content_type: "image/png",
        byte_size: 1024
      }

    expect(response).to redirect_to(sign_in_path)
  end

  it "スペースメンバーでない場合、403エラーを返すこと" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    sign_in(user_record:)

    post attachment_presign_path(space_identifier: space_record.identifier),
      params: {
        filename: "test.png",
        content_type: "image/png",
        byte_size: 1024
      }

    expect(response).to have_http_status(:forbidden)
  end

  it "バリデーションエラーの場合、422エラーを返すこと" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    FactoryBot.create(:space_member_record, space_record:, user_record:)
    sign_in(user_record:)

    post attachment_presign_path(space_identifier: space_record.identifier),
      params: {
        filename: "",
        content_type: "image/png",
        byte_size: 1024
      }

    expect(response).to have_http_status(:unprocessable_entity)
    json = JSON.parse(response.body)
    expect(json["errors"]).to be_present
  end

  # 実際のActiveStorageの動作はテスト環境でのセットアップが必要なため、
  # サービスクラスが呼び出されることだけを確認する
  it "正常なリクエストの場合、サービスクラスを呼び出すこと" do
    user_record = FactoryBot.create(:user_record, :with_password)
    space_record = FactoryBot.create(:space_record)
    FactoryBot.create(:space_member_record, space_record:, user_record:)
    sign_in(user_record:)

    # サービスクラスのモック
    service_double = instance_double(Attachments::CreatePresignedUploadService)
    result_double = instance_double(
      Attachments::CreatePresignedUploadService::Result,
      direct_upload_url: "https://example.com/upload",
      direct_upload_headers: {"Content-Type" => "image/png"},
      blob_signed_id: "test_signed_id"
    )
    allow(Attachments::CreatePresignedUploadService).to receive(:new).and_return(service_double)
    allow(service_double).to receive(:call).and_return(result_double)

    post attachment_presign_path(space_identifier: space_record.identifier),
      params: {
        filename: "test.png",
        content_type: "image/png",
        byte_size: 1024
      }

    expect(response).to have_http_status(:ok)
    expect(Attachments::CreatePresignedUploadService).to have_received(:new)
    expect(service_double).to have_received(:call).with(
      filename: "test.png",
      content_type: "image/png",
      byte_size: 1024,
      space_record:,
      user_record:
    )
  end
end
