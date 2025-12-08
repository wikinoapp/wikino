# typed: strict
# frozen_string_literal: true

module Attachments
  module Presigns
    class CreateController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable
      include ControllerConcerns::SpaceAware

      around_action :set_locale
      before_action :require_authentication

      sig { void }
      def call
        space_record = current_space_record
        space_policy = space_policy_for(space_record:)

        unless space_policy.can_upload_attachment?(space_record:)
          render json: {error: "Unauthorized"}, status: :forbidden
          return
        end

        form = Attachments::PresignForm.new(
          filename: params[:filename],
          content_type: params[:content_type],
          byte_size: params[:byte_size],
          checksum: params[:checksum]
        )

        if form.invalid?
          return render(json: {errors: form.errors.full_messages}, status: :unprocessable_entity)
        end

        # 署名付きURLの生成
        # フォームでサニタイズ済みのファイル名を使用
        result = Attachments::CreatePresignedUploadService.new.call(
          filename: form.filename.not_nil!,
          content_type: form.content_type.not_nil!,
          byte_size: form.byte_size.not_nil!,
          checksum: form.checksum.not_nil!,
          space_record:,
          user_record: current_user_record!
        )

        render json: {
          directUploadUrl: result.direct_upload_url,
          directUploadHeaders: result.direct_upload_headers,
          blobSignedId: result.blob_signed_id,
          attachmentId: result.attachment_id
        }
      end
    end
  end
end
