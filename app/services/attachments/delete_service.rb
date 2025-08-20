# typed: strict
# frozen_string_literal: true

module Attachments
  class DeleteService < ApplicationService
    sig { params(attachment_record_id: T::Wikino::DatabaseId).void }
    def call(attachment_record_id:)
      attachment_record = AttachmentRecord.find(attachment_record_id)

      with_transaction do
        PageAttachmentReferenceRecord.where(attachment_id: attachment_record.id).destroy_all

        attachment_record.destroy!
      end

      nil
    end
  end
end
