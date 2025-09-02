# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      class CreateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable
        include ControllerConcerns::SpaceAware

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = current_space_record
          space_policy = space_policy_for(space_record:)

          unless space_policy.can_export_space?(space_record:)
            return render_404
          end

          space_member_record = current_space_member_record!(space_record:)
          result = Spaces::ExportService.new.call(
            space_record:,
            queued_by_record: space_member_record
          )

          flash[:notice] = t("messages.exports.started")
          redirect_to space_settings_export_path(space_record.identifier, result.export_record.id)
        end
      end
    end
  end
end
