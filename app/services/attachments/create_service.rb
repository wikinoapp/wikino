# typed: strict
# frozen_string_literal: true

module Attachments
  class CreateService < ApplicationService
    class Result < T::Struct
      const :attachment_record, AttachmentRecord
    end

    sig { params(space_record: SpaceRecord, blob: ActiveStorage::Blob, user_id: String).returns(Result) }
    def call(space_id:, blob:, user_id:)
      ActiveRecord::Base.transaction do
        active_storage_attachment = ActiveStorage::Attachment.create!(
          name: "file",
          record_type: "AttachmentRecord",
          record_id: SecureRandom.uuid, # 一時的なID
          blob:
        )

        attachment_record = AttachmentRecord.create!(
          space_id: space_record.id,
          active_storage_attachment_id: active_storage_attachment.id,
          attached_user_id:,
          attached_at: Time.current
        )

        active_storage_attachment.update!(record_id: attachment_record.id)

        Result.new(attachment_record:)
      end
    end
  end
end
