# typed: strict
# frozen_string_literal: true

module Attachments
  class CreateService
    extend T::Sig

    sig { params(space_id: String, blob: ActiveStorage::Blob, user_id: String).void }
    def initialize(space_id:, blob:, user_id:)
      @space_id = space_id
      @blob = blob
      @user_id = user_id
    end

    sig { returns(AttachmentRecord) }
    def call
      AttachmentRecord.transaction do
        # ActiveStorageのアタッチメントを作成
        active_storage_attachment = create_active_storage_attachment

        # Attachmentレコードを作成
        attachment_record = create_attachment_record(active_storage_attachment)

        # ActiveStorageのアタッチメントを更新
        update_active_storage_attachment(active_storage_attachment, attachment_record)

        attachment_record
      end
    end

    private

    sig { returns(ActiveStorage::Attachment) }
    private def create_active_storage_attachment
      ActiveStorage::Attachment.create!(
        name: "file",
        record_type: "AttachmentRecord",
        record_id: SecureRandom.uuid, # 一時的なID
        blob: @blob
      )
    end

    sig { params(active_storage_attachment: ActiveStorage::Attachment).returns(AttachmentRecord) }
    private def create_attachment_record(active_storage_attachment)
      AttachmentRecord.create!(
        space_id: @space_id,
        active_storage_attachment_id: active_storage_attachment.id,
        attached_user_id: @user_id,
        attached_at: Time.current
      )
    end

    sig { params(active_storage_attachment: ActiveStorage::Attachment, attachment_record: AttachmentRecord).void }
    private def update_active_storage_attachment(active_storage_attachment, attachment_record)
      active_storage_attachment.update!(record_id: attachment_record.id)
    end
  end
end
