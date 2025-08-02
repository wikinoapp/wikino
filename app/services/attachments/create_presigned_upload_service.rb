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

    sig { params(filename: String, content_type: String, byte_size: Integer, space_record: SpaceRecord, user_record: UserRecord).void }
    def initialize(filename:, content_type:, byte_size:, space_record:, user_record:)
      @filename = filename
      @content_type = content_type
      @byte_size = byte_size
      @space_record = space_record
      @user_record = user_record
    end

    sig { returns(Result) }
    def call
      blob = create_blob!

      Result.new(
        direct_upload_url: blob.service_url_for_direct_upload,
        direct_upload_headers: blob.service_headers_for_direct_upload,
        blob_signed_id: blob.signed_id
      )
    end

    private

    sig { returns(ActiveStorage::Blob) }
    private def create_blob!
      ActiveStorage::Blob.create_before_direct_upload!(
        filename: @filename,
        content_type: @content_type,
        byte_size: @byte_size,
        checksum: OpenSSL::Digest::MD5.base64digest(@filename), # 一時的なチェックサム
        metadata: {
          space_id: @space_record.id,
          user_id: @user_record.id
        }
      )
    end
  end
end
