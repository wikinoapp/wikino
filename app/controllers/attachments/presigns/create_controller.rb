# typed: strict
# frozen_string_literal: true

module Attachments
  module Presigns
    class CreateController < ApplicationController
      extend T::Sig
      include ControllerConcerns::Authenticatable

      before_action :require_authentication

      # ダイレクトアップロード用の署名付きURL生成
      # POST /s/:space_identifier/attachments/presign
      sig { void }
      def call
        # スペースの権限確認
        authorize_space_member!

        # ファイルのメタデータを検証
        form = AttachmentPresignForm.new(
          filename: params[:filename],
          content_type: params[:content_type],
          byte_size: params[:byte_size]
        )

        if form.valid?
          # 署名付きURLの生成
          blob = ActiveStorage::Blob.create_before_direct_upload!(
            filename: form.filename,
            content_type: form.content_type,
            byte_size: form.byte_size,
            checksum: OpenSSL::Digest::MD5.base64digest(form.filename),  # 一時的なチェックサム
            metadata: {
              space_id: current_space.id,
              user_id: current_user_record.not_nil!.id
            }
          )

          render json: {
            direct_upload: {
              url: blob.service_url_for_direct_upload,
              headers: blob.service_headers_for_direct_upload
            },
            blob_signed_id: blob.signed_id
          }
        else
          render json: {errors: form.errors.full_messages}, status: :unprocessable_entity
        end
      end

      private

      # 現在のスペースを取得
      sig { returns(SpaceRecord) }
      private def current_space
        @current_space ||= T.let(SpaceRecord.find_by!(identifier: params[:space_identifier]), T.nilable(SpaceRecord))
      end

      # スペースメンバーであることを確認
      sig { void }
      private def authorize_space_member!
        space_member_record = current_user_record&.space_member_record(space_record: current_space)
        policy = SpaceMemberPolicy.new(
          user_record: current_user_record,
          space_member_record: space_member_record
        )
        unless policy.joined_space?
          render json: {error: "Unauthorized"}, status: :forbidden
        end
      end
    end
  end
end
