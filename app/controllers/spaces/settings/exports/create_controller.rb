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
          space = Space.find_by_identifier!(params[:space_identifier])
          space_viewer = Current.viewer!.space_viewer!(space:)
          space_entity = space.to_entity(space_viewer:)

          unless space_entity.viewer_can_export?
            return render_404
          end

          result = ExportService.new.call(
            space:,
            started_by: space_viewer,
            locale: current_locale
          )

          flash[:notice] = t("messages.exports.started")
          redirect_to space_settings_export_path(space.identifier, result.export.id)
        end
      end
    end
  end
end
