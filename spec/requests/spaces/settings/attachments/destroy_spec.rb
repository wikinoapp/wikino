# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe "DELETE /s/:space_identifier/settings/attachments/:attachment_id", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトされること" do
    space_record = FactoryBot.create(:space_record)
    space_member = FactoryBot.create(:space_member_record, space_record:)

    # ActiveStorageのテーブルに直接データを作成
    blob = ActiveStorage::Blob.create!(
      key: SecureRandom.uuid,
      filename: "test.txt",
      content_type: "text/plain",
      byte_size: 1024,
      checksum: Digest::MD5.base64digest("test content")
    )

    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record_type: "SpaceRecord",
      record_id: space_record.id,
      blob:
    )

    attachment_record = AttachmentRecord.create!(
      space_record:,
      attached_space_member_record: space_member,
      active_storage_attachment_record: active_storage_attachment,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "スペースメンバーでない場合、404エラーになること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    sign_in(user_record:)

    space_record = FactoryBot.create(:space_record)
    space_member = FactoryBot.create(:space_member_record, space_record:)

    # ActiveStorageのテーブルに直接データを作成
    blob = ActiveStorage::Blob.create!(
      key: SecureRandom.uuid,
      filename: "test.txt",
      content_type: "text/plain",
      byte_size: 1024,
      checksum: Digest::MD5.base64digest("test content")
    )

    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record_type: "SpaceRecord",
      record_id: space_record.id,
      blob:
    )

    attachment_record = AttachmentRecord.create!(
      space_record:,
      attached_space_member_record: space_member,
      active_storage_attachment_record: active_storage_attachment,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(404)
  end

  it "添付ファイルの削除権限がない場合、404エラーになること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    sign_in(user_record:)

    space_record = FactoryBot.create(:space_record)
    FactoryBot.create(:space_member_record, :member, space_record:, user_record:)

    # 他のメンバーがアップロードしたファイル
    other_member = FactoryBot.create(:space_member_record, space_record:)

    # ActiveStorageのテーブルに直接データを作成
    blob = ActiveStorage::Blob.create!(
      key: SecureRandom.uuid,
      filename: "test.txt",
      content_type: "text/plain",
      byte_size: 1024,
      checksum: Digest::MD5.base64digest("test content")
    )

    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record_type: "SpaceRecord",
      record_id: space_record.id,
      blob:
    )

    attachment_record = AttachmentRecord.create!(
      space_record:,
      attached_space_member_record: other_member,
      active_storage_attachment_record: active_storage_attachment,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(404)
  end

  it "添付ファイルを削除できること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    sign_in(user_record:)

    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, :owner, space_record:, user_record:)

    # ActiveStorageのテーブルに直接データを作成
    blob = ActiveStorage::Blob.create!(
      key: SecureRandom.uuid,
      filename: "test.txt",
      content_type: "text/plain",
      byte_size: 1024,
      checksum: Digest::MD5.base64digest("test content")
    )

    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record_type: "SpaceRecord",
      record_id: space_record.id,
      blob:
    )

    attachment_record = AttachmentRecord.create!(
      space_record:,
      attached_space_member_record: space_member_record,
      active_storage_attachment_record: active_storage_attachment,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    # 削除サービスがモック化されるように
    delete_service = instance_double(Attachments::DeleteService)
    allow(Attachments::DeleteService).to receive(:new).and_return(delete_service)
    allow(delete_service).to receive(:call).with(attachment_record_id: attachment_record.id)

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space_record.identifier}/settings/attachments")
    expect(flash[:notice]).to eq(I18n.t("messages.attachments.deleted_successfully"))
    expect(delete_service).to have_received(:call).with(attachment_record_id: attachment_record.id)
  end

  it "自分がアップロードした添付ファイルは一般メンバーでも削除できること" do
    user_record = FactoryBot.create(:user_record, :with_password)
    sign_in(user_record:)

    space_record = FactoryBot.create(:space_record)
    space_member_record = FactoryBot.create(:space_member_record, :member, space_record:, user_record:)

    # 自分がアップロードしたファイル
    # ActiveStorageのテーブルに直接データを作成
    blob = ActiveStorage::Blob.create!(
      key: SecureRandom.uuid,
      filename: "test.txt",
      content_type: "text/plain",
      byte_size: 1024,
      checksum: Digest::MD5.base64digest("test content")
    )

    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record_type: "SpaceRecord",
      record_id: space_record.id,
      blob:
    )

    attachment_record = AttachmentRecord.create!(
      space_record:,
      attached_space_member_record: space_member_record,
      active_storage_attachment_record: active_storage_attachment,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    # 削除サービスがモック化されるように
    delete_service = instance_double(Attachments::DeleteService)
    allow(Attachments::DeleteService).to receive(:new).and_return(delete_service)
    allow(delete_service).to receive(:call).with(attachment_record_id: attachment_record.id)

    delete("/s/#{space_record.identifier}/settings/attachments/#{attachment_record.id}")

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space_record.identifier}/settings/attachments")
    expect(delete_service).to have_received(:call).with(attachment_record_id: attachment_record.id)
  end
end
