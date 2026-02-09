# typed: strict
# frozen_string_literal: true

module Attachments
  class CreateService < ApplicationService
    class Result < T::Struct
      const :attachment_record, AttachmentRecord
    end

    sig do
      params(
        space_record: SpaceRecord,
        blob_record: ActiveStorage::Blob,
        attached_space_member_id: Types::DatabaseId
      ).returns(Result)
    end
    def call(space_record:, blob_record:, attached_space_member_id:)
      attachment_record = ActiveRecord::Base.transaction do
        # ActiveStorage::AttachmentをSpaceRecordに関連付けて作成
        active_storage_attachment_record = ActiveStorage::Attachment.create!(
          name: "file",
          record: space_record,
          blob: blob_record
        )

        created_attachment_record = AttachmentRecord.create!(
          space_id: space_record.id,
          active_storage_attachment_record:,
          attached_space_member_id:,
          attached_at: Time.current
        )

        created_attachment_record
      end

      AttachmentProcessingJob.perform_later(attachment_record.id)

      Result.new(attachment_record:)
    end
  end
end
