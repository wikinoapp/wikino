# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :attachment_record do
    space_record
    attached_space_member_record { association :space_member_record, space_record: }
    attached_at { Time.current }
    processing_status { AttachmentProcessingStatus::Completed.serialize }

    # デフォルトでActive Storage Attachmentを作成
    after(:build) do |attachment_record|
      # 一時的なダミーレコードを作成してActive Storage Attachmentを生成
      temp_attachment = AttachmentRecord.new(id: SecureRandom.uuid)

      blob = ActiveStorage::Blob.create_and_upload!(
        io: StringIO.new("test file content"),
        filename: "test-file.txt",
        content_type: "text/plain"
      )

      active_storage_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: temp_attachment,
        blob:
      )

      attachment_record.active_storage_attachment_record = active_storage_attachment
    end

    trait :with_blob do
      before(:create) do |attachment_record|
        # 一時的なダミーレコードを作成してActive Storage Attachmentを生成
        temp_attachment = AttachmentRecord.new(id: SecureRandom.uuid)

        blob = ActiveStorage::Blob.create_and_upload!(
          io: StringIO.new("test file content"),
          filename: "test-file.txt",
          content_type: "text/plain"
        )

        active_storage_attachment = ActiveStorage::Attachment.create!(
          name: "file",
          record: temp_attachment,
          blob:
        )

        attachment_record.active_storage_attachment_id = active_storage_attachment.id
      end
    end

    trait :processing do
      processing_status { AttachmentProcessingStatus::Processing.serialize }
    end

    trait :failed do
      processing_status { AttachmentProcessingStatus::Failed.serialize }
    end

    trait :with_image do
      before(:create) do |attachment_record|
        # 一時的なダミーレコードを作成してActive Storage Attachmentを生成
        temp_attachment = AttachmentRecord.new(id: SecureRandom.uuid)

        # テスト用の小さな画像を作成
        file = StringIO.new(Base64.decode64("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="))

        blob = ActiveStorage::Blob.create_and_upload!(
          io: file,
          filename: "test-image.png",
          content_type: "image/png"
        )

        active_storage_attachment = ActiveStorage::Attachment.create!(
          name: "file",
          record: temp_attachment,
          blob:
        )

        attachment_record.active_storage_attachment_id = active_storage_attachment.id
      end
    end

    trait :with_pdf do
      before(:create) do |attachment_record|
        # 一時的なダミーレコードを作成してActive Storage Attachmentを生成
        temp_attachment = AttachmentRecord.new(id: SecureRandom.uuid)

        blob = ActiveStorage::Blob.create_and_upload!(
          io: StringIO.new("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n"),
          filename: "test-document.pdf",
          content_type: "application/pdf"
        )

        active_storage_attachment = ActiveStorage::Attachment.create!(
          name: "file",
          record: temp_attachment,
          blob:
        )

        attachment_record.active_storage_attachment_id = active_storage_attachment.id
      end
    end
  end
end
