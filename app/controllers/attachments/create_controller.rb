# typed: true
# frozen_string_literal: true

module Attachments
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { void }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record!.space_member_record(space_record:)
      policy = SpacePolicyFactory.build(
        user_record: current_user_record!,
        space_member_record:
      )

      unless policy.can_upload_attachment?(space_record:)
        return render(json: {error: "Unauthorized"}, status: :forbidden)
      end

      form = Attachments::CreationForm.new(blob_signed_id: params[:blob_signed_id])

      if form.invalid?
        return render(json: {errors: form.errors.full_messages}, status: :unprocessable_entity)
      end

      result = Attachments::CreateService.new.call(
        space_record:,
        blob_record: form.blob_record.not_nil!,
        attached_space_member_id: space_member_record.not_nil!.id
      )

      # 署名付きURLを生成
      url = result.attachment_record.generate_signed_url(space_member_record:)

      attachment = AttachmentRepository.new.to_model(
        attachment_record: result.attachment_record,
        url:
      )

      render json: {
        id: attachment.database_id,
        filename: attachment.filename,
        content_type: attachment.content_type,
        byte_size: attachment.byte_size,
        url: attachment.url
      }, status: :created
    end
  end
end
