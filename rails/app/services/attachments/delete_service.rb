# typed: strict
# frozen_string_literal: true

module Attachments
  class DeleteService < ApplicationService
    sig { params(attachment_record_id: Types::DatabaseId).void }
    def call(attachment_record_id:)
      attachment_record = AttachmentRecord.find(attachment_record_id)

      with_transaction do
        PageAttachmentReferenceRecord.where(attachment_id: attachment_record.id).destroy_all

        # Active StorageのAttachmentを保持
        active_storage_attachment = attachment_record.active_storage_attachment_record

        # AttachmentRecordを削除
        attachment_record.destroy!

        # Active StorageのAttachmentとBlobを削除
        active_storage_attachment&.purge
      end

      nil
    end
  end
end
