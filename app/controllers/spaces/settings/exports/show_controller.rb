# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      class ShowController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_viewer = Current.viewer!.space_viewer!(space:)
          space_entity = space.to_entity(space_viewer:)

          unless space_entity.viewer_can_export?
            return render_404
          end

          export = space.export_records.find(params[:export_id])

          render Spaces::Settings::Exports::ShowView.new(
            current_user: current_user!,
            space_entity:,
            export_entity: export.to_entity(space_viewer:),
            export_status_entity: export.latest_status_record.not_nil!.to_entity(space_viewer:)
          )
        end
      end
    end
  end
end
