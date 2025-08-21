# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :attachment_record do
    space_record
    attached_space_member_record { association :space_member_record, space_record: }
    attached_at { Time.current }
    processing_status { AttachmentProcessingStatus::Completed.serialize }

    # Active Storage Attachment は作成時に設定する必要がある
    active_storage_attachment_id { nil }

    trait :with_blob do
      after(:create) do |attachment_record|
        blob = ActiveStorage::Blob.create_and_upload!(
          io: StringIO.new("test file content"),
          filename: "test-file.txt",
          content_type: "text/plain"
        )
        active_storage_attachment = ActiveStorage::Attachment.create!(
          name: "file",
          record_type: "AttachmentRecord",
          record_id: attachment_record.id,
          blob:
        )
        attachment_record.update!(active_storage_attachment_id: active_storage_attachment.id)
      end
    end

    trait :processing do
      processing_status { AttachmentProcessingStatus::Processing.serialize }
    end

    trait :failed do
      processing_status { AttachmentProcessingStatus::Failed.serialize }
    end
  end
end
