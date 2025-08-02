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
        attached_space_member_id: T::Wikino::DatabaseId
      ).returns(Result)
    end
    def call(space_record:, blob_record:, attached_space_member_id:)
      attachment_record = ActiveRecord::Base.transaction do
        active_storage_attachment_record = ActiveStorage::Attachment.create!(
          name: "file",
          record_type: "AttachmentRecord",
          record_id: SecureRandom.uuid, # 一時的なID
          blob: blob_record
        )

        attachment_record = AttachmentRecord.create!(
          space_id: space_record.id,
          active_storage_attachment_record:,
          attached_space_member_id:,
          attached_at: Time.current
        )

        active_storage_attachment_record.update!(record: attachment_record)

        attachment_record
      end

      # トランザクション外で非同期ジョブをキュー
      AttachmentProcessingJob.perform_later(attachment_record.id)

      Result.new(attachment_record:)
    end
  end
end
