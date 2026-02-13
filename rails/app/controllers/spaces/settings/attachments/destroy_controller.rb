# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Attachments
      class DestroyController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable
        include ControllerConcerns::SpaceAware

        around_action :set_locale
        before_action :require_authentication

        sig { void }
        def call
          space_record = current_space_record
          space_policy = space_policy_for(space_record:)

          attachment_record = space_record.attachment_records.find(params[:attachment_id])

          unless space_policy.can_delete_attachment?(attachment_record:)
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
