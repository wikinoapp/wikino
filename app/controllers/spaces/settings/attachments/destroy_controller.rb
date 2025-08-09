# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Attachments
      class DestroyController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { void }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_member_record = current_user_record!.space_member_record(space_record:)
          space_member_policy = SpaceMemberPolicy.new(
            user_record: current_user_record!,
            space_member_record:
          )

          attachment_record = AttachmentRecord.find(params[:attachment_id])

          # スペースに属していることを確認
          unless attachment_record.space_id == space_record.id
            return render(json: {error: "Not found"}, status: :not_found)
          end

          unless space_member_policy.can_delete_attachment?(attachment_record:)
            return render(json: {error: "Unauthorized"}, status: :forbidden)
          end

          # 関連するページ内リンクを削除
          PageAttachmentReferenceRecord.where(attachment_id: attachment_record.id).destroy_all

          # 添付ファイルを削除
          attachment_record.destroy!

          render json: {message: "Attachment deleted successfully"}, status: :ok
        end
      end
    end
  end
end
