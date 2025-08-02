# typed: strict
# frozen_string_literal: true

module Attachments
  class GenerateSignedUrlService < ApplicationService
    class Result < T::Struct
      const :url, T.nilable(String)
      const :error, T.nilable(String)
    end

    sig do
      params(
        attachment_record: AttachmentRecord,
        space_member_record: T.nilable(SpaceMemberRecord),
        expires_in: ActiveSupport::Duration
      ).returns(Result)
    end
    def call(attachment_record:, space_member_record:, expires_in: 1.hour)
      # ポリシーを使用してアクセス権限を確認
      policy = SpaceMemberPolicy.new(
        user_record: space_member_record&.user_record,
        space_member_record:
      )

      unless policy.can_view_attachment?(attachment_record:)
        return Result.new(url: nil, error: "Unauthorized")
      end

      blob = attachment_record.blob_record
      unless blob
        return Result.new(url: nil, error: "Blob not found")
      end

      # 署名付きURLを生成
      url = blob.url(expires_in:)
      Result.new(url:, error: nil)
    rescue => e
      Rails.logger.error("Failed to generate signed URL: #{e.message}")
      Result.new(url: nil, error: "Failed to generate URL")
    end
  end
end
