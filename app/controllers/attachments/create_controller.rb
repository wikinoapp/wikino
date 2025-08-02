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
      policy = SpaceMemberPolicy.new(
        user_record: current_user_record!,
        space_member_record:
      )

      unless policy.joined_space?
        return render json: {error: "Unauthorized"}, status: :forbidden
      end

      form = Attachments::CreationForm.new(blob_signed_id: params[:blob_signed_id])

      if form.invalid?
        return render(json: {errors: form.errors.full_messages}, status: :unprocessable_entity)
      end

      result = Attachments::CreateService.new.call(
        space_record:,
        blob_record: form.blob.not_nil!,
        attached_space_member_id: space_member_record.not_nil!.id
      )

      attachment = AttachmentRepository.new.to_model(
        attachment_record: result.attachment_record,
        url: attachment_url(result.attachment_record)
      )

      render json: {
        id: attachment.id,
        filename: attachment.filename,
        content_type: attachment.content_type,
        byte_size: attachment.byte_size,
        url: attachment.url
      }, status: :created
    end

    # 添付ファイルのURLを生成
    sig { params(attachment_record: AttachmentRecord).returns(String) }
    private def attachment_url(attachment_record)
      # 署名付きURLを生成（1時間有効）
      blob = attachment_record.blob_record.not_nil!
      blob.url(expires_in: 1.hour)
    end
  end
end
