# typed: strict
# frozen_string_literal: true

module Attachments
  class CreatePresignedUploadService < ApplicationService
    extend T::Sig

    class Result < T::Struct
      const :direct_upload_url, String
      const :direct_upload_headers, T::Hash[String, String]
      const :blob_signed_id, String
    end

    sig do
      params(
        filename: String,
        content_type: String,
        byte_size: Integer,
        space_record: SpaceRecord,
        user_record: UserRecord
      ).returns(Result)
    end
    def call(filename:, content_type:, byte_size:, space_record:, user_record:)
      blob = create_blob!(
        filename:,
        content_type:,
        byte_size:,
        space_record:,
        user_record:
      )

      Result.new(
        direct_upload_url: blob.service_url_for_direct_upload,
        direct_upload_headers: blob.service_headers_for_direct_upload,
        blob_signed_id: blob.signed_id
      )
    end

    private

    sig do
      params(
        filename: String,
        content_type: String,
        byte_size: Integer,
        space_record: SpaceRecord,
        user_record: UserRecord
      ).returns(ActiveStorage::Blob)
    end
    private def create_blob!(filename:, content_type:, byte_size:, space_record:, user_record:)
      ActiveStorage::Blob.create_before_direct_upload!(
        filename:,
        content_type:,
        byte_size:,
        checksum: OpenSSL::Digest::MD5.base64digest(filename), # 一時的なチェックサム
        metadata: {
          space_id: space_record.id,
          user_id: user_record.id
        }
      )
    end
  end
end
