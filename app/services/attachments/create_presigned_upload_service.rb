# typed: strict
# frozen_string_literal: true

module Attachments
  class CreatePresignedUploadService < ApplicationService
    class Result < T::Struct
      const :direct_upload_url, String
      const :direct_upload_headers, T::Hash[String, String]
      const :blob_signed_id, String
      const :attachment_id, T::Wikino::DatabaseId
    end

    sig do
      params(
        filename: String,
        content_type: String,
        byte_size: Integer,
        checksum: String,
        space_record: SpaceRecord,
        user_record: UserRecord
      ).returns(Result)
    end
    def call(filename:, content_type:, byte_size:, checksum:, space_record:, user_record:)
      blob = create_blob!(
        filename:,
        content_type:,
        byte_size:,
        checksum:,
        space_record:,
        user_record:
      )

      # 添付ファイルレコードを作成
      space_member_record = user_record.space_member_record(space_record:)
      attachment_result = Attachments::CreateService.new.call(
        space_record:,
        blob_record: blob,
        attached_space_member_id: space_member_record.not_nil!.id
      )

      Result.new(
        direct_upload_url: blob.service_url_for_direct_upload,
        direct_upload_headers: blob.service_headers_for_direct_upload,
        blob_signed_id: blob.signed_id,
        attachment_id: attachment_result.attachment_record.id
      )
    end

    sig do
      params(
        filename: String,
        content_type: String,
        byte_size: Integer,
        checksum: String,
        space_record: SpaceRecord,
        user_record: UserRecord
      ).returns(ActiveStorage::Blob)
    end
    private def create_blob!(filename:, content_type:, byte_size:, checksum:, space_record:, user_record:)
      ActiveStorage::Blob.create_before_direct_upload!(
        filename:,
        content_type:,
        byte_size:,
        # クライアントから提供されたMD5チェックサムを使用
        checksum:,
        metadata: {
          space_id: space_record.id,
          user_id: user_record.id
        }
      )
    end
  end
end
