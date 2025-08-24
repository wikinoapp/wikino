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

          attachment_record = space_record.attachment_records.find(params[:attachment_id])

          unless space_member_policy.can_delete_attachment?(attachment_record:)
            return render_404
          end

          ::Attachments::DeleteService.new.call(attachment_record_id: attachment_record.id)

          flash[:notice] = t("messages.attachments.deleted_successfully")
          redirect_to space_settings_attachments_path(space_record.identifier)
        end
      end
    end
  end
end
