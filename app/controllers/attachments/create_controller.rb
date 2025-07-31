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
      # スペースの権限確認
      authorize_space_member!

      # フォームオブジェクトで検証
      form = Attachments::CreationForm.new(blob_signed_id: params[:blob_signed_id])
      unless form.valid?
        render json: {errors: form.errors.full_messages}, status: :unprocessable_entity
        return
      end

      # Attachmentレコードの作成
      attachment_record = AttachmentRecord.transaction do
        # ActiveStorageのアタッチメントを作成
        active_storage_attachment = ActiveStorage::Attachment.create!(
          name: "file",
          record_type: "AttachmentRecord",
          record_id: SecureRandom.uuid, # 一時的なID
          blob: form.blob
        )

        # Attachmentレコードを作成
        attachment = AttachmentRecord.create!(
          space_id: current_space.id,
          active_storage_attachment_id: active_storage_attachment.id,
          attached_user_id: current_user_record.not_nil!.id,
          attached_at: Time.current
        )

        # ActiveStorageのアタッチメントを更新
        active_storage_attachment.update!(record_id: attachment.id)

        attachment
      end

      # Attachmentモデルに変換
      attachment = AttachmentRepository.new.to_model(
        attachment_record: attachment_record,
        url: attachment_url(attachment_record)
      )

      render json: {
        id: attachment.id,
        filename: attachment.filename,
        content_type: attachment.content_type,
        byte_size: attachment.byte_size,
        url: attachment.url
      }, status: :created
    rescue ActiveRecord::RecordInvalid => e
      render json: {error: e.message}, status: :unprocessable_entity
    end

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

    # 添付ファイルのURLを生成
    sig { params(attachment_record: AttachmentRecord).returns(String) }
    private def attachment_url(attachment_record)
      # 署名付きURLを生成（1時間有効）
      blob = attachment_record.blob.not_nil!
      blob.url(expires_in: 1.hour)
    end
  end
end
