# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      class CreateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_member_record = current_user_record!.space_member_record(space_record:)
          space_member_policy = SpaceMemberPolicy.new(
            user_record: current_user_record!,
            space_member_record:
          )

          unless space_member_policy.can_export_space?(space_record:)
            return render_404
          end

          result = Spaces::ExportService.new.call(
            space_record:,
            queued_by_record: space_member_record.not_nil!
          )

          flash[:notice] = t("messages.exports.started")
          redirect_to space_settings_export_path(space_record.identifier, result.export_record.id)
        end
      end
    end
  end
end
