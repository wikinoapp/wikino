# typed: strict
# frozen_string_literal: true

module Attachments
  module SignedUrls
    class CreateController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::SpaceAware

      before_action :restore_user_session

      # POST /attachments/signed_urls
      # 複数の添付ファイルIDに対する署名付きURLをバッチで生成
      sig { void }
      def call
        attachment_ids = params.fetch(:attachment_ids, [])

        # 空のレスポンスを返す
        if attachment_ids.empty?
          render json: {signed_urls: {}}
          return
        end

        # 各添付ファイルに対して署名付きURLを生成
        signed_urls = {}

        attachment_ids.each do |attachment_id|
          attachment_record = AttachmentRecord.find_by(id: attachment_id)
          next unless attachment_record

          space_record = attachment_record.space_record.not_nil!
          space_member_record = current_user_record&.space_member_record(space_record:)
          space_policy = space_policy_for(space_record:)

          # 権限チェック
          next unless space_policy.can_view_attachment?(attachment_record:)

          # 署名付きURLを生成
          signed_url = attachment_record.generate_signed_url(
            space_member_record:,
            expires_in: 1.hour
          )

          signed_urls[attachment_id] = signed_url if signed_url
        end

        render json: {signed_urls:}
      end
    end
  end
end
