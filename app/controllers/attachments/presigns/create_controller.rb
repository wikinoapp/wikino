# typed: strict
# frozen_string_literal: true

module Attachments
  module Presigns
    class CreateController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { void }
      def call
        space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
        space_member_record = current_user_record!.space_member_record(space_record:)
        policy = SpaceMemberPolicy.new(
          user_record: current_user_record!,
          space_member_record:
        )

        unless policy.joined_space?
          render json: {error: "Unauthorized"}, status: :forbidden
        end

        form = Attachments::PresignForm.new(
          filename: params[:filename],
          content_type: params[:content_type],
          byte_size: params[:byte_size]
        )

        if form.invalid?
          return render(json: {errors: form.errors.full_messages}, status: :unprocessable_entity)
        end

        # 署名付きURLの生成
        blob = ActiveStorage::Blob.create_before_direct_upload!(
          filename: form.filename,
          content_type: form.content_type,
          byte_size: form.byte_size,
          checksum: OpenSSL::Digest::MD5.base64digest(form.filename),  # 一時的なチェックサム
          metadata: {
            space_id: space_record.id,
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
      end
    end
  end
end
