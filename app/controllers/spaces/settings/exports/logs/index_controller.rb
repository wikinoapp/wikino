# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      module Logs
        class IndexController < ApplicationController
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

            export = space.exports.find(params[:export_id])
            export_logs = export.logs.preload([:space]).order(logged_at: :desc)

            render Spaces::Settings::Exports::Logs::IndexView.new(
              current_user_entity: Current.viewer!.user_entity,
              space_entity:,
              export_log_entities: export_logs.map { |log| log.to_entity(space_viewer:) }
            )
          end
        end
      end
    end
  end
end
