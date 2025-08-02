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
      ActiveRecord::Base.transaction do
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

        # SVGファイルの場合はサニタイズ処理を実行
        attachment_record.sanitize_svg_content

        # 画像ファイルの場合はEXIF削除と自動回転を実行
        blob_record.process_image_with_exif_removal

        Result.new(attachment_record:)
      end
    end
  end
end
